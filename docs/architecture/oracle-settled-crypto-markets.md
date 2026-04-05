# Oracle-Settled Crypto Markets

这份文档把“加密类市场通过预言机获取价格自动结算”的 V1 contract 收口成一个明确边界。

目标不是立刻宽改运行时代码，而是先固定：

- 哪些 market 可以进入这条 lane
- market metadata 的 canonical shape
- `market_resolutions` 里的 evidence / status contract
- 自动结算应该落在哪个服务边界
- 手动 operator resolve 在什么情况下仍然是安全 fallback

## 1. 设计结论

V1 的 oracle-settled crypto market 先只支持：

- `category_key = CRYPTO`
- 交易选项仍然是二元 `YES / NO`
- 结算规则是“某个价格是否满足一个阈值比较规则”
- `markets.close_at` 是 trading cutoff，不是 settlement timestamp
- 结算触发点仍然以 `markets.resolve_at` 为 canonical settlement timestamp
- 自动结算由一个独立的 oracle worker 完成，不塞进 `chain-service` 或 `settlement`
- 最终 settlement trigger 仍然复用现有 `market.event -> settlement` 主链路
- `market_resolutions` 继续作为 resolution checkpoint 和 evidence 容器
- 现有手动 operator resolve 保留为 fallback，但只能在自动 lane 还没有进入 `OBSERVED / RESOLVED` 时介入

这意味着：

- 不需要先改撮合、仓位或 payout 语义
- 不需要先引入新的链上合约
- 不需要先扩成多选项市场
- 不需要先把价格证据塞进 Kafka event payload

## 2. 适用范围

这条 lane 只适用于“二元加密价格判断市场”，例如：

- `BTC/USDT 在 resolve_at 时是否 >= 85000`
- `ETH/USDT 在 resolve_at 时是否 < 4200`

YES/NO 的判定规则固定为：

- 比较规则成立 -> `resolved_outcome = YES`
- 比较规则不成立 -> `resolved_outcome = NO`

不在这次 contract 里的能力：

- 多阈值区间市场
- 多 outcome 加密市场
- 多数据源仲裁 / median / quorum
- 已经结算后的反向改判
- 链上强制验证或链上 settlement

## 3. Metadata Contract

### 3.1 Canonical location

加密类自动结算的配置放在：

- `markets.metadata.resolution`

现有 `metadata.sourceUrl / sourceName / sourceKind` 继续表示内容来源或运营来源。
它们不是结算预言机字段，不能复用成价格源配置。

### 3.2 Required shape

```json
{
  "resolution": {
    "version": 1,
    "mode": "ORACLE_PRICE",
    "market_kind": "CRYPTO_PRICE_THRESHOLD",
    "manual_fallback_allowed": true,
    "oracle": {
      "source_kind": "HTTP_JSON",
      "provider_key": "BINANCE",
      "instrument": {
        "kind": "SPOT",
        "base_asset": "BTC",
        "quote_asset": "USDT",
        "symbol": "BTCUSDT"
      },
      "price": {
        "field": "LAST_PRICE",
        "scale": 8,
        "rounding_mode": "ROUND_HALF_UP",
        "max_data_age_sec": 120
      },
      "window": {
        "anchor": "RESOLVE_AT",
        "before_sec": 300,
        "after_sec": 300
      },
      "rule": {
        "type": "PRICE_THRESHOLD",
        "comparator": "GTE",
        "threshold_price": "85000.00000000"
      }
    }
  }
}
```

### 3.3 Field rules

- `version`
  - 先固定为 `1`
- `mode`
  - `ORACLE_PRICE` 表示由预言机价格自动给出 YES / NO
- `market_kind`
  - 先固定为 `CRYPTO_PRICE_THRESHOLD`
- `manual_fallback_allowed`
  - 先固定为 `true`
- `oracle.source_kind`
  - V1 first cut 只实现 `HTTP_JSON`
  - 预留将来扩成 `EVM_READ` 或 `SIGNED_ATTESTATION`
- `oracle.provider_key`
  - 先作为稳定机器值，例如 `BINANCE`
- `oracle.instrument`
  - `kind` first cut 只支持 `SPOT`
  - `symbol` 是 resolver 真正请求价格时用的 canonical symbol
- `oracle.price.field`
  - first cut 只支持 `LAST_PRICE`
- `oracle.price.scale`
  - 价格规范化后的小数位数
- `oracle.price.rounding_mode`
  - first cut 固定 `ROUND_HALF_UP`
- `oracle.price.max_data_age_sec`
  - 拉到的价格样本离 `effective_at` 太久就视为 stale
- `oracle.window.anchor`
  - first cut 固定 `RESOLVE_AT`
  - 也就是 `markets.resolve_at` 是 settlement timestamp 的 canonical source
