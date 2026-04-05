# SQL Schema Notes

`/Users/zhangza/code/funnyoption/migrations/001_init.sql` is the first cut of the PostgreSQL core schema.

`/Users/zhangza/code/funnyoption/migrations/002_ownership.sql` is the follow-up grant/ownership migration for the `funnyoption` app role.

`/Users/zhangza/code/funnyoption/migrations/003_wallet_sessions_and_deposits.sql` adds session authorization and direct-vault deposit mirrors.

`/Users/zhangza/code/funnyoption/migrations/004_account_balance_events.sql` adds idempotent external balance credit tracking.

`/Users/zhangza/code/funnyoption/migrations/005_chain_transaction_queue.sql` hardens the claim queue with retry metadata.

`/Users/zhangza/code/funnyoption/migrations/006_chain_withdrawals.sql` adds on-chain withdrawal queue mirrors.

`/Users/zhangza/code/funnyoption/migrations/007_market_taxonomy_and_options.sql` adds formal market categories plus per-market option-set JSON storage.

`/Users/zhangza/code/funnyoption/migrations/008_user_profiles.sql` adds user profile display metadata.

`/Users/zhangza/code/funnyoption/migrations/009_chain_listener_cursors.sql` adds a persisted restart cursor for vault event scans.

`/Users/zhangza/code/funnyoption/migrations/010_chain_deposits_tx_hash_width_repair.sql` reconciles reused local `chain_deposits.tx_hash` width drift back to the repo truth.

`/Users/zhangza/code/funnyoption/migrations/011_trading_key_challenges.sql` adds one-time V2 trading-key challenge storage.

`/Users/zhangza/code/funnyoption/migrations/012_wallet_sessions_vault_scope.sql` adds durable `vault_address` scope to the `wallet_sessions` compatibility carrier.

`/Users/zhangza/code/funnyoption/migrations/013_wallet_sessions_vault_key_uniqueness.sql` replaces the legacy wallet/public-key uniqueness rule with durable `wallet + chain + vault + public key` uniqueness.

## Trading domain

- `markets`: market master data and lifecycle state
- `market_categories`: canonical market taxonomy such as `加密 / 体育`
- `market_option_sets`: one JSON option schema per market
- `market_resolutions`: one row per market resolution workflow
- `orders`: accepted orders and final order state
- `trades`: immutable fills emitted by matching
- `positions`: current user position snapshot by market + outcome

## Market lifecycle runtime contract

- durable sources:
  - `markets.status`
  - `markets.close_at`
  - `markets.resolve_at`
  - `market_resolutions.status`
- runtime-effective market status is:
  - `OPEN` only when stored `markets.status = OPEN` and `close_at` is unset or still in the future
  - `CLOSED` when stored `markets.status = CLOSED`, or when stored `markets.status = OPEN` but `now >= close_at`
  - `RESOLVED` only after settlement flips `markets.status = RESOLVED`
- practical consequence:
  - `close_at` is the real trading boundary for ingress and matching runtime
  - once `now >= close_at`, still-resting `LIMIT` orders on that market are proactively cancelled through the matching/order-event lane rather than remaining only as inert in-memory book state
  - `resolve_at` remains the canonical auto-resolution timestamp only for oracle markets
  - non-oracle markets past `close_at` stay truthfully `CLOSED` and await manual resolution
- this contract is derived from existing durable columns; no extra migration is required for the first truthful runtime slice

## Account domain

- `account_balances`: mutable available/frozen balance snapshot
- `freeze_records`: pre-trade freeze records keyed by `freeze_id`
- `account_balance_events`: idempotent external balance delta references such as deposits and withdrawals

## Ledger domain

- `ledger_entries`: append-only business entries
- `ledger_postings`: double-entry postings under each entry

## Settlement and chain domain

