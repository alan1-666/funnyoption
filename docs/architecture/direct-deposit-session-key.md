# Direct Deposit and Trading Key Authorization V2

## V2 decision

FunnyOption V2 still does **not** implement StarkEx, SHARP, STARK proof
batching, state trees, or forced withdrawal.

FunnyOption V2 adopts the following model instead:

- user funds are still deposited directly from the frontend into a BSC vault
  contract
- the backend still runs centralized matching, balance, and settlement logic
- the auth model changes from a short-lived browser `session key` to a
  wallet-authorized browser-local `trading key`
- the wallet signs once during first authorization or later revoke / rotate
- later order placement is signed only by the browser-local trading key

This keeps the desired Stark-style UX without widening into prover
infrastructure.

## What "Stark-style" means here

For FunnyOption V2, "Stark-style" means:

- one EVM wallet signature to authorize a non-EVM local trading key
- later orders signed by that local trading key
- direct on-chain vault custody stays unchanged

It does **not** mean:

- Stark curve signatures in V2
- on-chain trading-key registration
- proof systems
- a state tree

The initial implementation target keeps the current off-chain signing algorithm:

- `ED25519` trading keypairs

That is a UX and auth-boundary change, not a proof-system change.

## Explicit decision: reject signature-derived deterministic trading keys

FunnyOption V2 does **not** accept deriving a trading private key from a wallet
signature.

The rejected model was:

- wallet signs one message
- browser deterministically derives the trading private key from the resulting
  signature bytes

That model is rejected because:

- wallet signature bytes are not a stable KDF input across `personal_sign`,
  `eth_signTypedData_v4`, hardware wallets, and smart-contract wallets
- reproducible recovery becomes wallet-implementation-dependent instead of
  protocol-dependent
- rotation and revoke are cleaner when the wallet authorizes a public key
  rather than becoming secret material itself
- storage loss should force explicit re-authorization, not silent re-derivation
  from a replayable signature artifact

The accepted V2 model is:

- browser generates a random trading keypair locally
- wallet signs a typed authorization for the trading public key
- backend stores the authorized public key and rejects any later order not
  signed by that key

## Auth objects

### Wallet identity

- one EVM wallet address on one target chain
- durable binding to one FunnyOption user account
- deposits and withdrawals are attributed by this wallet binding, not by local
  browser storage state

### Trading key challenge

- one-time backend challenge for wallet authorization or revoke
- server generated
- scoped to `wallet_address + chain_id + vault_address`
- expires after `5 minutes`
- consumed exactly once on success

### Trading key authorization

- one active trading key per `wallet_address + chain_id + vault_address`
- registering a different key for the same
  `wallet_address + chain_id + vault_address` rotates the old active key
- canonical storage uniqueness is
  `wallet_address + chain_id + vault_address + trading_public_key`
- reusing the same trading public key across two different vaults on the same
  wallet + chain is allowed because those are two independent vault-scoped
  authorizations
- default `scope = TRADE`
- default `key_expires_at = 0`
  - `0` means no automatic expiry
  - a key ends only by revoke or rotate

### Order intent

- per-order payload signed by the local trading private key
- verified against the active authorized trading public key
- replay-protected by per-key nonce and short expiry

## Wallet signature standard

Trading-key authorization uses:

- `eth_signTypedData_v4`
- `EIP-712`

Do **not** use `personal_sign` for V2 trading-key authorization.

## Domain and chain binding

The typed-data domain is:

```json
{
  "name": "FunnyOption Trading Authorization",
  "version": "2",
  "chainId": 97,
  "verifyingContract": "0xFunnyVaultAddress"
}
```

Rules:

- `chainId` must equal the connected wallet chain and the backend target chain
- `verifyingContract` must equal the active `FunnyVault` address for that
  environment
- `name` and `version` must match exactly
- a signature for one chain or vault address must be invalid on another chain
  or vault

## Authorization message format