- `oracle.window.before_sec`
  - 允许 price sample 早于 `resolve_at` 的最大秒数
- `oracle.window.after_sec`
  - 允许 price sample 晚于 `resolve_at` 的最大秒数
- `oracle.rule.type`
  - first cut 固定 `PRICE_THRESHOLD`
- `oracle.rule.comparator`
  - first cut 支持 `GT / GTE / LT / LTE`
- `oracle.rule.threshold_price`
  - 用 decimal string 存，避免 JS / JSON float 歧义

### 3.4 Market-level validation

如果 market 进入这条 lane，必须同时满足：

- `category_key = CRYPTO`
- `options` 通过当前二元交易校验，也就是只能是 `YES / NO`
- `resolve_at > 0`
- `metadata.resolution.mode = ORACLE_PRICE`
- `oracle.window.anchor = RESOLVE_AT`
- `oracle.instrument.symbol` 非空
- `oracle.rule.threshold_price` 能被规范化到 `price.scale`

## 4. Resolver Contract

### 4.1 Chosen boundary

选择：

- 新增独立 oracle worker / service

不选择：

- `chain-service` 扩责
- `settlement` 自己去拉外部价格
- admin / API 线程里直接做自动 resolve

原因：

- `chain-service` 当前职责是 vault 监听与链上 claim/withdraw，不应该混入外部 HTTP price fetch
- `settlement` 当前是 `market.event` 的消费者，保持它“只消费结果，不负责找结果”更安全
- admin / API 是 operator 入口，不应变成定时扫描和重试执行器

### 4.2 Resolver responsibilities

oracle worker 负责：

- 选择到期且 eligible 的 oracle crypto markets
- 拉取价格并规范化
- 根据 comparator 计算 YES / NO
- 先把 observation / evidence 写入 `market_resolutions`
- 再发布现有 `market.event`
- 在 source outage / stale data / conflict 时维护 retryable vs terminal 状态

oracle worker 不负责：

- 直接做 payout
- 改 matching 规则
- 改 vault 合约
- 改 claim 流程

## 5. Resolution State Contract

`market_resolutions` 继续复用，不新增第一批 schema。

### 5.1 Columns

- `resolver_type`
  - `ADMIN`
  - `ORACLE_PRICE`
- `resolver_ref`
  - 自动结算用稳定 idempotency key：
  - `oracle_price:{provider_key}:{symbol}:{resolve_at}`
- `resolved_outcome`
  - 只有在拿到可结算 observation 后才填 `YES / NO`
- `evidence`
  - 保存规范化证据与错误摘要

### 5.2 Status values

first cut 约定：

- `PENDING`
  - 市场已声明 `ORACLE_PRICE`，但还未得到可用 observation
- `RETRYABLE_ERROR`
  - 外部源暂时失败，可继续重试
- `TERMINAL_ERROR`
  - 元数据错误、symbol 不支持、比较规则不支持、或观测冲突，必须人工介入
- `OBSERVED`
  - 已拿到 final observation，`resolved_outcome` 已确定
  - `evidence.dispatch.status = PENDING` 表示 observation 已写入，但对应 resolved `market.event` 仍允许补发
  - `evidence.dispatch.status = DISPATCHED` 表示 resolved `market.event` 已成功发出，后续 poll 不应再重复 emit
- `RESOLVED`
  - 现有 settlement 消费 `market.event` 后落成最终状态

说明：

- 现有 `internal/settlement/service/sql_store.go` 在 `ON CONFLICT` 时只更新
  `status / resolved_outcome / updated_at`
- 所以 oracle worker 先写入 `resolver_type / resolver_ref / evidence` 后，
  settlement 复用当前 upsert 逻辑时不会把 oracle evidence 覆盖掉

## 6. Evidence Contract

`market_resolutions.evidence` 的 canonical shape：

```json
{
  "version": 1,
  "resolution_mode": "ORACLE_PRICE",
  "source": {
    "source_kind": "HTTP_JSON",
    "provider_key": "BINANCE",
    "instrument": {
      "kind": "SPOT",
      "base_asset": "BTC",
      "quote_asset": "USDT",
      "symbol": "BTCUSDT"
    },
    "price_field": "LAST_PRICE",
    "price_scale": 8
  },
  "rule": {
    "type": "PRICE_THRESHOLD",
    "comparator": "GTE",
    "threshold_price": "85000.00000000"
  },
  "window": {
    "anchor": "RESOLVE_AT",
    "target_time": 1775886400,
    "before_sec": 300,
    "after_sec": 300,
    "max_data_age_sec": 120
  },
  "observation": {
    "observation_id": "oracle_price:BINANCE:BTCUSDT:1775886400",
    "fetched_at": 1775886412,
    "effective_at": 1775886405,
    "observed_price": "85123.45000000",
    "resolved_outcome": "YES",
    "raw_payload_hash": "sha256:...",
    "raw_payload": {
      "symbol": "BTCUSDT",
      "price": "85123.45"
    }
  },
  "retry": {
    "attempt_count": 2,
    "last_attempt_at": 1775886412,
    "next_retry_at": 0,
    "last_error_code": ""
  },
  "dispatch": {
    "status": "DISPATCHED",
    "attempt_count": 1,
    "last_attempt_at": 1775886413,
    "dispatched_at": 1775886413
  }
}
```

