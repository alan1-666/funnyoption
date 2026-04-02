# Direct Deposit and Session Key V1

## V1 decision

Funnyoption V1 does **not** implement StarkEx, SHARP, or STARK proof batching.

Funnyoption V1 adopts the following model instead:

- user funds are deposited directly from the frontend into a BSC vault contract
- the backend runs a centralized matching and balance engine
- users sign trading actions with a browser-local session key
- the wallet only pops up for:
  - first session authorization
  - on-chain deposit
  - on-chain withdrawal / claim

This keeps the UX close to Stark-style systems without introducing prover infrastructure.

## Why this is better than centralized deposit wallets

Compared with a per-user centralized wallet model:

- no per-user deposit address allocation
- no internal transfer address handling
- no collection / sweeping jobs
- no hot-wallet credit ambiguity
- users can inspect vault inflow directly on-chain

The backend still remains centralized for:

- order validation
- matching
- risk checks
- account snapshots
- market resolution

The trust boundary becomes:

- custody path: on-chain vault
- trading and state progression: backend operator

## First connection and session authorization

### Flow

1. User connects MetaMask
2. Browser generates a local session keypair
3. Wallet signs an off-chain authorization message once
4. Backend verifies the wallet signature and stores an active session grant
5. Later order / cancel actions are signed only with the session private key

### Why not derive a private key from the wallet signature

V1 should **not** derive a raw signing key from the wallet signature.

Instead:

- browser generates a random session keypair
- wallet signs an authorization granting that session key limited trading rights

This is easier to rotate, revoke, and audit.

### Session grant payload

The wallet-signed payload should contain:

- `wallet_address`
- `session_public_key`
- `scope`
  - `TRADE`
- `chain_id`
  - BSC testnet / mainnet
- `issued_at`
- `expires_at`
- `nonce`

### Suggested wallet message

```text
FunnyOption Session Authorization

wallet: 0x...
session_public_key: 0x...
scope: TRADE
chain_id: 97
issued_at: 1711972800000
expires_at: 1712059200000
nonce: sess_123456
```

## Deposit flow

1. User chooses `Deposit` in frontend
2. Frontend calls vault contract `deposit(amount)`
3. ERC20 moves from wallet to vault contract
4. `chain-service` listens for `Deposited` events
5. Backend writes a `chain_deposits` record
6. Backend credits `account_balances.available`
7. Backend writes append-only `ledger` evidence

### V1 rule

Deposit credit should happen only after:

- on-chain confirmation threshold is reached
- deposit event is persisted idempotently by `tx_hash + log_index`

## Trading flow

1. Frontend builds an order intent
2. Browser signs it with the session private key
3. API validates:
   - session status
   - session expiry
   - nonce / replay rules
   - order parameters
4. API calls `account` for pre-freeze
5. API publishes `order.command`
6. `matching` consumes and emits:
   - `order.event`
   - `trade.matched`
   - `position.changed`
   - `quote.depth`
   - `quote.ticker`

## Resolution and settlement flow

1. Admin resolves a market
2. API emits `market.event`
3. `settlement` computes winning payouts
4. `account` credits winners
5. `ledger` records immutable settlement entries
6. If V1 wants on-chain settlement anchoring, `chain-service` may later submit a settlement hash or payout batch reference

V1 does **not** require every fill to be settled on-chain individually.

## Withdrawal flow

### V1 simplified path

1. User requests withdrawal in frontend
2. Wallet signs a withdrawal authorization or directly submits a vault withdrawal transaction
3. Backend validates available balance
4. Backend creates a withdrawal record
5. If backend-controlled payout is used, `chain-service` submits payout from vault operator flow
6. If user-self-claim is used, vault contract releases funds to the wallet after backend marks the request claimable

For BSC V1, the recommended target is:

- **frontend direct deposit**
- **backend-approved claimable withdrawal**

That keeps custody simpler than a fully centralized hot wallet while avoiding proof-system complexity.

## New backend components implied by this model

- `wallet_sessions`
  - active browser session authorization records
- `chain_deposits`
  - on-chain deposit event mirror
- `chain-service`
  - BSC event listener and vault transaction submitter

## Security rules

- session keys must have expiry
- session keys must be revocable
- session private keys should not be stored as plain long-lived wallet substitutes
- use IndexedDB or another browser-local store, not raw reusable global secrets
- backend must enforce per-session nonce / replay control
- deposits must be idempotent by transaction identity
- backend balance snapshots must always be reconcilable to ledger evidence

## What stays out of scope for V1

- StarkEx key derivation
- STARK proof batching
- state trees
- forced withdrawal / escape hatch
- shared prover infrastructure
- data availability committee

Those belong to a later Mode-B style architecture, not the first production MVP.