- `settlement_payouts`: resolved winner payouts
- `chain_transactions`: deposit / withdraw / settlement on-chain references
- `chain_deposits`: direct frontend-to-vault deposit mirror keyed by transaction event identity
- `chain_withdrawals`: mirrored `queueWithdrawal` events keyed by transaction event identity
- `chain_listener_cursors`: persisted `next_block` checkpoint for restart-safe vault log scans

## `chain_deposits` width notes

- repo truth:
  - `deposit_id = VARCHAR(64)`
  - `tx_hash = VARCHAR(128)`
- observed legacy local drift from reused databases:
  - `deposit_id = VARCHAR(64)`
  - `tx_hash = VARCHAR(64)`
- current listener-driven local proof still works on that drifted local shape because the chain listener stores:
  - deterministic deposit ids that fit within `VARCHAR(64)`
  - normalized lowercase tx hashes without the `0x` prefix, which fit within `VARCHAR(64)`
- repo-local repair path:
  - [`migrations/010_chain_deposits_tx_hash_width_repair.sql`](/Users/zhangza/code/funnyoption/migrations/010_chain_deposits_tx_hash_width_repair.sql)
  - [`docs/operations/local-chain-deposits-schema-repair.md`](/Users/zhangza/code/funnyoption/docs/operations/local-chain-deposits-schema-repair.md)

## Wallet and session domain

- `wallet_sessions`: wallet-signed browser session authorization records
- `trading_key_challenges`: one-time wallet auth challenges for V2 trading-key registration

## Auth V2 compatibility contract

Until a dedicated rename migration lands, the existing `wallet_sessions` table
is the persistence slot for V2 trading-key authorization.

Current-field to V2-semantic mapping:

- `session_id` -> `trading_key_id`
- `session_public_key` -> `trading_public_key`
- `scope` -> trading scope such as `TRADE`
- `chain_id` -> target EVM chain id from the EIP-712 domain
- `vault_address` -> durable target vault scope for canonical trading-key rows
- `session_nonce` -> consumed wallet auth challenge
- `last_order_nonce` -> last accepted order nonce for that trading key
- `status` -> `ACTIVE | REVOKED | ROTATED`
- `issued_at` -> wallet authorization acceptance time
- `expires_at` -> trading key expiry; `0` means durable until revoke / rotate
- `revoked_at` -> revoke or rotate time

V2 rules:

- one active trading key per `wallet_address + chain_id + vault_address`
- canonical trading-key row uniqueness is
  `wallet_address + chain_id + vault_address + session_public_key`
- public auth flows must stop treating client-provided `user_id` as the source
  of truth
- deposit and withdrawal attribution should use the durable wallet-to-user
  binding, not the presence of a currently active browser-local key
- the current durable wallet binding can be sourced from
  `user_profiles.wallet_address`

Current runtime truth:

- canonical trading-key rows in `wallet_sessions` now durably persist
  `vault_address`
- active-key rotation and active-key listing are now scoped by
  `wallet_address + chain_id + vault_address`
- canonical trading-key rows can now reuse the same `session_public_key`
  across two vaults on the same `wallet_address + chain_id` because durable
  uniqueness now includes `vault_address`
- browser restore can read back and disambiguate remote active keys by vault
  instead of depending on a single-vault-per-environment assumption
- deprecated `/api/v1/sessions` compatibility rows still carry blank
  `vault_address`, so they stay in their own legacy blank-vault scope because
  the old session-grant contract never signed a vault value

Temporary route compatibility:

- `POST /api/v1/sessions` remains as a deprecated compatibility route for repo
  legacy compatibility tooling only
- `POST /api/v1/trading-keys/challenge`, `POST /api/v1/trading-keys`, and
  `GET /api/v1/trading-keys` are the canonical V2 registration / readback
  routes for verifier-eligible proof-tooling paths

Follow-up schema work that may be implemented later, but is not required in
this narrow runtime slice:

- rename `wallet_sessions` to a trading-key-specific name
- add `key_scheme`
- add `wallet_sig_standard`
- add `replaced_by_session_id`
- add `auth_version`

The first runtime slice now stores one-time auth challenges in
`trading_key_challenges` with:

- uniqueness
- expiry
- single-use consumption

## Current design principles

- snapshots live in `orders / positions / account_balances`
- immutable evidence lives in `trades / ledger_entries / ledger_postings / settlement_payouts`
- replay and reconciliation should prefer immutable evidence over mutable snapshots
- direct deposit mode should use on-chain vault custody and mirror confirmed deposit / withdrawal events into PostgreSQL

## Mode B rollup boundary note

The target `Mode B` design in
[`docs/architecture/mode-b-zk-rollup.md`](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
explicitly does **not** treat the current SQL schema as canonical rollup truth.

When the repo is still in the current direct-vault centralized mode:

- `account_balances`
- `freeze_records`
- `orders`
- `positions`
- `settlement_payouts`
- `wallet_sessions.last_order_nonce`
- `chain_withdrawals`

are all part of the operator-run truth boundary.

When `Mode B` is implemented, those tables should be treated as:

- operator caches
- read models
- reconciliation mirrors
- operational indexing surfaces

The future canonical settlement truth must move to:

- L1 deposit queue state
- L1-published batch data
- verified `state_root` updates
- withdrawal nullifiers
- forced-withdrawal / freeze / escape-hatch contract state

Practical schema implication:

- do not overload the current mutable SQL tables and call that `Mode B`
- if the rollup lane adds local persistence, add dedicated rollup artifacts such as:
  - sequencer journal storage
  - durable batch input storage
  - proven root / batch metadata storage
  - forced-withdrawal request mirrors

## Shadow rollup tranche 1 artifacts

`migrations/014_rollup_shadow_lane.sql` lands the first explicit shadow-rollup
storage boundary.

New tables:

- `rollup_shadow_journal_entries`
  - append-only ordered shadow inputs
  - source uniqueness is `(entry_type, source_type, source_ref)` so the shadow
    lane can retry without relying on Kafka offsets as replay truth
  - current captured inputs are:
    - `TRADING_KEY_AUTHORIZED`
    - `NONCE_ADVANCED`
    - `ORDER_ACCEPTED`
    - `ORDER_CANCELLED`
    - `TRADE_MATCHED`
    - `DEPOSIT_CREDITED`
    - `WITHDRAWAL_REQUESTED`
    - `MARKET_RESOLVED`
    - `SETTLEMENT_PAYOUT`
- `rollup_shadow_batches`
  - durable materialized batch input artifact
  - stores the canonical `input_data` blob, `input_hash`, `prev_state_root`,
    component roots, and final `state_root`
  - `input_hash` is the current repo-local `shadow-batch-v1` `batch_data_hash`
    surface for future prover / L1 metadata consumers
  - replay should consume `input_data`, not current mutable SQL snapshots

Current `shadow-batch-v1` contract:

- witness:
  - ordered typed `entries[]` from `rollup_shadow_batches.input_data`
  - one explicit namespace-truth contract
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

Truthfulness note:

- these tables are still `shadow-only`
- current shadow payloads temporarily mirror `account_id` from existing
  `user_id`; they do not yet implement the final `wallet + chain + vault`
  canonical account contract
- current `orders_root.nonce_root` now truthfully mirrors API/auth accepted
  order-nonce advances from durable `NONCE_ADVANCED` entries
  - the leaf key is the current shadow `(account_id, auth_key_id)` mirror
  - the leaf value is a monotonic `next_nonce` floor plus mirrored
    `scope/key_status`
  - this is still not proof-ready final auth semantics because the current API
    gate allows nonce gaps and still relies on operator-side signature checks
  - `TASK-CHAIN-012` fixes the first proof-lane decision here:
    - keep this monotonic-floor nonce contract for tranche 1
    - do **not** force a gapless SQL/runtime rewrite first
    - require the future prover lane to add auth evidence for each
      `NONCE_ADVANCED` transition instead of treating the API's prior signature
      check as proof truth
- current settlement-phase shadowing now includes:
  - market resolution markers
  - market-resolution-triggered order cancellations
  - settlement payout markers
- current `positions_funding_root` no longer keeps `market_funding_root` at a
  full zero root once a market is resolved; it truthfully mirrors market
  settlement state while leaving funding index fixed at `0`
- current `insurance_root` remains a deterministic zero placeholder
- they do **not** replace current production truth in:
  - `account_balances`
  - `orders`
  - `positions`
  - `settlement_payouts`
  - `chain_withdrawals`
- the current tranche intentionally does **not** make prover output, verifier
  acceptance, or withdrawal claim nullifiers part of production truth

First proof-lane storage / migration consequence:

- existing `rollup_shadow_batches.input_data` is enough to replay the current
  monotonic nonce floor, but it is not yet enough to prove final auth truth on
  its own
- before verifier-gated batches start, the repo needs one narrow auth witness
  artifact that binds canonical V2 trading-key authorization to the accepted
  order nonce lane without reopening the stable public-input shape
- current landed artifact is:
  - `TRADING_KEY_AUTHORIZED` witness-only journal entries for canonical V2
    trading-key registration
  - `NONCE_ADVANCED.payload.order_authorization` carrying the exact order
    intent message / hash / signature plus `authorization_ref`
  - `authorization_ref = trading_key_id:challenge` as the durable join between
    the key-authorization witness and the order-nonce witness
- current verifier-prep contract is now explicit in code:
  - normalized binding tuple =
    `authorization_ref + trading_key_id + account_id + wallet_address +
    chain_id + vault_address + trading_public_key + trading_key_scheme +
    scope + key_status`
  - [`BuildVerifierAuthProofContract(history, batch)`](/Users/zhangza/code/funnyoption/internal/rollup/verifier_contract.go)
    classifies each target-batch nonce auth row as:
    - `JOINED`
    - `MISSING_TRADING_KEY_AUTHORIZED`
    - `NON_VERIFIER_ELIGIBLE`
  - [`BuildVerifierGateBatchContract(history, batch)`](/Users/zhangza/code/funnyoption/internal/rollup/verifier_contract.go)
    then packages that auth-proof view next to the unchanged batch public
    inputs / metadata surface
  - [`BuildVerifierStateRootAcceptanceContract(history, batch)`](/Users/zhangza/code/funnyoption/internal/rollup/verifier_contract.go)
    projects that same boundary down to the minimal acceptance-facing shape:
    - unchanged `public_inputs`
    - unchanged `l1_batch_metadata`
    - target-batch auth row statuses for the `JOINED` gate
    - one stable `solidity_export` payload that fixes:
      - `FunnyRollupCore.acceptVerifiedBatch(...)` argument order
      - struct field names and Solidity types
      - `AuthJoinStatus` enum ordinals
      - normalized `0x`-prefixed `bytes32` calldata values
  - [`BuildVerifierArtifactBundle(history, batch)`](/Users/zhangza/code/funnyoption/internal/rollup/verifier_contract.go)
    now directly consumes that `solidity_export` and materializes the first
    deterministic prover/verifier artifact contract:
    - unchanged acceptance contract
    - `authProofHash = keccak256(abi.encode(authStatuses))`
    - `verifierGateHash = keccak256(abi.encode(batchEncodingHash, publicInputs..., authProofHash))`
    - explicit `verifierPublicSignals = { batchEncodingHash, authProofHash,
      verifierGateHash }`
    - explicit inner `proofData = abi.encode(proofDataSchemaHash,
      proofTypeHash, batchEncodingHash, authProofHash, verifierGateHash,
      proofBytes)`
    - current placeholder lane sets
      `proofDataSchemaHash = keccak256("funny-rollup-proof-data-v1")`,
      `proofTypeHash = keccak256("funny-rollup-proof-placeholder-v1")`, and
      `proofBytes = bytes("")`
    - explicit `verifierProof = abi.encode(proofSchemaHash,
      publicSignalsSchemaHash, verifierPublicSignals, proofData)`
    - verifier-facing
      [`IFunnyRollupBatchVerifier.verifyBatch(context, proof)`](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol)
      calldata
  - Foundry-only
    [`FunnyRollupCore.acceptVerifiedBatch(...)`](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
    now:
    - requires the target batch to have been previously anchored through
      `recordBatchMetadata(...)`
    - rejects any batch whose projected auth status contains
      `MISSING_TRADING_KEY_AUTHORIZED` or `NON_VERIFIER_ELIGIBLE` before
      verifier verdict / `latestAcceptedStateRoot` advancement
    - passes a full verifier-facing context contract, not just one bare hash:
      - `batchEncodingHash`
      - `publicInputs`
      - `authProofHash`
      - `verifierGateHash`
    - the first concrete
      [`FunnyRollupVerifier`](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol)
      implementation recomputes/constrains `verifierGateHash` onchain, decodes
      the explicit proof/public-signal schema plus `proofData-v1`, dispatches
      on the fixed first real
      `proofTypeHash = keccak256("funny-rollup-proof-groth16-bn254-2x128-shadow-state-root-gate-v1")`,
      decodes `proofData-v1.proofBytes` as
      `abi.encode(uint256[2] a, uint256[2][2] b, uint256[2] c)`, derives six
      `BN254` field inputs from the unchanged outer public signals, and calls
      one Foundry-only fixed-vk
      [`FunnyRollupGroth16Backend`](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupGroth16Backend.sol)
      verifier contract
  - `TASK-CHAIN-021` fixes the first real proof contract without reopening the
    existing verifier-facing boundary:
    - first real `proofTypeHash =
      keccak256("funny-rollup-proof-groth16-bn254-2x128-shadow-state-root-gate-v1")`
    - `proofTypeHash` identifies the whole verifier-facing contract, not only
      the proving-family label:
      - proving system + curve
      - `bytes32` public-signal lifting rule
      - exact circuit / verifying-key lane
      - `proofBytes` ABI codec
    - real prover output stays inside `proofData-v1.proofBytes` as
      `abi.encode(uint256[2] a, uint256[2][2] b, uint256[2] c)`
    - outer public signals stay
      `{batchEncodingHash, authProofHash, verifierGateHash}`
    - the first Groth16 backend should derive its field inputs by splitting
      each outer `bytes32` into `hi/lo uint128` limbs in fixed order
    - no `proofData-v2` is required for that first fixed-vk lane
    - a future `proofData-v2` is only required if verifier-relevant
      vk/circuit/aggregation metadata must travel separately from
      `proofTypeHash + proofBytes`
- Go and Foundry tests now pin one deterministic fixed-vk lane that produces
  batch-specific proof artifacts from actual outer
  `{batchEncodingHash, authProofHash, verifierGateHash}` signals while keeping
  the outer proof/public-signal envelope, `proofData-v1`, fixed
  `proofTypeHash`, limb splitting, proof-bytes codec, and verifier verdict
  parity aligned across both runtimes
- deprecated blank-vault `/api/v1/sessions` rows should remain shadow /
  compatibility-only; proof tooling should migrate to V2 trading-key rows
  before those batches are treated as verifier-eligible

Current replay contract:

- rebuild shadow roots from ordered `rollup_shadow_batches.input_data`
- if needed, materialize the next batch from
  `rollup_shadow_journal_entries.sequence_no`
- do not use:
  - live SQL balance snapshots
  - current order snapshot rows
  - Kafka consumer offsets
  - ad hoc operator patches
