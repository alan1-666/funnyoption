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
  proof tooling such as local lifecycle and staging concurrency scripts
- `POST /api/v1/trading-keys/challenge` plus `POST /api/v1/trading-keys`
  remains the canonical V2 browser registration flow

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