Primary type:

```json
{
  "AuthorizeTradingKey": [
    { "name": "action", "type": "string" },
    { "name": "wallet", "type": "address" },
    { "name": "tradingPublicKey", "type": "bytes32" },
    { "name": "tradingKeyScheme", "type": "string" },
    { "name": "scope", "type": "string" },
    { "name": "challenge", "type": "bytes32" },
    { "name": "challengeExpiresAt", "type": "uint64" },
    { "name": "keyExpiresAt", "type": "uint64" }
  ]
}
```

Example message:

```json
{
  "action": "AUTHORIZE_TRADING_KEY",
  "wallet": "0x1c7d4b196cb0c7b01d743fbc6116a902379c7238",
  "tradingPublicKey": "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
  "tradingKeyScheme": "ED25519",
  "scope": "TRADE",
  "challenge": "0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a",
  "challengeExpiresAt": 1711973100000,
  "keyExpiresAt": 0
}
```

## First connection flow

1. User connects MetaMask and switches to the target chain.
2. Browser generates one local `ED25519` trading keypair.
3. Browser requests a one-time registration challenge.
4. Wallet signs the typed `AuthorizeTradingKey` message.
5. Browser sends the trading public key plus wallet signature to the API.
6. Backend verifies:
   - challenge exists
   - challenge is unused
   - challenge is unexpired
   - signer address matches `wallet_address`
   - typed-data domain matches target chain and vault
7. Backend resolves or creates the wallet-to-user binding.
8. Backend stores the new active trading key.
9. Browser stores the private key locally and later signs orders with it.

## Trading public key registration payload

`POST /api/v1/trading-keys/challenge`

```json
{
  "wallet_address": "0x1c7d4b196cb0c7b01d743fbc6116a902379c7238",
  "chain_id": 97,
  "vault_address": "0xFunnyVaultAddress"
}
```

Response:

```json
{
  "challenge_id": "tkc_01HTY5V1S8E9Q3P8W2V5K19J4P",
  "challenge": "0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a",
  "challenge_expires_at": 1711973100000
}
```

`POST /api/v1/trading-keys`

```json
{
  "wallet_address": "0x1c7d4b196cb0c7b01d743fbc6116a902379c7238",
  "chain_id": 97,
  "vault_address": "0xFunnyVaultAddress",
  "challenge_id": "tkc_01HTY5V1S8E9Q3P8W2V5K19J4P",
  "challenge": "0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a",
  "challenge_expires_at": 1711973100000,
  "trading_public_key": "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
  "trading_key_scheme": "ED25519",
  "scope": "TRADE",
  "key_expires_at": 0,
  "wallet_signature_standard": "EIP712_V4",
  "wallet_signature": "0x..."
}
```

Server-side contract:

- `user_id` is resolved server-side from the wallet binding
- client must not be the source of truth for public auth `user_id`
- `trading_key_id` is deterministic from wallet + chain + vault + public key
- if the same key is already active, registration is idempotent
- if a different key is already active for the same wallet, the old key becomes
  `ROTATED` and the new key becomes `ACTIVE`

Suggested response:

```json
{
  "user_id": 1001,
  "wallet_address": "0x1c7d4b196cb0c7b01d743fbc6116a902379c7238",
  "chain_id": 97,
  "vault_address": "0xFunnyVaultAddress",
  "trading_key_id": "tk_8bb731f6db073cf41f26ed1fdd2cb6b6",
  "trading_public_key": "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
  "trading_key_scheme": "ED25519",
  "scope": "TRADE",
  "status": "ACTIVE",
  "last_order_nonce": 0,
  "authorized_at": 1711972800000,
  "expires_at": 0,
  "revoked_at": 0
}
```

## Order signature payload

Each order is signed by the active trading key.

Signature scheme:

- `ED25519`
- sign the exact UTF-8 bytes of the canonical message below

Canonical message:

