# Mode B `ZK-Rollup` Architecture Contract

这份文档把 FunnyOption 的目标 `Mode B` 架构收口成一份可执行设计。

先讲结论：

- 当前 FunnyOption **还不是** `Mode B`
- 当前系统仍然是：
  - `FunnyVault` 链上托管
  - `internal/api + matching + account + settlement + oracle` 链下执行
  - `PostgreSQL + Kafka` 作为当前事实边界
- 当前系统还没有：
  - proof-verified state transition
  - `state_root`
  - `ZK-Rollup` data availability
  - forced withdrawal / freeze / escape hatch
  - canonical slow / fast / forced withdrawal contracts

所以这次任务的目标不是把现有实现包装成 `Mode B`，而是明确：

- 哪些边界必须替换
- 哪些服务还可以继续 operator-run
- 什么状态、批处理、退出模型才算 FunnyOption 的 `Mode B`

## 1. Fixed decisions

- `Mode B` 固定为：
  - offchain operator execution
  - onchain custody
  - proof-verified batch settlement
  - user exit guarantees
- data availability 固定为 `ZK-Rollup`：
  - first cut 只接受 L1-native DA
  - first truthful lane 以 L1 `calldata` 为 canonical DA
  - 不做 validium
  - 不做 DAC
  - 不做 external DA first cut
- withdrawal model 固定包含三条 lane：
  - slow withdrawal
  - fast withdrawal
  - forced withdrawal
- 本文档只定义 architecture contract，不进入：
  - prover implementation
  - verifier implementation
  - full L1 contract implementation

## 2. Target system boundary

### 2.1 Offchain operator services

这些服务在 `Mode B` 下仍然可以继续 operator-run，但它们不再是最终 settlement truth：

- API / auth gateway
  - 接收 wallet auth、trading key auth、order intent、withdraw request
- sequencer / matcher
  - 维护 order book
  - 执行 price-time priority
  - 产生 deterministic execution result
- risk / fee / market policy
  - 做产品规则校验
  - 做 fee policy、inventory rule、market lifecycle policy
- oracle adapter
  - 拉取价格或解析外部 resolution input
  - 做规范化与 evidence snapshot
- read models / websocket / admin
  - 继续做面向产品和运营的查询与展示
- fast-withdraw LP quoting
  - 给用户报价并垫资

这些服务可以 operator-run 的原因是：

- 它们决定的是 liveness、体验、撮合策略、运营策略
- 最终资金安全、状态正确性、退出保证不应再依赖它们单方面写 SQL

### 2.2 New proving services

`Mode B` 需要新增一组 offchain proving / batching services。它们仍然可以由 operator 运行，但它们产出的 artifact 才是新的 truth boundary。

- sequencer journal writer
  - 为每个被接收的状态变化写 append-only ordered journal
- batch materializer
  - 把 journal span、L1 deposits、oracle inputs、withdrawal instructions 组装成 durable batch input
- deterministic state replayer / witness builder
  - 从 `prev_state_root + batch_input` 计算 `next_state_root`
- prover coordinator
  - 生成并聚合证明
- DA publisher
  - 把 batch input 的 canonical encoding 发布到 L1
- proof submitter / finality watcher
  - 提交 proof
  - 等待 state update finality

### 2.3 L1 contracts

L1 contract boundary first cut 固定成三类：

- `FunnyRollupCore`
  - verifier boundary
  - latest proven state root
  - batch metadata hash
  - deposit cursor / forced-withdrawal queue cursor
  - freeze flag
  - escape-hatch mode
- `FunnyRollupVault`
  - custody
  - deposit intake
  - canonical slow-withdraw / forced-withdraw claim payout
  - claim nullifier
- `FunnyFastWithdrawPool`
  - LP liquidity pool
  - fast withdrawal reimbursement
  - LP claim right redemption

当前 [`FunnyVault.sol`](/Users/zhangza/code/funnyoption/contracts/src/FunnyVault.sol) 只能算 direct-vault custody helper，不是 `Mode B` rollup contract，因为它没有：

- verifier
- state root
- batch data commitment
- withdrawal nullifier
- forced withdrawal queue
- freeze / escape hatch

## 3. Canonical state model

### 3.1 Identity contract

当前 SQL `user_id` 不能直接当作 `Mode B` canonical identity。

`Mode B` 需要一个稳定的 `account_id`：

- 绑定到 `wallet_address + chain_id + vault_scope`
- trading key 只是这个 account 的授权键，不是账户本体
- 当前 `wallet_sessions` 可以继续作为 operator mirror
- 但 `wallet_sessions.last_order_nonce` 不能再是 replay truth

### 3.2 Global root composition

`Mode B` first cut 的 global root contract 固定为：

```text
state_root =
  H(
    version,
    balances_root,
    orders_root,
    positions_funding_root,
    withdrawals_root
  )

orders_root =
  H(
    nonce_root,
    open_orders_root
  )

positions_funding_root =
  H(
    position_root,
    market_funding_root,
    insurance_root
  )
```

说明：

- L1 contract 只需要存一个 canonical `state_root`
- component roots 是 offchain replay、proof public input、审计和 future migration 的 canonical decomposition
- deposit queue 保留在 L1 contract storage，不单独放进 first-cut root

### 3.3 Balances tree

`balances_root` 的 key 固定为 `(account_id, asset_id)`。

leaf 至少包含：

- `free_balance`
- `locked_balance`
- `last_batch_id`

这棵树替代当前的：

- `account_balances.available`
- `account_balances.frozen`
- `freeze_records` 作为最终资金事实

`freeze_records` 在 `Mode B` 里可以继续存在，但只能作为 operator read model / debug mirror。

### 3.4 Orders / replay protection

`orders_root` first cut 固定包含两部分：

- `nonce_root`
  - key: `(account_id, auth_key_id)`
  - leaf:
    - `next_nonce`
    - `key_status`
    - `scope`
