# 订单生命周期：下单 → 撮合 → 结算 → 上链

## 全链路概览

```
用户下单 (API)
    │
    ▼
OrderService                      Account Service
├─ 校验市场/价格/数量              ├─ PreFreeze 冻结保证金
├─ 计算冻结金额                    └─ freeze_records 表
└─ 发送 OrderCommand ──────────────────► Kafka: order.command
                                              │
                                              ▼
                                    InputGateway (ConsumerGroup)
                                    ├─ per-partition goroutine
                                    ├─ JSON decode
                                    └─ Route(bookKey)
                                              │
                                              ▼
                                    BookEngine (per-book 单线程)
                                    ├─ Engine.PlaceOrder()
                                    ├─ match() 撮合
                                    └─ 产出 Result{Order, Trades, Affected, Book}
                                              │
                                              ▼
                                    OutputDispatcher (4 workers)
                                    └─ PostTrade.ProcessBatch()
                                        ├─ DB 落盘 (orders, trades)
                                        └─ 发布 Kafka 事件 ──┬─► order.event
                                                             ├─► trade.matched
                                                             ├─► position.changed
                                                             ├─► quote.depth
                                                             ├─► quote.ticker
                                                             └─► quote.candle
                                                                    │
                                    ┌───────────────────────────────┘
                                    ▼
                          Account EventProcessor           Settlement Processor
                          ├─ 释放冻结                       ├─ 消费 MarketEvent
                          ├─ 卖方入账 USDT                  ├─ 撤销所有挂单
                          ├─ 买方入账 Position              └─ 结算赢家仓位
                          └─ 平台手续费归集                     │
                                                               ▼
                                                   SettlementCompletedEvent
                                                   ├─ 扣减 Position 资产
                                                   └─ 入账 USDT 赔付
                                                               │
                                    ┌──────────────────────────┘
                                    ▼
                            Rollup Shadow Lane
                            ├─ 追加 journal entry
                            ├─ 累积 state root
                            └─ 生成 submission bundle
                                    │
                                    ▼
                            Chain Service
                            ├─ recordBatchMetadata()
                            ├─ publishBatchData()
                            └─ acceptVerifiedBatch(proof)
                                    │
                                    ▼
                              链上状态确认
```

---

## 1. 下单

### 入口

```
POST /api/v1/orders
    → api/handler/order_handler.go: OrderHandler.SubmitOrder()
    → order/service.go: Service.SubmitOrder()
```

### 流程

1. **校验** — 市场是否存在、是否可交易、价格范围 [1, 9999]、数量 > 0
2. **冻结计算**
   - **买单**: 冻结 = price × quantity 的抵押品 (USDT)
   - **卖单**: 冻结 = quantity 的持仓资产 (POSITION:{market_id}:{outcome})
3. **预冻结** — `account.PreFreeze()` 在 `freeze_records` 表创建冻结记录
4. **类型转换** — MARKET 单在上游转为 LIMIT IOC（引擎只处理 LIMIT）
5. **发 Kafka** — 构建 `OrderCommand`，produce 到 `funnyoption.order.command`
   - partition key = `bookKey` (market_id:outcome)，保证同 book 在同 partition

### OrderCommand 结构

```json
{
  "command_id": "cmd_xxx",
  "order_id": "ord_xxx",
  "freeze_id": "frz_xxx",
  "freeze_asset": "USDT",
  "freeze_amount": 50000,
  "user_id": 1001,
  "market_id": 42,
  "outcome": "YES",
  "book_key": "42:YES",
  "side": "BUY",
  "type": "LIMIT",
  "time_in_force": "GTC",
  "stp_strategy": "",
  "price": 5000,
  "quantity": 10
}
```

---

## 2. 撮合

### 消费 (InputGateway)

```
gateway.go: ConsumerGroup → per-partition goroutine
    → FetchMessage → json.Unmarshal → CommandFromKafka → supervisor.Route(bookKey)
    → BookEngine.TryPublish(ringbuffer)
```

- ConsumerGroup 按 partition 分配 goroutine，每个 partition 独立 fetch+decode+route
- bookKey Hash 保证同 book 始终走同一 partition → SPSC ringbuffer 安全
- 异步 offset commit（后台 goroutine，每 256 条或 200ms flush 一次）

### 撮合 (Engine.PlaceOrder)

```
engine.go: PlaceOrder()
    → Validate() → getOrCreateBook() → processLimitOrder()
```

根据 `TimeInForce` 分发：

| TIF | 行为 |
|---|---|
| **GTC** | 尝试撮合，余量挂单 |
| **IOC** | 尝试撮合，余量撤销 |
| **FOK** | 预检能否全填 → 全填则执行，否则撤销 |
| **POST_ONLY** | 如果会穿价则撤销，否则挂单 |