规则：

- `source / rule / window` 要做 snapshot，不能只依赖 market metadata 事后回读
- `observation.raw_payload` first cut 可以直接内联在 JSONB 里，不要求先上对象存储
- `raw_payload_hash` 用于后续做 UI / 审计校验
- 如果没有成功 observation，`observation` 可以为空，但 `retry.last_error_code` 必须可读
- `dispatch` 只在已有 final observation 时出现：
  - `PENDING` 表示允许后续 poll / restart 继续补发 resolved `market.event`
  - `DISPATCHED` 表示当前 observation 的 resolved `market.event` 已发出，worker 必须继续保持 duplicate-emit guard

### 6.1 Error code contract

建议先固定以下错误码：

- `SOURCE_TIMEOUT`
- `SOURCE_UNAVAILABLE`
- `STALE_PRICE`
- `PRICE_NOT_IN_WINDOW`
- `UNSUPPORTED_SYMBOL`
- `UNSUPPORTED_RULE`
- `CONFLICTING_OBSERVATION`
- `INVALID_METADATA`

## 7. End-to-End Auto-Resolution Flow

1. admin 创建 market
   - `category_key = CRYPTO`
   - `options = YES / NO`
   - `metadata.resolution.mode = ORACLE_PRICE`
2. oracle worker 扫到到期 market
   - 条件是 `resolve_at <= now`
   - 交易是否早已停止仍然看 `close_at`；这条 lane 不把 `resolve_at` 当成 trading boundary
   - market 尚未 `RESOLVED`
3. worker 拉取外部价格
4. worker 规范化 price，按 comparator 计算 `YES / NO`
5. worker upsert `market_resolutions`
   - `status = OBSERVED`
   - `resolver_type = ORACLE_PRICE`
   - `resolver_ref = oracle_price:{provider}:{symbol}:{resolve_at}`
   - `resolved_outcome = YES | NO`
   - `evidence = canonical observation snapshot`
   - `evidence.dispatch.status = PENDING`
6. worker 发布当前格式不变的 `market.event`
   - 这一步仍然只传 `market_id + status + resolved_outcome`
   - evidence 不塞进 Kafka payload
7. publish 成功后，worker 把同一行 `evidence.dispatch.status` 更新成 `DISPATCHED`
   - 如果 publish 失败，则保留 `OBSERVED + dispatch=PENDING`
   - 后续 poll 或 restart 可以继续按同一 observation 安全补发
8. settlement 按现有逻辑消费 `market.event`
   - `markets.status = RESOLVED`
   - `markets.resolved_outcome` 落库
   - 取消 active orders
   - 计算 payout
   - `market_resolutions.status` 被更新成 `RESOLVED`

close / resolve 语义补充：

- `close_at` 到达后，oracle market 也应先进入 runtime `CLOSED`
- 在 `resolve_at` 之前，它只是停止交易，不代表已经结算
- 到了 `resolve_at` 之后，oracle worker 才尝试 observation / publish / settlement

## 8. Idempotency Rules

### 8.1 Resolver idempotency

自动结算的天然幂等键是：

- `oracle_price:{provider_key}:{symbol}:{resolve_at}`

同一个 market 的重复扫描必须满足：

- 如果 `market.status = RESOLVED`，直接跳过
- 如果 `market_resolutions.status = OBSERVED` 且
  `resolver_ref` 相同、`resolved_outcome` 相同：
  - `evidence.dispatch.status = DISPATCHED` -> 直接跳过
  - `evidence.dispatch.status = PENDING` 或缺失 -> 允许补发同一 resolved `market.event`
- 如果 `market_resolutions.status = RESOLVED` 且
  `resolver_ref` 相同、`resolved_outcome` 相同，直接跳过
- 如果同一 `resolver_ref` 下得到不同 `resolved_outcome` 或不同规范化价格，
  直接转 `TERMINAL_ERROR / CONFLICTING_OBSERVATION`，禁止自动发布第二个 outcome

