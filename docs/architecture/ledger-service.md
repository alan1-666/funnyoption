# Ledger Service

`ledger` is the system backstop. Balances in `account` are mutable snapshots for fast trading reads, while the ledger remains the append-only evidence chain.

## Core rules

- every balance-changing business event must produce a ledger entry
- ledger entries are append-only; corrections happen through compensating entries
- debit must equal credit per asset inside one entry
- reconciliation compares ledger-derived liabilities with on-chain balances

## Service boundary

- `account`: current available/frozen balance view, optimized for pre-trade checks
- `ledger`: immutable journal, replay, reconciliation, and audit evidence

## Initial entry families

- `DEPOSIT`
- `WITHDRAW`
- `TRADE`
- `FEE`
- `TRANSFER`
- `SETTLEMENT`

## Posting model

- `TRADE` records both cash movement and position movement in one balanced entry
- `SETTLEMENT` records winner position burn plus payout cash in one balanced entry
- position assets are represented as `POSITION:<market_id>:<OUTCOME>`