### STP (自成交保护)

当 taker.UserID == maker.UserID 时：

| 策略 | 行为 |
|---|---|
| 空 (默认) | 跳过该 maker，继续下一个 |
| `CANCEL_TAKER` | 撤 taker，保留 maker |
| `CANCEL_MAKER` | 撤 maker，taker 继续匹配 |
| `CANCEL_BOTH` | 双撤 |

### match() 核心

```
engine.go: match(order, book)
    → 遍历对手盘 bucket (bitmap 跳跃)
    → 逐个 maker 比对价格
    → 计算 tradeQty = min(taker_remaining, maker_remaining)
    → 更新 taker/maker 的 FilledQuantity
    → 生成 Trade{TradeID, Sequence, Price, Qty, ...}
    → 全填的 maker 移出 book
```

### 撮合结果

```go
Result{
    Order:    *Order          // taker 订单（含最终状态）
    Affected: []*Order        // 被影响的 maker（含最终状态）
    Trades:   []Trade         // 产生的成交
    Book:     BookSnapshot    // 撮合后的 5 档盘口
}
```

---

## 3. 后置处理 (PostTrade)

### 流程

```
dispatcher.go: OutputDispatcher (4 workers, 按 bookKey hash 分片)
    → 批量收集 MatchResult (最多 128 条，5ms 超时)
    → posttrade/service.go: ProcessBatch()
        → store.PersistBatch()       // 一个 DB 事务
        → publisher.PublishJSONBatch() // 一次 Kafka 批量发送
```

### DB 落盘 (单事务)

- **orders 表**: upsert taker + 所有 affected maker
- **trades 表**: insert 所有 trade
- **rollup_shadow_journal_entries**: 追加 rollup 日志（如果开启）

### Kafka 事件发布

每个 MatchResult 产生以下事件：

| 事件 | Topic | 数量 | 内容 |
|---|---|---|---|
| OrderEvent | order.event | 1 + len(affected) | 订单状态变更 |
| TradeMatchedEvent | trade.matched | len(trades) | 成交详情 + 手续费 |
| PositionChangedEvent | position.changed | len(trades) × 2 | 买方 +qty, 卖方 -qty |
| QuoteDepthEvent | quote.depth | 1 | 撮合后 5 档盘口 |
| QuoteTickerEvent | quote.ticker | 1 | 最新价/最优买卖 |
| QuoteCandleEvent | quote.candle | len(trades) | K 线数据 |

例：1 个 taker 和 2 个 maker 成交 → 1 OrderEvent(taker) + 2 OrderEvent(maker) + 2 TradeMatched + 4 PositionChanged + 1 Depth + 1 Ticker + 2 Candle = **13 条 Kafka 消息**。

---

## 4. 账本 (Ledger / Account Service)

### 事件消费

```
account/service/event_processor.go
    → HandleOrderEvent()            // 冻结/释放
    → HandleTradeMatched()          // 资产转移
    → HandleSettlementCompleted()   // 结算入账
```

### 交易资产转移

当一笔 Trade 成交时（以 BUY 55@10 为例）：

| 操作 | 用户 | 资产 | 变动 | 说明 |
|---|---|---|---|---|
| 释放冻结 | 买方 | USDT | 释放 550 | 原 PreFreeze 的保证金 |
| 扣减 | 买方 | USDT | -550 | 支付给卖方 |
| 入账 | 买方 | POSITION:42:YES | +10 | 获得持仓 |
| 释放冻结 | 卖方 | POSITION:42:YES | 释放 10 | 原 PreFreeze 的持仓 |
| 扣减 | 卖方 | POSITION:42:YES | -10 | 交出持仓 |
| 入账 | 卖方 | USDT | +550 - maker_fee | 获得对价 |
| 入账 | 平台 | USDT | +taker_fee + maker_fee | 手续费 |

### 关键表

- **account_balances**: user_id, asset, available, frozen
- **freeze_records**: freeze_id, ref_type, ref_id, asset, amount, status (PENDING/APPLIED/RELEASED/CONSUMED)

---

## 5. 结算

### 触发条件

市场 oracle 判定结果后，发布 `MarketEvent{Status: "RESOLVED", ResolvedOutcome: "YES"}`

### 流程

```
settlement/service/processor.go
    → HandleMarketEvent()
        → 1. 记录 market_resolutions
        → 2. 撤销该市场所有挂单 (通过 Pipeline.SubmitCancel)
        → 3. settleWinningPositions()
            → 查询所有 outcome == ResolvedOutcome 的持仓
            → 每个赢家持仓生成 SettlementCompletedEvent
            → 发布到 Kafka: settlement.completed
```