### 8.2 Manual fallback safety

手动 operator resolve 继续保留，但安全边界要收紧成：

- 如果 `market.status = RESOLVED`，拒绝手动 resolve
- 如果 `market_resolutions.status = OBSERVED`，拒绝手动 resolve
- 只有在 `PENDING / RETRYABLE_ERROR / TERMINAL_ERROR` 时，才允许 operator 走人工 fallback

这条规则的核心目的，是避免同一个 market 先后发出两个不同 outcome 的
`market.event`，从而触发双边 payout 或结算翻转。

### 8.3 Important limit

first cut 明确不支持：

- 市场已经 `RESOLVED` 之后再改 outcome

这类“post-settlement override”会涉及 payout rollback，不属于当前 MVP 安全范围。

## 9. Failure Handling

### 9.1 Delayed price availability

如果 source 还没返回落在允许窗口内的有效价格：

- 写 `RETRYABLE_ERROR`
- `last_error_code = PRICE_NOT_IN_WINDOW`
- 在 `resolve_at + after_sec` 之前继续重试

### 9.2 Source outage

如果 HTTP 超时、5xx、限流：

- 写 `RETRYABLE_ERROR`
- 保留 `attempt_count / next_retry_at`
- 超过运营允许等待时间后，再交给人工 fallback

### 9.3 Stale data

如果样本时间戳距离 `effective_at` 超过 `max_data_age_sec`：

- 写 `RETRYABLE_ERROR`
- `last_error_code = STALE_PRICE`

### 9.4 Conflicting observations

如果同一个 `resolver_ref` 被拉到两个互相冲突的规范化结果：

- 立刻写 `TERMINAL_ERROR`
- `last_error_code = CONFLICTING_OBSERVATION`
- 停止自动 emit，要求人工处理

### 9.5 Invalid metadata

如果 symbol、comparator、threshold 或 scale 本身不合法：

- 写 `TERMINAL_ERROR`
- `last_error_code = INVALID_METADATA`
- 不进入自动结算

## 10. First Implementation Cut

推荐第一条实现任务边界：

- 只支持 `CRYPTO + YES/NO + ORACLE_PRICE + HTTP_JSON + BINANCE + SPOT + LAST_PRICE`
- 新增一个独立 oracle worker
- market create / read 路径只做 `metadata.resolution` 的校验与透传
- 自动结算只复用现有 `market_resolutions` 表，不加新 migration
- `market.event` payload 保持不变
- 手动 resolve 只补一个“若已 `OBSERVED / RESOLVED` 则拒绝”的冲突保护

这个 cut 有几个好处：

- 风险最小，不会先打开撮合 / payout / chain 的大面积改动
- 直接复用当前 settlement 流程
- 能把 metadata / evidence / resolver contract 真的跑通一条闭环
- 为后续扩 provider 或扩 source_kind 留出了明确位置

## 11. Foundry Boundary

这次 design 不要求新增链上 helper。

原因：

- 当前 resolution 结果仍然由后端触发 settlement
- `FunnyVault.sol` 只负责 custody / claim，不参与价格判定
- 先把链下 oracle contract 跑通，比先设计链上 adapter 更安全

如果未来确实要加链上 adapter，边界必须仍然是：

- Solidity source: `contracts/src`
- tests: `contracts/test`
- scripts: `contracts/script`
- toolchain: repo root `foundry.toml`

但这不属于 first cut。

## 12. Rejected Options

- 把 price fetch 塞进 `chain-service`
  - 拒绝，职责会从链上事件监听漂移到通用外部数据采集
- 让 `settlement` 在消费 `market.event` 时自己去拉价格
  - 拒绝，会把“找结果”和“消费结果”混在一起，出错面更大
- 一上来加 append-only resolution attempts 新表
  - 暂时拒绝，first cut 先复用 `market_resolutions` 单行 checkpoint；后续如果审计深度不够，再单独扩表
- 一上来支持多 provider 仲裁
  - 暂时拒绝，先把单 provider 的 deterministic contract 跑通
- 一上来做链上 price adapter / on-chain settlement
  - 暂时拒绝，当前 MVP 的 trust boundary 仍然是“链上托管，链下交易与结算驱动”

## 13. Residual Risks

- `market_resolutions` 只有一行最新状态，不是完整 append-only attempts ledger
- `dispatch` 也是 latest-row checkpoint，不是完整 append-only dispatch attempts ledger
- first cut 的 manual fallback 如果不补状态检查，仍然存在和 oracle worker 竞争 emit 的风险
- 单 provider 模式会把 source outage 暴露给运营 fallback
- `raw_payload` 直接放 JSONB 对长期体积不够优雅，但对 first cut 足够简单