- `open_orders_root`
  - key: `order_id`
  - leaf:
    - `account_id`
    - `market_id`
    - `outcome_or_leg`
    - `side`
    - `limit_price`
    - `remaining_quantity`
    - `reserved_collateral`
    - `status`

这样做的目的不是把整个 book 原样搬上链，而是固定下面两件事：

- replay protection 必须是 proof-enforced，不再只是 SQL nonce
- 任何会长期占用 collateral 的 resting order，都必须有 state commitment

当前 [`CreateOrder`](/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go) 里的流程：

- 校验 trading key
- 递增 `wallet_sessions.last_order_nonce`
- 预冻结余额
- 发布 `order.command`

在 `Mode B` 里仍然可以保留“API -> sequencer”运行时形状，但 truth 必须改成：

- nonce advance 进入 `nonce_root`
- reserved collateral 进入 `balances_root / open_orders_root`
- journal / batch input 才是可重放事实

### 3.5 Positions / funding / insurance state

`positions_funding_root` 用来承载用户仓位、市场 funding 累加器、保险池余额。

`position_root`：

- key: `(account_id, market_id, leg_id)`
- leaf:
  - `quantity`
  - `cost_basis`
  - `realized_pnl`
  - `funding_snapshot`
  - `settlement_status`

`market_funding_root`：

- key: `market_id`
- leaf:
  - `cumulative_funding_index`
  - `last_oracle_ref`
  - `market_settlement_state`

`insurance_root`：

- key: `risk_bucket_id` or `asset_id`
- leaf:
  - `insurance_balance`
  - `socialized_loss_accumulator`

对当前 FunnyOption 来说：

- `positions` 现在是 SQL snapshot
- `settlement_payouts` 现在是 SQL payout truth
- 现有二元市场 first cut 可以把 funding-related field 固定为 `0`
- 但 root shape 先预留 funding / insurance namespace，避免未来再次改 root contract

### 3.6 Withdrawal state

`withdrawals_root` 的 key 固定为 `withdrawal_id`。

leaf 至少包含：

- `account_id`
- `asset_id`
- `amount`
- `recipient`
- `lane`
  - `SLOW`
  - `FAST`
  - `FORCED`
- `status`
- `beneficiary`
  - user or LP
- `request_batch_id`
- `claim_nullifier`

这棵树替代当前“operator 记账后再调合约打款”的提款事实。

当前 [`chain_withdrawals`](/Users/zhangza/code/funnyoption/migrations/006_chain_withdrawals.sql) 和 `FunnyVault.processClaim()` 只适合作为 direct-vault 模式的队列/镜像，不足以构成 `Mode B` canonical withdrawal truth。

## 4. Batch truth model

### 4.1 Sequencer journal

`Mode B` 必须引入 append-only `sequencer journal`。

journal entry 是 first-class truth，不是 debug log。

entry type first cut 至少覆盖：

- `DepositCredited`
- `OrderAccepted`
- `OrderCancelled`
- `TradeMatched`
- `MarketResolved`
- `FundingApplied`
- `WithdrawalRequested`
- `FastWithdrawalAssigned`
- `ForcedWithdrawalSatisfied`
- `FeeTransferred`

规则：

- sequencer 只有在 journal durable write 成功后，才能把 action 当成已接收
- Kafka topic 可以继续存在，但 Kafka offset 不是 canonical truth
- SQL snapshot 可以继续存在，但 SQL row state 不是 canonical truth

### 4.2 Durable batch input

每个 batch 都必须有一个可重放、可哈希、可发布到 L1 的 durable batch input。

batch input first cut 至少绑定：

- `batch_id`
- `prev_state_root`
- `journal_range`
- `deposit_queue_cursor`
- `forced_withdrawal_cursor`
- oracle / resolution input refs
- execution payload
- `next_state_root`
- `batch_data_hash`

要求：

- batch input 必须能在 operator 重启后完整重放
- batch input 必须能从 journal 和外部引用中重新构造
- batch input 必须有 canonical binary encoding
- prover、auditor、disaster-recovery replayer 都必须消费同一份 input

### 4.2.1 `shadow-batch-v1` current witness / public-input boundary

当前 repo 里给 prover follow-up 固定下来的 `shadow-batch-v1` contract 是：

- witness:
  - ordered `entries[]`
  - each entry carries one canonical typed payload:
    - `NonceAdvanced`
    - `OrderAccepted`
    - `OrderCancelled`
    - `TradeMatched`
    - `DepositCredited`
    - `WithdrawalRequested`
    - `MarketResolved`
    - `SettlementPayout`
  - one explicit namespace-truth contract:
    - `balances_root` = truthful shadow
    - `orders_root.open_orders_root` = truthful shadow
    - `orders_root.nonce_root` = truthful shadow of API/auth accepted
      order-nonce advances as a monotonic `next_nonce` floor
    - `positions_funding_root.position_root` = truthful shadow
    - `positions_funding_root.market_funding_root` = truthful shadow for
      market settlement state, while funding index remains `0`
    - `positions_funding_root.insurance_root` = deterministic zero placeholder
    - `withdrawals_root` = direct-vault shadow mirror, not canonical future
      claim-nullifier truth
- public inputs:
  - `batch_id`
  - `first_sequence_no`
  - `last_sequence_no`
  - `entry_count`
  - `batch_data_hash`
  - `prev_state_root`
  - `balances_root`
  - `orders_root`
  - `positions_funding_root`
  - `withdrawals_root`
  - `next_state_root`
- minimal L1 metadata subset:
  - `batch_id`
  - `batch_data_hash`
  - `prev_state_root`
  - `next_state_root`

这份 boundary 的目的不是声称 prover / verifier 已经存在，而是让下一条 prover
worker 不必重新决定：

- batch 到底消费哪些 durable input
- 哪些 namespace 已经 truthful shadow
- 哪些 namespace 还是 placeholder