### 赢家赔付

- 持有胜出 outcome 的每份仓位 = 1 USDT
- 例：用户持有 10 份 YES@55 买入的仓位，市场结算 YES 胜出
  - 入账 10 USDT
  - 扣减 10 POSITION:42:YES
  - 净利 = 10 - 5.5 = 4.5 USDT (扣除买入成本)

### 输家

- 持有非胜出 outcome 的仓位价值归零
- 仓位资产到期自动失效（不需要额外操作）

---

## 6. 上链 (Rollup)

### 目的

将链下撮合的状态以加密证明的形式提交到链上，确保：
- 所有交易可验证
- 用户余额可审计
- 支持强制提款 (escape hatch)

### 数据流

```
PostTrade → rollup_shadow_journal_entries (每笔交易/结算追加)
    ↓
Rollup Batcher → rollup_shadow_batches (累积 entries，计算 state root)
    ↓
Submission Bundle → Solidity calldata
    ├─ recordBatchMetadata()    // 记录 batch 元数据
    ├─ publishBatchData()       // 数据可用性
    └─ acceptVerifiedBatch()    // 提交 ZK proof (Groth16)
    ↓
Chain Service → 提交到链上
    ↓
rollup_accepted_batches (记录链上确认)
```

### State Root 组成

```
StateRoot
    ├─ BalancesRoot      // 所有账户余额的 Merkle 根
    ├─ OrdersRoot        // 所有活跃订单的 Merkle 根
    ├─ PositionsRoot     // 所有持仓的 Merkle 根
    └─ WithdrawalsRoot   // 待处理提款的 Merkle 根
```

### Rollup Freeze

提交 batch 期间全局冻结交易：
- `rollup_freeze_state.frozen = true`
- 阻止新订单、仓位变更、结算
- batch 上链确认后解冻
- 确保链下 ledger 和链上 state root 一致

---

## 7. 充提 (Custody)

### 充值

```
外部钱包 → Wallet SaaS webhook → POST /internal/custody/deposit/notify
    → custody/handler.go: DepositNotify()
        → 1. 幂等检查 (防重复)
        → 2. 地址 → 用户映射
        → 3. 链上精度 → 账本精度转换
        → 4. account.CreditBalance(USDT, amount)
        → 5. 记录 custody_deposits
        → 6. 发布 CustodyDepositEvent
```

### 提现

```
用户请求提现 → 冻结 USDT → 记录 chain_withdrawals
    → Rollup 追加到 WithdrawalsRoot
    → Batch 上链后执行链上转账
    → 如果 Rollup 离线，用户可通过 escape hatch 强制提款
```

---

## 关键表一览

| 阶段 | 表名 | 用途 |
|---|---|---|
| 下单 | orders, freeze_records | 订单状态、保证金冻结 |
| 撮合 | trades | 成交记录 |
| 账本 | account_balances, freeze_records | 余额管理、冻结/释放 |
| 持仓 | positions | 用户持仓 |
| 结算 | market_resolutions, settlement_payouts | 市场结算、赔付记录 |
| 上链 | rollup_shadow_journal_entries, rollup_shadow_batches, rollup_accepted_batches | 状态证明 |
| 充提 | custody_deposits, custody_address_mapping, chain_withdrawals | 链上资产出入 |

## Kafka Topic 一览

| Topic | 生产者 | 消费者 | 内容 |
|---|---|---|---|
| order.command | OrderService | Matching Gateway | 下单指令 |
| order.event | PostTrade | Account, WebSocket | 订单状态变更 |
| trade.matched | PostTrade | Account, WebSocket, kafka-bench | 成交事件 |
| position.changed | PostTrade | Settlement, Account | 持仓变更 |
| settlement.completed | Settlement | Account | 结算完成 |
| quote.depth | PostTrade | WebSocket | 盘口快照 |
| quote.ticker | PostTrade | WebSocket | 行情 ticker |
| quote.candle | PostTrade | WebSocket | K 线 |
| custody.deposit | Custody | — | 充值事件 |

## 幂等性保证

| 组件 | 幂等键 | 说明 |
|---|---|---|
| 订单 | OrderID | 引擎拒绝重复 OrderID |
| 成交 | TradeID | 确定性生成: `bookKey:localSeq` |
| 冻结 | FreezeID | 每次下单唯一 |
| 账本 | EntryID | 每笔转账唯一 |
| 充值 | BizID | 外部钱包去重 |
| Rollup | Sequence | 全局递增，保证连续 |
