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

## Current design principles

- snapshots live in `orders / positions / account_balances`
- immutable evidence lives in `trades / ledger_entries / ledger_postings / settlement_payouts`
- replay and reconciliation should prefer immutable evidence over mutable snapshots
- direct deposit mode should use on-chain vault custody and mirror confirmed deposit / withdrawal events into PostgreSQL