### 4.2.2 First proof-lane nonce / auth contract

`TASK-CHAIN-012` 对第一版 proof lane 的选择固定为：

- 保留当前 `orders_root.nonce_root` 的 monotonic-floor 语义
- 不先把 API/runtime 改成 gapless nonce
- 但 verifier-gated batch acceptance **不能** 继续把当前 operator-side auth
  当作最终 proof truth

具体 contract：

- durable nonce leaf 继续固定为当前 shadow contract：
  - key = `(account_id, auth_key_id)`
  - leaf = `next_nonce + scope + key_status`
  - transition rule = `accepted_nonce >= current next_nonce`
- 第一版 proof lane 的 auth obligation 固定为：
  - 每个推进 `nonce_root` 的 `NONCE_ADVANCED` witness，都必须对应一个
    可验证的 canonical trading-key order authorization
  - verifier/prover 需要验证 trading-key order signature，而不是只信任
    API 先前已经做过签名检查
  - canonical V2 trading-key registration now lands one witness-only
    `TRADING_KEY_AUTHORIZED` journal entry with:
    - `authorization_ref = trading_key_id:challenge`
    - wallet-scoped `chain_id + vault_address + trading_public_key`
    - EIP-712 typed-data hash + wallet signature
  - 每个 verifier-eligible `NONCE_ADVANCED` payload 现在也要带：
    - `order_authorization.authorization_ref`
    - exact order-intent message / hash / signature
    - accepted trading-key scope metadata copied from the active key row
  - deprecated blank-vault `/api/v1/sessions` rows 仍然可以继续服务当前兼容
    runtime，但它们必须明确标成 non-verifier-eligible
  - nonce proof 证明的是“这个 auth key 的 floor 被合法推进”，不是
    “所有 nonce 都 gapless 地逐个消费”
- 当前 repo 里已经把这条 auth lane materialize 成一份 explicit verifier-prep
  contract：
  - normalized join tuple =
    `authorization_ref + trading_key_id + account_id + wallet_address +
    chain_id + vault_address + trading_public_key + trading_key_scheme +
    scope + key_status`
  - [`BuildVerifierAuthProofContract(history, batch)`](/Users/zhangza/code/funnyoption/internal/rollup/verifier_contract.go)
    会消费：
    - prior `TRADING_KEY_AUTHORIZED` witness refs
    - target batch 里的 `NONCE_ADVANCED.payload.order_authorization`
  - 它会为 target batch 的每条 nonce auth 明确输出：
    - `JOINED`
    - `MISSING_TRADING_KEY_AUTHORIZED`
    - `NON_VERIFIER_ELIGIBLE`
  - future verifier gate 只能把 target batch 里 auth rows 全部为 `JOINED`
    的 batch 当作候选；其它状态必须显式拒绝或延后
- 这份 contract 仍然只是 verifier prep：
  - 它不改 `shadow-batch-v1` public inputs
  - 它也还不直接验证 wallet `EIP-712` signature 或 `ED25519` order
    signature；那仍是后续 verifier/prover worker 的职责

为什么不先收紧成 gapless：

- 当前 API truth 就不是 gapless：
  [`AdvanceSessionNonce`](/Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go)
  只要求 `last_order_nonce < nonce`
- `NONCE_ADVANCED` 发生在 order freeze / Kafka publish 之前，后续失败不会回滚
  nonce；因此它代表的是“accepted auth attempt floor”，不是“最终成功落成的
  order count”
- 即使把 nonce 改成 gapless，也仍然不能跳过“每个 order auth 都要可证明”
  这件事；gapless 只会增加 runtime / migration rewrite，却不消除 auth gadget
  或 proof-friendly signature 的需求

第一版 verifier lane 对 auth source 的边界也固定为：

- canonical baseline = `POST /api/v1/trading-keys/challenge` +
  `POST /api/v1/trading-keys` 产出的 V2 trading-key authorization
- deprecated `POST /api/v1/sessions` blank-vault compatibility rows 继续允许
  repo-local shadow / proof tooling 使用，但不应作为 verifier-gated batch 的
  canonical auth baseline
- 因此第一条 prover/implementation tranche 应先把 repo proof tooling 从 legacy
  `/api/v1/sessions` 迁出，或显式把 legacy auth contract 单列成非首选兼容层

### 4.2.3 Verifier-gated `FunnyRollupCore` acceptance boundary