```text
FunnyOption Order Authorization V2

chain_id: 97
vault_address: 0xFunnyVaultAddress
trading_key_id: tk_8bb731f6db073cf41f26ed1fdd2cb6b6
market_id: 123
outcome: YES
side: BUY
order_type: LIMIT
time_in_force: GTC
price: 56
quantity: 100
client_order_id: cli_123
nonce: 7
issued_at: 1711972800000
expires_at: 1711973100000
```

HTTP payload:

```json
{
  "market_id": 123,
  "outcome": "YES",
  "side": "BUY",
  "type": "LIMIT",
  "time_in_force": "GTC",
  "price": 56,
  "quantity": 100,
  "client_order_id": "cli_123",
  "trading_key_id": "tk_8bb731f6db073cf41f26ed1fdd2cb6b6",
  "order_nonce": 7,
  "issued_at": 1711972800000,
  "expires_at": 1711973100000,
  "trading_signature_scheme": "ED25519",
  "trading_signature": "0x..."
}
```

Server verification contract:

- load the active trading-key record by `trading_key_id`
- rebuild the canonical message exactly
- verify `ED25519` signature against the stored public key
- reject if the key is not `ACTIVE`
- reject if the key is expired
- reject if `order_nonce != last_order_nonce + 1`
- reject if `expires_at <= issued_at`
- reject if `expires_at - issued_at > 300000`
- reject if current server time is greater than `expires_at`
- on success, atomically advance `last_order_nonce`

## Nonce, replay, and expiry model

### Registration

- one-time challenge
- challenge lifetime: `5 minutes`
- replaying a consumed challenge must fail

### Trading key lifetime

- default `key_expires_at = 0`
- `0` means durable until revoke or rotate
- durable key authorization is acceptable because:
  - the wallet only signs on first authorization and rare admin actions
  - replay protection is carried by per-order nonce
  - compromise recovery happens through revoke / rotate, not automatic expiry

### Order lifetime

- order nonce starts at `1`
- order nonce increments by exactly `1` per active key
- order lifetime is bounded by `issued_at` and `expires_at`
- maximum order lifetime: `5 minutes`
- if client nonce is out of sync, client must refetch `last_order_nonce` from
  the server before signing again

### Revoke behavior

- revoked or rotated keys cannot sign new orders
- revoke does not silently cancel already accepted orders
- open-order cancel remains a separate action and must be explicit

## Browser-local storage semantics

### Storage target

- trading private key should live in `IndexedDB`
- `localStorage` may hold only lightweight metadata:
  - `trading_key_id`
  - `wallet_address`
  - `chain_id`
  - `vault_address`
  - `trading_public_key`
  - `last_known_nonce`

### Restore

- on refresh, the app may restore metadata first
- before trading, the app must confirm that:
  - the connected wallet address still matches
  - the connected chain still matches
  - the server still reports the trading key as `ACTIVE`
- if metadata exists but the private key is missing, clear the metadata and
  require re-authorization

### Wallet switch

- active in-memory trading state must be dropped immediately when the connected
  wallet changes
- a trading key stored for wallet `A` must never be reused for wallet `B`
- storage should be namespaced by `wallet_address + chain_id + vault_address`

### Device migration and browser storage loss

- there is no deterministic recovery from a wallet signature
- if a user moves to a new browser or loses local storage, the user must:
  - reconnect the same wallet
  - generate a new local trading key
  - sign one new `AuthorizeTradingKey` message
- registering that new key rotates the old active key for the same
  `wallet_address + chain_id + vault_address`

## Revoke and rotate contract

Revoke uses the same EIP-712 domain as authorization.

Suggested revoke primary type:

```json
{
  "RevokeTradingKey": [
    { "name": "action", "type": "string" },
    { "name": "wallet", "type": "address" },
    { "name": "tradingKeyId", "type": "string" },
    { "name": "challenge", "type": "bytes32" },
    { "name": "challengeExpiresAt", "type": "uint64" }
  ]
}
```

