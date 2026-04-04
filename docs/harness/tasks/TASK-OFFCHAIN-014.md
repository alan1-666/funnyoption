# TASK-OFFCHAIN-014

## Summary

Define the Stark-style wallet-linked trading-key architecture so a user signs
once with MetaMask, gets one browser-local off-chain trading key, and then
submits subsequent orders without repeated wallet prompts.

## Scope

- stay design-first for this task; do not jump into a wide runtime rewrite
  before the auth contract is explicit
- design around the user-stated target flow:
  - first connection:
    - user connects MetaMask
    - MetaMask signs one off-chain message, not a gas-paying transaction
    - browser derives or registers one Stark-style trading keypair
    - trading private key stays browser-local
    - trading public key is registered to the operator and bound to the user's
      EVM wallet address
  - later trading:
    - orders are signed by the browser-local trading key
    - MetaMask should not pop up on every order
  - deposits:
    - remain direct on-chain vault deposits
    - operator / chain service listens for deposit events and credits off-chain
      balances
- explicitly decide one key question instead of leaving it ambiguous:
  - preferred target from product is deterministic trading-key derivation from
    the wallet signature
  - if that is too fragile across wallets / signing modes / reproducibility,
    the worker must record the exact blocker and compare it against the safer
    fallback of locally generating a Stark key and wallet-authorizing its public
    key
- define the auth contract end to end:
  - exact wallet message format and signing standard
  - chain/domain binding
  - trading-key registration payload
  - order-signature payload shape
  - nonce / replay / expiry model
  - local browser storage and recovery behavior
  - revoke / rotate / wallet-switch / device-migration semantics
  - migration path from the current ed25519-style session-key model
- keep current centralized matching and direct-vault deposit model; do not widen
  into StarkEx prover / state tree / forced-withdrawal infrastructure

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/web/lib/session-client.ts](/Users/zhangza/code/funnyoption/web/lib/session-client.ts)
- [/Users/zhangza/code/funnyoption/web/components/trading-session-provider.tsx](/Users/zhangza/code/funnyoption/web/components/trading-session-provider.tsx)
- [/Users/zhangza/code/funnyoption/internal/shared/auth/session.go](/Users/zhangza/code/funnyoption/internal/shared/auth/session.go)
- [/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go](/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go)
- [/Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go](/Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-014.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-014.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-014.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-014.md)

## Owned files

- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-014.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-014.md`
- optional narrow DTO / schema placeholders only if they directly support the
  design handoff

## Acceptance criteria

- one explicit auth design document or design-section update covers:
  - first-login flow
  - key derivation or key-authorization contract
  - operator registration shape
  - order-signature payload
  - nonce / replay / expiry / revoke rules
  - browser storage / recovery / wallet-switch behavior
  - migration from the current session-key model
- the design states clearly whether signature-derived deterministic Stark keys
  are accepted, and if not, why not
- the design preserves the current direct-vault deposit flow
- the design is explicit enough that a later implementation worker can proceed
  without reopening core auth architecture

## Validation

- design is internally consistent with the existing deposit/order/settlement
  docs
- any optional schema placeholders align with `docs/sql/schema.md`
- worklog records the chosen contract and the key rejected alternatives

## Dependencies

- current staging and local flows are the baseline

## Handoff

- return changed files, the final auth contract, and the recommended first
  implementation slice
- state explicitly what remains out of scope for the follow-up runtime worker
