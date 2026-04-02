# SQL Schema Notes

`/Users/zhangza/code/funnyoption/migrations/001_init.sql` is the first cut of the PostgreSQL core schema.

`/Users/zhangza/code/funnyoption/migrations/002_ownership.sql` is the follow-up grant/ownership migration for the `funnyoption` app role.

`/Users/zhangza/code/funnyoption/migrations/003_wallet_sessions_and_deposits.sql` adds session authorization and direct-vault deposit mirrors.

`/Users/zhangza/code/funnyoption/migrations/004_account_balance_events.sql` adds idempotent external balance credit tracking.

`/Users/zhangza/code/funnyoption/migrations/005_chain_transaction_queue.sql` hardens the claim queue with retry metadata.

`/Users/zhangza/code/funnyoption/migrations/006_chain_withdrawals.sql` adds on-chain withdrawal queue mirrors.

## Trading domain

- `markets`: market master data and lifecycle state
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

## Wallet and session domain

- `wallet_sessions`: wallet-signed browser session authorization records

## Current design principles

- snapshots live in `orders / positions / account_balances`
- immutable evidence lives in `trades / ledger_entries / ledger_postings / settlement_payouts`
- replay and reconciliation should prefer immutable evidence over mutable snapshots
- direct deposit mode should use on-chain vault custody and mirror confirmed deposit / withdrawal events into PostgreSQL
