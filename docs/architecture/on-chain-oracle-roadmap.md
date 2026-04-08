# 链上预言机演进规划（On-Chain Oracle Roadmap）

> 目标：在保留现有「链下撮合 + 链下结算事件」主链路的前提下，把加密类市场的**价格裁决**从「纯 HTTP 拉 Binance」升级为**可链上验证、可审计**的预言机路径。  
> 现状摘要见：`oracle-settled-crypto-markets.md`（当前 V1 为 `HTTP_JSON` + Binance，无链上读价）。

---

## 1. 「链上预言机」可以指什么（三种层级）

需要产品/合规先定调，技术再选型：

| 层级 | 含义 | 典型实现 | 优点 | 缺点 |
|------|------|----------|------|------|
| **A. 链上读价 + 链上裁决** | 结算结果由合约在 `resolve` 时从已部署的预言机读数并计算 YES/NO | Chainlink / Pyth / Redstone 等在目标链上的 Feed；或自研 `PriceOracle` 合约 | 用户可链上复核；信任模型清晰 | 需目标链有 Feed；gas；延迟与 heartbeats 要设计 |
| **B. 链上验证 + 链下提交** | 价格仍由 relayer/keeper 提交，但合约验证签名或 Merkle proof | Pyth pull model、API3 dAPI、自定义 `verifyAndResolve(feedId, price, ts, sig)` | 灵活；可复用多源 | 合约复杂度；仍需可信签名者集合或 L2 证明 |
| **C. 链上锚定 + 链下结算** | 链上只存 commitment（hash、批次根），业务结算仍在 Kafka/DB | 把 `observation_id` / 价格 hash 上链作审计锚 | 改动小 | **不算严格「价格来自链」**，只是审计锚 |

本路线图默认优先推进 **A 或 B**（按部署链上是否有标准 Feed 二选一），**C** 可作为过渡。

---

## 2. 设计原则（与现有系统对齐）

1. **单一事实来源**：`markets.resolve_at` 仍为裁决时间锚；链上读块时间戳或 Feed `updatedAt` 必须与 metadata 中的 `window` 规则一致（或显式升级 metadata schema）。
2. **不改 Vault 资金语义**：`FunnyVault` 继续只管托管与 claim；**不在 Vault 内做价格判断**（与现设计一致）。
3. **Settlement 仍消费 `market.event`**：链上路径的产出应是**确定性 outcome（YES/NO）**，最终仍通过现有 Kafka `market.event` + settlement 落库，避免两套 payout 逻辑。
4. **幂等与防分叉**：`resolver_ref` 从 `oracle_price:BINANCE:...` 扩展为可包含 `chain_id + feed + roundId`（或 `pyth price id + publish time`），与 `market_resolutions` 单行 upsert 规则兼容。
5. **Foundry 边界**：新合约放 `contracts/src`，测试 `contracts/test`，与 `foundry.toml` 一致。

---

## 3. 推荐分阶段路线

### Phase 0 — 决策冻结（1 周内）

- [ ] 确定**部署链**（例如 BSC / Base / Arbitrum）上 **Chainlink/Pyth 等是否已有目标交易对**（如 BTC/USD）。
- [ ] 确定产品要 **A（纯读 Feed）** 还是 **B（验证提交）**。
- [ ] 确定失败策略：Feed 停摆时是否允许 **operator 多签** 或 **延长 resolve 窗口**（写入 metadata 版本 2）。

### Phase 1 — 合约：只读预言机适配层（2–4 周）

**目标**：链上函数 `getObservation(marketId)` 或 `resolveOutcome(marketConfigHash)` 返回与链下规则一致的 outcome，**不先改 Go 主流程**（可先 Foundry fork 测试 + 脚本模拟）。

交付物示例（命名可调整）：

- `contracts/src/oracle/MarketPriceOracle.sol`（或按供应商拆包）
  - `function latestRoundData(address feed) returns (...)` 封装 Chainlink AggregatorV3Interface；或封装 Pyth `getPriceNoOlderThan`。
  - `function resolve(CryptoResolutionParams calldata p) external view returns (Outcome)`：输入为 ABI-encoded 的阈值、comparator、时间窗口上界（与 `metadata.resolution` 对齐）。
- 单元测试：mock feed / mock Pyth；覆盖 comparator、stale price revert、窗口外 revert。

**验收**：主网 fork 或本地 Anvil + mock，证明**同样输入 → 与 Go `ResolveOutcome` 一致**（需对齐小数位与 rounding）。

### Phase 2 — 链下 Oracle Worker：`source_kind = EVM_READ`（3–5 周）