当前 [`FunnyRollupCore.recordBatchMetadata(...)`](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
仍然保留为 metadata placeholder；
[`FunnyRollupCore.acceptVerifiedBatch(...)`](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
则是当前 repo 新落地的 Foundry-only verifier/state-root acceptance hook。

第一版 verifier-gated acceptance boundary 固定为：

- `FunnyRollupCore` 可以被 verifier gate 的只是一条 batch-level state
  transition
- onchain acceptance 继续只围绕当前已经稳定的 metadata / public-input
  surface：
  - `batch_id`
  - `batch_data_hash`
  - `prev_state_root`
  - `balances_root`
  - `orders_root`
  - `positions_funding_root`
  - `withdrawals_root`
  - `next_state_root`
- acceptance rule 必须是：
  - `batch_id` 连续
  - `prev_state_root` 等于合约当前 `latestAcceptedStateRoot`
  - target batch 的 auth proof 里所有 relevant nonce auth row 都必须是
    `JOINED`
  - 只要出现 `MISSING_TRADING_KEY_AUTHORIZED` 或
    `NON_VERIFIER_ELIGIBLE`，就必须在 state-root advancement 前直接拒绝
  - proof 证明 ordered `shadow-batch-v1` witness 从 `prev_state_root`
    deterministic 地导出这些 component roots 和 `next_state_root`
  - proof 同时证明上节固定下来的 monotonic-floor nonce/auth contract
- repo 当前已经补出一条 future acceptance worker 可直接消费的 code
  boundary：
  - [`BuildVerifierGateBatchContract(history, batch)`](/Users/zhangza/code/funnyoption/internal/rollup/verifier_contract.go)
    组合了：
    - stable `public_inputs`
    - current `l1_batch_metadata`
    - target-batch `auth_proof`
  - [`BuildVerifierStateRootAcceptanceContract(history, batch)`](/Users/zhangza/code/funnyoption/internal/rollup/verifier_contract.go)
    further narrows that boundary down to:
    - unchanged `public_inputs`
    - unchanged `l1_batch_metadata`
    - one acceptance-facing auth status projection
    - one stable `solidity_export` view that freezes the
      `FunnyRollupCore.acceptVerifiedBatch(...)` calldata contract:
      - exact argument order
      - exact struct field names and Solidity types
      - `AuthJoinStatus` enum ordinals
      - `0x`-prefixed `bytes32` material for the exported args
  - [`BuildVerifierArtifactBundle(history, batch)`](/Users/zhangza/code/funnyoption/internal/rollup/verifier_contract.go)
    now directly consumes that `solidity_export` and materializes the first
    deterministic prover/verifier artifact bundle:
    - unchanged acceptance contract
    - `authProofHash = keccak256(abi.encode(authStatuses))`
    - `verifierGateHash = keccak256(abi.encode(batchEncodingHash, publicInputs..., authProofHash))`
    - explicit `verifierPublicSignals = { batchEncodingHash, authProofHash,
      verifierGateHash }`
    - explicit inner `proofData = abi.encode(proofDataSchemaHash,
      proofTypeHash, batchEncodingHash, authProofHash, verifierGateHash,
      proofBytes)`
    - current placeholder lane sets:
      - `proofDataSchemaHash = keccak256("funny-rollup-proof-data-v1")`
      - `proofTypeHash = keccak256("funny-rollup-proof-placeholder-v1")`
      - `proofBytes = bytes("")`
    - explicit `verifierProof = abi.encode(proofSchemaHash,
      publicSignalsSchemaHash, verifierPublicSignals, proofData)`
    - one explicit verifier-facing export for
      [`IFunnyRollupBatchVerifier.verifyBatch(...)`](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol)
  - Foundry-only
    [`FunnyRollupCore.acceptVerifiedBatch(...)`](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
    now consumes that same shape in contract form:
    - metadata subset must match the public-input subset
    - target `batch_id` must already have matching
      `recordBatchMetadata(...)` state onchain; self-consistent calldata alone
      is no longer enough
    - every auth status must be `JOINED`
    - the verifier boundary is now one real Foundry verifier contract,
      [`FunnyRollupVerifier`](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol),
      behind the stable interface:
      - `batchEncodingHash`
      - `publicInputs`
      - `authProofHash`
      - `verifierGateHash`
      - schema-tagged `verifierProof` bytes carrying explicit public signals
    - `FunnyRollupVerifier` currently:
      - requires `batchEncodingHash == keccak256("shadow-batch-v1")`
      - recomputes `verifierGateHash` onchain from the supplied
        `VerifierContext`
      - decodes `proofSchemaHash + publicSignalsSchemaHash +
        {batchEncodingHash, authProofHash, verifierGateHash} + proofData`
      - rejects any proof whose exported public signals or inner `proofData`
        digests do not match the supplied `VerifierContext` / recomputed
        `verifierGateHash`
      - dispatches only on the fixed first real lane
        `proofTypeHash =
        keccak256("funny-rollup-proof-groth16-bn254-2x128-shadow-state-root-gate-v1")`
      - decodes `proofData-v1.proofBytes` as
        `abi.encode(uint256[2] a, uint256[2][2] b, uint256[2] c)`
      - derives six `BN254` field inputs from the unchanged outer public
        signals by splitting each `bytes32` into `hi/lo uint128` limbs in the
        fixed order
        `batchEncodingHashHi, batchEncodingHashLo, authProofHashHi,
        authProofHashLo, verifierGateHashHi, verifierGateHashLo`
      - forwards that tuple + six limbs into one Foundry-only fixed-vk
        [`FunnyRollupGroth16Backend`](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupGroth16Backend.sol)
        contract that performs a real `Groth16/BN254` pairing check
      - this backend is still intentionally narrow: it proves one fixed
        verifier lane / fixed-vk boundary, not a general prover pipeline or a
        production truth switch
    - `TASK-CHAIN-021` 现在把第一版真实 proving-system / proof-bytes
      contract 定死为：
      - `proofTypeHash = keccak256("funny-rollup-proof-groth16-bn254-2x128-shadow-state-root-gate-v1")`
      - first real verifier lane = fixed-vk `Groth16` on `BN254`
      - outer `verifierPublicSignals` 仍然保持
        `{batchEncodingHash, authProofHash, verifierGateHash}` 三个 `bytes32`
      - 为了安全映射到 `BN254` 标量域，真实 Groth16 public inputs 固定为每个
        `bytes32` 按大端拆成 `hi = uint128(x >> 128)` 和
        `lo = uint128(x)` 两个 limb，顺序固定为：
        `batchEncodingHashHi, batchEncodingHashLo, authProofHashHi,
        authProofHashLo, verifierGateHashHi, verifierGateHashLo`
      - `proofBytes` 继续留在 `proofData-v1.proofBytes`，contract 固定为：
        `abi.encode(uint256[2] a, uint256[2][2] b, uint256[2] c)`
      - `proofTypeHash` 标识的不是“Groth16”这个大类，而是完整 verifier-facing
        contract：
        - proving system + curve
        - `bytes32 -> field` lifting rule
        - exact circuit / verifying-key lane
        - `proofBytes` ABI codec
      - 因此第一版真实 prover 不需要先升级 `proofData-v2`
    - `proofData-v2` 只有在下面情况才必须出现：
      - verifier 必须显式接收会变化的 `vkHash` / `circuitHash` /
        aggregation-program id，而这些信息不能再只靠 `proofTypeHash`
        固定
      - `proofBytes` 不再能用一个单独 `bytes` blob 表示，必须拆成多个
        verifier-relevant payload
      - outer public-signal contract 本身要扩容到
        `{batchEncodingHash, authProofHash, verifierGateHash}` 之外
    - a later cryptographic verifier can replace the inner `proofData`
      implementation without reopening the current public-input boundary
  - `TASK-CHAIN-023` 再把这条 fixed-vk lane 从“一份共享 proof fixture”
    推进成“按 batch 的 deterministic proof artifact”：
    - 输入仍然只使用外层
      `{batchEncodingHash, authProofHash, verifierGateHash}`
    - outer proof/public-signal envelope 不变
    - `proofData-v1` 不变
    - 固定 `proofTypeHash` 不变
    - `shadow-batch-v1` public-input shape 不变
    - 变化的是 inner `proofBytes`：现在它会随着 batch-specific outer
      signals 变化，并由 repo 内 deterministic fixed-vk helper 直接生成
  - Go + Foundry tests now pin schema-hash / public-signal / `proofData` /
    `verifierProof` parity for more than one batch-specific artifact across
    the two runtimes
  - 这仍然没有把 repo 变成 `Mode B`：
    - 还没有 full prover
    - 还没有 full verifier
    - 还没有 production withdrawal rewrite

因此边界明确分成三层：

- 继续 metadata-only：
  - operator 对 batch artifact 的本地 materialization
  - `batch_data_hash / prev_state_root / next_state_root` 的离线审计用途
- 变成 acceptance-gated：
  - `latestAcceptedStateRoot` 的 onchain 前进
  - batch id / prev-root continuity
  - auth proof rows 全部为 `JOINED` 的 gate
  - 当前 public inputs 所代表的 component-root transition
- 仍然 shadow-only，不因为 verifier gate 就自动变成 production truth：
  - `withdrawals_root` 仍然只是 direct-vault request mirror，不是 canonical
    withdrawal claim-nullifier truth
  - `positions_funding_root.insurance_root` 仍然是 deterministic zero
    placeholder
  - `account_id` 仍然是当前 `user_id` mirror，不是最终 canonical
    `wallet + chain + vault` account contract
  - SQL/Kafka settlement、direct-vault claim、forced withdrawal runtime 都不在
    这条 acceptance boundary 里

### 4.3 Replay contract

`Mode B` 的 replay contract 固定为：

给定下面输入：

- `prev_state_root`
- L1 上已存在的 pending deposits / forced withdrawals
- one ordered durable batch input

必须 deterministic 地导出：

- `next_state_root`
- withdrawal-ready set
- nullifier updates
- public outputs hash

任何依赖下面这些隐式输入的执行都不算合法 replay：

- wall-clock timing
- SQL 当前快照
- Kafka 当前 offset
- mutable operator config
- ad hoc admin patch

### 4.4 Finality rule

batch 只有同时满足下面三件事才算 final：

- batch data 已发布到 L1
- proof 已通过 verifier
- `FunnyRollupCore` 已接受新的 `state_root`

如果缺任意一项：

- 该 batch 不是 `Mode B` final state
- operator 只能把它当 pending operator state

### 4.5 DA contract

`Mode B` first cut 的 DA 合同固定为：

- 所有重建状态与退出所需的 batch data 必须出现在 L1-native DA 上
- first truthful lane 以 `calldata` 为 canonical source
- offchain object storage 只允许当 cache
- 如果某批数据没有上 L1，就不能宣称用户可无 operator 配合重建退出数据

这也是为什么 first cut 不讨论：

- validium
- DAC
- external DA bridge

## 5. Withdrawal state machines

### 5.1 Slow withdrawal

canonical state machine：

```text
NONE
  -> REQUESTED
  -> INCLUDED_IN_PROVEN_BATCH
  -> READY_TO_CLAIM
  -> CLAIMED
```

contract：

- user 向 operator 提交 withdrawal intent
- sequencer 在 batch 中把余额转成 withdrawal leaf
- proof 通过后，`FunnyRollupVault` 接受该 leaf 的 claim proof
- contract 检查 `claim_nullifier`
- payout 给 user 指定 recipient

安全结论：

- slow withdrawal 是 canonical exit lane
- 没有 LP 也成立
- 没有 operator 的 `processClaim()` 自由裁量

### 5.2 Fast withdrawal

canonical state machine：

```text
NONE
  -> FAST_REQUESTED
  -> LP_FILLED
  -> INCLUDED_IN_PROVEN_BATCH
  -> LP_READY_TO_CLAIM
  -> LP_CLAIMED
```

contract：

- user 请求 fast withdrawal quote
- LP 接单并先行垫资
- sequencer 仍然必须创建同一个 canonical withdrawal leaf
- leaf 的 `beneficiary` 变成 LP
- `FunnyFastWithdrawPool` 或 LP route 先把钱打给 user
- proof finality 后 LP 再从 canonical withdrawal claim 回款

安全结论：

- fast withdrawal 只是 financing layer，不是新的 settlement truth
- 如果 LP 不可用，用户必须能无损 fallback 到 slow withdrawal
- fast lane 不能绕开 canonical withdrawal leaf

### 5.3 Forced withdrawal / freeze / escape hatch

canonical state machine：

```text
NONE
  -> FORCED_REQUESTED_ON_L1
  -> SATISFIED_IN_BATCH
  -> READY_TO_CLAIM

or

NONE
  -> FORCED_REQUESTED_ON_L1
  -> DEADLINE_MISSED
  -> FROZEN
  -> ESCAPE_CLAIMABLE
  -> ESCAPE_CLAIMED
```

contract：

- user 可以直接在 L1 提交 forced withdrawal request
- `FunnyRollupCore` 记录 request 与 deadline
- operator 必须在 deadline 前把它纳入 batch
- 若未满足，任何人都可触发 `freeze`
- `freeze` 后常规 state update 停止
- 用户可以基于最后一个 proven `state_root` 走 `escape hatch`

exit guarantees：

- 对 free collateral 与已生成的 canonical withdrawal leaf，用户必须能在无 operator 配合下退出
- forced withdrawal 是 censorship escape，不是 LP convenience

first-cut limit：

- unresolved open positions 不能在 first cut 中自动转换成 collateral withdrawal
- 如果 freeze 发生在未结算市场期间，仍需要单独的 emergency market resolution policy
- 这不阻止先实现余额级 forced withdrawal，但必须被明确记录为 residual risk

## 6. L1 contract boundary

### 6.1 `FunnyRollupCore`

责任：

- 存 verifier address
- 验证 proof 并更新 `state_root`
- 绑定 `batch_data_hash`
- 记录 pending deposit cursor
- 记录 forced withdrawal queue
- 维护 `frozen` / `escape_hatch_enabled`

不负责：

- order matching
- LP quoting
- oracle HTTP fetch
- frontend session auth

### 6.2 `FunnyRollupVault`

责任：

- 持有 collateral
- 接收 deposits
- 执行 slow / forced / escape claim payout
- 记录 withdrawal nullifier

不负责：

- proof generation
- matching
- operator accounting

### 6.3 `FunnyFastWithdrawPool`

责任：

- 接收 LP 资金
- 对 fast withdrawal 先行垫资
- 在 canonical withdrawal ready 后回款 LP

不负责：

- 替代 slow withdrawal
- 替代 forced withdrawal
- 定义 canonical settlement truth

## 7. Minimum proof obligations

`Mode B` first cut 至少要证明下面这些约束：

- `prev_state_root -> next_state_root` 转换合法
- L1 deposit 只能被消费一次
- total balance conservation 成立
- fee / insurance / funding delta 记账守恒
- every executed order / cancel 都对应一个合法授权输入
- nonce 单调前进，不能 replay
- resting order 的 reserved collateral 不能 double spend
- withdrawal leaf 创建、beneficiary assignment、claim nullifier 逻辑正确
- fast withdrawal 不能制造第二份可兑付 claim
- forced withdrawal 要么在 deadline 内被满足，要么进入可 freeze 的 onchain state

当前 [`VerifyOrderIntentSignature`](/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go) + SQL nonce 只能算 operator-side auth gate，不足以单独构成 proof truth。

因此一个明确的 follow-up contract 是：

- 要么把当前 order auth 升级成 proof-friendly key / signature scheme
- 要么引入专门的 signature-verification coprocessor / gadget boundary

但具体选型不属于这次任务。

## 8. Recommended first implementation tranche

推荐的第一条实现 tranche 不是“直接上全量 `Mode B`”，而是先做 shadow rollup lane：

- 新增 append-only `sequencer journal`
- 新增 durable batch input materialization
- 新增 deterministic state replayer
- 从现有执行结果导出 shadow：
  - `balances_root`
  - `orders_root`
  - `positions_funding_root`
  - `withdrawals_root`
- 定义 `FunnyRollupCore` / `FunnyRollupVault` storage contract 与 event contract
- 保持当前产品交易路径继续跑
- 明确标注这仍然 **不是** `Mode B`

为什么先做这条 tranche：

- 先把 replay contract 固定住
- 先把“哪些状态要进 root”固定住
- 先把 SQL/Kafka truth 降级成 shadow source
- 避免团队直接跳进 prover / verifier / full contract implementation

### 8.1 Shadow tranche 1 landed boundary

这条 tranche 在 repo 里落成的 boundary 必须明确保持“shadow-only”：

- 新表：
  - `rollup_shadow_journal_entries`
  - `rollup_shadow_batches`
- 当前 durable shadow inputs 只覆盖：
  - `matching` 写出的 `OrderAccepted / OrderCancelled / TradeMatched`
  - `chain` 写出的 `DepositCredited / WithdrawalRequested`
  - `settlement` 写出的:
    - `MarketResolved`
    - market-resolution-triggered `OrderCancelled`
    - `SettlementPayout`
- durable batch input 使用 one canonical `input_data` blob +
  `input_hash`
- `shadow-batch-v1` now has an explicit witness / public-input boundary:
  - witness = ordered typed entries + namespace-truth contract
  - public inputs = batch metadata hash + component roots + `prev/next_state_root`
- deterministic replay 只允许消费：
  - ordered batch `input_data`
  - append-only journal sequence
- API/auth order replay-protection truth now also enters the durable lane:
  - `NONCE_ADVANCED` is emitted when the API accepts a trading-key nonce
  - canonical V2 trading-key registration also emits `TRADING_KEY_AUTHORIZED`
    as a witness-only journal entry
  - replay rebuilds `orders_root.nonce_root` only from those durable entries
  - current leaf semantics are still the current operator truth:
    `(account_id, auth_key_id) -> next_nonce floor + scope + key_status`
  - `TRADING_KEY_AUTHORIZED` is replay no-op witness material, while each
    `NONCE_ADVANCED.payload.order_authorization` now binds the nonce advance to
    one exact V2 trading-key order authorization without widening public inputs
  - this closes the old `ZeroNonceRoot()` placeholder and the old “trust the
    API's prior signature check” gap, but it still does not land prover /
    verifier execution
- 当前 shadow payload 里的 `account_id` 先临时镜像现有 `user_id`
  - 这只是 transitional mirror
  - 还不是本文 3.1 节要求的最终 canonical account contract

这条 tranche **没有** 宣称下面这些东西已经成立：

- prover
- verifier
- L1 state update finality
- production withdrawal claim rewrite
- forced withdrawal / freeze runtime

因此当前 stage-1 root contract 要诚实表达为：

- `balances_root`
  - 对已捕获的 deposits / order reserves / matched trades /
    settlement payouts / queued withdrawals 做 deterministic mirror
- `orders_root`
  - 对 matching 与 market-resolution-triggered cancellation 后的
    open-order lifecycle 做 deterministic mirror
  - `nonce_root` 现在 truthfully 镜像 API/auth 已接受的 order nonce
    progression
  - 当前 leaf 仍然只是 `(account_id, auth_key_id)` keyed monotonic
    `next_nonce` floor，不代表最终 proof-friendly signature semantics
- `positions_funding_root`
  - `position_root` 会在 settlement payout 时消费 winning position
    quantity
  - `market_funding_root` 现在 truthfully 镜像 market settlement state
  - `insurance_root` 仍然是 deterministic zero root
- `withdrawals_root`
  - 目前镜像的是 direct-vault `queueWithdrawal` shadow request
  - 不是未来 `Mode B` canonical claim-nullifier truth
- L1 placeholder contract:
  - repo 里现在有两条明确分开的 `FunnyRollupCore` lane：
    - `recordBatchMetadata(...)` 继续只做 metadata placeholder
    - `acceptVerifiedBatch(...)` 作为 Foundry-only acceptance hook 记录
      accepted batch public-input roots
  - 它仍然不是 full verifier，也不代表 proof finality 已经存在

## 9. Migration stages from current FunnyOption

### Stage 0: current centralized direct-vault system

现状：

- current `FunnyVault` custody
- centralized matching
- SQL balances / positions / payouts
- Kafka event ordering
- operator claim / withdrawal flow

状态：

- 明确不是 `Mode B`

### Stage 1: shadow journal and shadow roots

- 引入 sequencer journal
- 引入 durable batch input
- 引入 shadow state roots
- 保持现有 settlement / payout / withdrawal 仍为生产 truth
- 当前 shadow source 只接：
  - API/auth `NONCE_ADVANCED`
  - `matching` order/trade lifecycle
  - confirmed chain deposits
  - confirmed direct-vault queued withdrawals
- settlement-driven additions now also shadow:
  - market resolution markers
  - market-resolution-triggered cancellations
  - settlement payout markers
- 当前仍未完成到 `Mode B` final truth 的部分：
  - proof-friendly signature / auth-key verification semantics
  - funding-index / insurance accounting
  - prover witness / proof / verifier artifacts

状态：

- 仍然不是 `Mode B`
- 但状态边界开始显式化

### Stage 1.5: first proof-lane prep without runtime rewrite

- 保留当前 monotonic-floor nonce contract，不做 gapless runtime rewrite
- repo proof tooling 应迁出 deprecated `/api/v1/sessions` blank-vault auth
  baseline，改走 canonical `POST /api/v1/trading-keys/challenge`,
  `POST /api/v1/trading-keys`, `GET /api/v1/trading-keys`
- 第一条 prover tranche 需要补一条 narrow auth witness lane：
  - 它要把 `NONCE_ADVANCED` 与对应的 trading-key order authorization 绑定起来
  - 它不能改写当前 `shadow-batch-v1` public inputs
  - 它也不能把 production withdrawal claim rewrite 一起带进来
- repo 当前落地的 narrow lane 是：
  - `TRADING_KEY_AUTHORIZED` witness-only entry for canonical V2 registration
  - `NONCE_ADVANCED.payload.order_authorization` for the exact order signature
  - `authorization_ref` as the stable join key between the two
- repo 当前也已经把这条 lane 组织成 explicit verifier-prep contract：
  - `BuildVerifierAuthProofContract(history, batch)` materializes one target-
    batch auth-proof view with `JOINED / MISSING_TRADING_KEY_AUTHORIZED /
    NON_VERIFIER_ELIGIBLE` status
  - `BuildVerifierGateBatchContract(history, batch)` packages that auth-proof
    view next to the unchanged batch public inputs / L1 metadata
  - `BuildVerifierStateRootAcceptanceContract(history, batch)` then projects
    only the acceptance-facing auth status lane needed by the current
    `FunnyRollupCore` hook
- repo 当前还补了一个最小 Foundry-only acceptance hook：
  - `FunnyRollupCore.acceptVerifiedBatch(...)` only advances
    `latestAcceptedStateRoot` when:
    - batch continuity holds
    - metadata subset matches
    - the target batch was previously anchored by
      `recordBatchMetadata(...)` with matching metadata
    - every auth row is `JOINED`
    - one real verifier-facing interface call returns success
- `BuildVerifierStateRootAcceptanceContract(history, batch)` 现在还会导出
  一个 stable `solidity_export` artifact，避免下一条 verifier/prover worker
  再去猜 `acceptVerifiedBatch(...)` 的 enum 编码、`bytes32` 规范化或参数顺序
- `BuildVerifierArtifactBundle(history, batch)` 现在会直接消费这个
  `solidity_export`，产出 repo 第一条 deterministic prover/verifier artifact：
  - unchanged acceptance contract
  - deterministic `authProofHash`
  - deterministic `verifierGateHash`
  - deterministic `verifierPublicSignals = { batchEncodingHash,
    authProofHash, verifierGateHash }`
  - deterministic `proofData = abi.encode(proofDataSchemaHash, proofTypeHash,
    batchEncodingHash, authProofHash, verifierGateHash, proofBytes)`
  - fixed first real `proofData-v1` lane with
    `proofDataSchemaHash = keccak256("funny-rollup-proof-data-v1")`,
    `proofTypeHash =
    keccak256("funny-rollup-proof-groth16-bn254-2x128-shadow-state-root-gate-v1")`,
    and non-empty
    `proofBytes = abi.encode(uint256[2] a, uint256[2][2] b, uint256[2] c)`
  - the verifier keeps the outer proof/public-signal envelope unchanged and
    internally lifts the three outer `bytes32` public signals into six
    `2 x uint128` `Groth16/BN254` field inputs
  - the repo now includes one Foundry-only fixed-vk
    `FunnyRollupGroth16Backend` contract plus one deterministic Go-side
    batch-specific artifact generator for:
    - limb splitting
    - proof-bytes codec
    - expected verifier verdict
    - per-batch proof-bytes variation derived from actual outer signals
  - `proofData-v2` is still unnecessary for this first fixed-vk lane; it only
    becomes necessary once vk/circuit/aggregation metadata must travel as
    separate verifier calldata
  - deterministic `verifierProof = abi.encode(proofSchemaHash,
    publicSignalsSchemaHash, verifierPublicSignals, proofData)`
  - verifier-facing `IFunnyRollupBatchVerifier.verifyBatch(context, proof)`
    calldata contract
- repo 当前已经用同一条 fixed-vk lane 证明 Go / Solidity 对
  batch-specific proof/public-signal schema、`proofData`、`verifierProof`、
  limb splitting、proofBytes 和 verifier verdict 的 parity
- 已经 materialized 的 shadow batches 仍然只是 shadow/debug artifacts；没有
  上述 auth witness 的 batch 不应直接进入 verifier-gated acceptance

状态：

- 这是第一条 prover/verifier artifact lane，不是 `Mode B`
- 它的目标只是让 verifier gate 的
  nonce/auth/state-root/digest boundary 不再摇摆

### Stage 2: proof-bound state update and slow withdrawal claim

- 上 `FunnyRollupCore`
- 上 `FunnyRollupVault`
- deposits 进入 onchain queue / cursor
- slow withdrawal 变成 canonical claim path
- operator `processClaim()` 不再是最终提款事实

状态：

- 可以开始接近 proof-verified settlement
- 但如果 forced withdrawal / freeze / escape 还没落地，仍不能宣称完整 exit guarantee

### Stage 3: forced withdrawal + freeze + escape hatch

- 上 forced withdrawal queue
- 上 freeze rule
- 上 escape hatch
- 把 censorship resistance 变成 explicit contract

状态：

- 只有到这一步，FunnyOption 才能诚实地宣称自己具备 `Mode B` 核心退出保证

### Stage 4: fast withdrawal LP lane

- 上 `FunnyFastWithdrawPool`
- LP 垫资与 canonical withdrawal leaf 对接

状态：

- 完成 slow / fast / forced 三条 withdrawal lane

### Stage 5: oracle / market-resolution truth hardening

- 把 oracle input 从“operator 拉 HTTP 后自己说结果”升级成更可验证的输入边界
- 可选：
  - signed attestation
  - onchain readable feed
  - attested resolver input

状态：

- 这一步不影响 `Mode B` 的资金与退出安全定义
- 但会影响市场 outcome trust assumption 的强弱

## 10. What can remain operator-run vs what must be replaced

### 10.1 Can remain operator-run

- `internal/api`
  - auth、challenge issuance、read API
- `internal/matching`
  - sequencer / order book / match engine
- `internal/oracle`
  - fetch / normalize / evidence snapshot
- `internal/ws`
  - fanout
- admin surface
  - market operations / review tools
- fast withdrawal LP quoting

这些服务在 `Mode B` 下仍然可以是 operator-run，因为它们主要承担：

- intake
- policy
- orchestration
- UX

### 10.2 Must stop being canonical truth

- `account_balances`
- `freeze_records`
- `positions`
- `settlement_payouts`
- `wallet_sessions.last_order_nonce`
- `chain_withdrawals`
- Kafka `order.command / trade.matched / market.event` 作为最终 settlement truth
- `FunnyVault.processClaim()` 作为 operator final payout gate

更直白地说：

- 当前 `internal/account`、`internal/settlement`、`internal/chain` 可以继续存在
- 但它们必须降级为 operator cache、indexer、orchestrator
- 最终 truth 必须迁移到：
  - L1 deposits
  - published batch data
  - verified state roots
  - withdrawal nullifiers
  - forced-withdrawal / freeze contract state

## 11. Rejected options

- validium first cut
  - 拒绝，因为用户无法仅依赖 L1 重建退出数据
- external DA first cut
  - 拒绝，因为会把退出保证外包给新的信任层
- “SQL/Kafka 继续当真，只给 daily snapshot 做 proof”
  - 拒绝，因为这不改变真实 settlement boundary
- “fast withdrawal 直接替代 canonical withdrawal”
  - 拒绝，因为 fast lane 只是 LP financing
- “继续用 operator claim 代替 canonical withdrawal claim”
  - 拒绝，因为这无法提供无许可退出
- “当前 `ED25519 + SQL nonce` 就足够称为 proof-verified auth”
  - 拒绝，因为它还停留在 operator-side verification
- “先把 nonce 改成 gapless，再说 proof”
  - 拒绝，因为当前 durable truth 里 `NONCE_ADVANCED` 表示的是 monotonic
    auth floor 前进，而且它可能先于 freeze / publish 失败路径落盘；gapless 会
    触发 runtime rewrite，却仍然不能替代 proof-side auth verification
- “沿用当前 `shadow-batch-v1` 输入，不额外补 auth witness 就直接做
  verifier-gated acceptance”
  - 拒绝，因为现有 durable batch input 只 truthfully 记录了 nonce floor 和
    order lifecycle，还没有把 canonical trading-key auth 证据绑定进 proof
    contract

## 12. Residual risks and explicit out-of-scope

- prover system selection 还没定
- verifier circuit shape 还没定
- proof-friendly auth key / signature path 还没定
- unresolved positions 的 freeze-time emergency resolution 还没定
- oracle correctness 仍然可能依赖 operator-collected input，除非后续升级输入边界
- funding / insurance namespace 在当前二元市场 first cut 里大概率先置零

但这些未定项不影响这次任务形成一个明确的 architecture contract：

- current FunnyOption is not yet `Mode B`
- `Mode B` = `ZK-Rollup` only
- withdrawal lanes = slow + fast + forced
- state truth = roots, batch data, nullifiers, L1 queues
- migration = staged replacement，不是一次性重写