Rules:

- explicit revoke requires a wallet signature, not a trading-key signature
- rotate is implemented as:
  - register a new trading key for the same
    `wallet_address + chain_id + vault_address`
  - atomically mark the previous active key `ROTATED`
- server must clear order acceptance for the old key immediately after revoke or
  rotate

## Deposit and withdrawal impact

The direct-vault model stays unchanged:

- deposits still go straight from the wallet to `FunnyVault`
- withdrawals / claims still stay on the direct-vault path

The auth change is only:

- how the wallet authorizes off-chain trading

Important V2 rule:

- deposit and withdrawal attribution must follow the durable wallet-to-user
  binding
- deposit credit must **not** depend on whether a short-lived local browser key
  currently exists

Until a dedicated wallet-binding table exists, the current durable binding can
be sourced from:

- `user_profiles.wallet_address`

An active trading key is required for trading, not for ownership of deposit
events already tied to the wallet.

## Migration from the current session-key model

### Phase 1: compatibility mapping

Keep the current persistence slot and verifier where possible:

- `wallet_sessions.session_id` becomes the compatibility carrier for
  `trading_key_id`
- `wallet_sessions.session_public_key` becomes `trading_public_key`
- `wallet_sessions.vault_address` becomes the durable vault scope for canonical
  trading-key rows
- `wallet_sessions.session_nonce` becomes the consumed wallet auth challenge
- `wallet_sessions.last_order_nonce` stays the order replay counter
- `wallet_sessions.expires_at = 0` means durable trading key

This lets the first worker reuse the current `ED25519` verifier and nonce logic
instead of rewriting the whole order path first.

Current landed truth:

- `wallet_sessions` now persists `vault_address` for canonical trading-key rows
- durable active-key rotation and listing now scope by
  `wallet_address + chain_id + vault_address`
- canonical trading-key rows now use durable uniqueness
  `wallet_address + chain_id + vault_address + session_public_key`, so one
  wallet can reuse the same trading public key across two vaults on the same
  chain without a uniqueness collision
- browser restore now reads back remote active keys by vault, so correctness no
  longer depends on a single-vault-per-environment assumption
- deprecated `/api/v1/sessions` compatibility rows still keep blank
  `vault_address`, so they remain in their own legacy blank-vault scope
  because the old session-grant message did not include a vault field

### Phase 2: API contract shift

- add challenge issuance and EIP-712 registration
- stop treating client-provided `user_id` as the public auth source of truth
- accept `trading_key_id` and `trading_signature` on orders
- temporarily allow old `session_*` field aliases during the rollout

Temporary compatibility rule for repo proof tooling:

- `POST /api/v1/sessions` remains available as a deprecated compatibility route
  for repo-local lifecycle and staging proof tooling
- that route preserves the legacy session-grant contract and should not be used
  as the V2 browser auth baseline
- rows created through that route continue to carry blank `vault_address`
- the canonical V2 browser flow remains
- `POST /api/v1/trading-keys/challenge` issues the one-time V2 auth challenge
- `POST /api/v1/trading-keys` registers the wallet-authorized trading key

### Phase 3: durable wallet binding for chain attribution

- move chain listener wallet lookup from "active wallet session" semantics to
  durable wallet binding semantics
- the first narrow implementation target can use `user_profiles.wallet_address`
  before a later dedicated wallet-auth table rename

### Phase 4: naming cleanup

- rename session-key terminology in frontend and backend code to trading-key
  terminology
- optionally rename `wallet_sessions` to a dedicated trading-key authorization
  table in a later schema migration

## Out of scope

- Stark curve signatures
- STARK proof batching
- state trees
- forced withdrawal / escape hatch
- on-chain trading-key registry transactions
- multi-device concurrent active trading keys
- encrypted key export / import or social recovery
- full smart-contract-wallet / `EIP-1271` support