**目标**：worker 不再调用 Binance HTTP，改为：

1. 调 **JSON-RPC** `eth_call` 读上述合约的 `view` 结果；或
2. 读 Feed + 在 worker 内只做**与合约相同的纯函数校验**（双实现一致性测试）。

变更点：

- `internal/oracle/service/metadata.go` / `ParseContract`：扩展 `oracle.source_kind = EVM_READ`，字段含 `chain_id`、`oracle_adapter_address`、`feed_address` 或 `pyth_price_id`。
- 新 package：`internal/oracle/evm`（或 `chainread`）：复用项目已有 viem/ethers 风格 client（若 Go 侧仅有 ethclient，用 `abigen` 生成绑定）。
- **证据 `evidence`**：写入 `chain_id`、`block_number`、`calldata` 或 `round_id`、`returned_price`、`tx` 无（纯 view 则无 tx），`raw_payload_hash` 改为 RPC 响应 hash。

**验收**：本地 Anvil 部署 mock feed + 跑通 `pollOnce` → `OBSERVED` → `market.event`（与现 worker 测试同构）。

### Phase 3 — 可选：链上提交（仅当 Phase 1 用 view 不够）

若需 **B 模式**（pull/update price）：

- 增加 `keeper` 角色：`submitPrice` 交易，合约 `verify` 后发出 `OutcomeCommitted(marketId, outcome, ref)`。
- Worker 或独立 `keeper` 服务监听事件，再发 Kafka（仍一条 settlement 主链路）。

**验收**：staging 上跑通一次「到期 → 链上交易 → 事件 → settlement」。

### Phase 4 — 运维与风控

- [ ] Staging：`oracle` 服务 + RPC 配置 + Feed 地址表（按链分环境变量）。
- [ ] 监控：Feed heartbeat、RPC 错误率、`TERMINAL_ERROR` 告警。
- [ ] 文档：升级 `oracle-settled-crypto-markets.md` 的 metadata schema（version 2）。

---

## 4. 与现有 `HTTP_JSON` / Binance 的关系

- **长期**：加密类市场默认走 `EVM_READ`；Binance 可作为 **fallback**（metadata 显式 `fallback_provider`）或仅用于非生产调试。
- **兼容**：旧市场 metadata `version:1` + `HTTP_JSON` 继续被旧 worker 支持，直到迁移脚本或运营手动关闭。

---

## 5. 关键风险与缓解

| 风险 | 缓解 |
|------|------|
| Feed 更新频率低于市场 `resolve_at` 精度 | 在规则中要求 `resolve_at` 对齐 Feed heartbeat；或允许 `resolve_at` 落在「上一有效 round」 |
| 链重组 | view 调用指定 `block_number`（历史块）若 Feed 支持；否则以 finalized block 为准并写进 evidence |
| Go/Solidity 舍入不一致 | 共享测试向量（JSON fixtures）双向断言 |
| Gas / 多链 | Phase 1 仅 view；提交路径再评估 gas 与 relayer |

---

## 6. 待你拍板的开放问题

1. **目标链**上优先接入 **Chainlink** 还是 **Pyth**（或两者按交易对选）？
2. 裁决是否必须 **100% 链上可复现**，还是允许 **operator 紧急多签** 覆盖（写入合约 `Governance`）？
3. 是否要求 **用户可在浏览器用同一 RPC 独立验证** outcome（影响 evidence 字段与 UI）？

---

## 7. Staging：oracle worker（当前迭代）

在动 EVM / 新合约之前，先在 staging 跑通：

- **Compose 服务** `oracle`（[deploy/staging/docker-compose.staging.yml](/deploy/staging/docker-compose.staging.yml)），镜像 [deploy/docker/oracle.Dockerfile](/deploy/docker/oracle.Dockerfile)。
- **可观测**：`GET :9191/healthz`、`GET :9191/debug/oracle`（计数器 + Kafka topic + 回放说明）。
- **运维说明**：[docs/operations/oracle-staging.md](/docs/operations/oracle-staging.md)。

## 8. 建议的下一步（执行顺序）

1. 选定链 + Feed 清单（Phase 0）。
2. 开 `contracts` PR：Phase 1 适配合约 + Foundry 测试。
3. 并行：在 `internal/oracle` 增加 `EVM_READ` 解析与 golden tests。
4. 再合：worker 切换与 staging 部署（Phase 2–4）。

本文档为规划项；实现时应在 `docs/harness/plans/active` 拆具体任务与验收标准，并更新 `oracle-settled-crypto-markets.md` 的 metadata 附录。
