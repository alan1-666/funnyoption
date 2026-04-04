# HANDSHAKE-OFFCHAIN-014

## Task

- [TASK-OFFCHAIN-014.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-014.md)

## Thread owner

- off-chain auth design worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `web/lib/session-client.ts`
- `web/components/trading-session-provider.tsx`
- `internal/shared/auth/session.go`
- `internal/api/handler/order_handler.go`
- `internal/api/handler/sql_store.go`
- `docs/sql/schema.md`
- this handshake
- `WORKLOG-OFFCHAIN-014.md`

## Files in scope

- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-014.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-014.md`
- optional narrow DTO / schema placeholders only if they directly support the
  design handoff

## Inputs from other threads

- the current repo baseline is not StarkEx-style:
  - it uses a wallet-signed authorization for a browser-generated session key
  - it explicitly says not to derive a trading private key from the wallet
    signature
- product direction has now changed:
  - one MetaMask signature at first login
  - one browser-local off-chain trading key for later order signing
  - direct on-chain deposits still feed the off-chain account system
- commander wants this lane to stay design-first so implementation does not
  silently widen across frontend auth, backend auth, schema, and order ingress
  without one explicit contract

## Outputs back to commander

- changed files
- final auth contract
- recommendation on deterministic signature-derived trading key versus
  wallet-authorized locally generated trading key
- recommended first implementation slice

## Blockers

- do not widen into full StarkEx prover / DA / escape-hatch scope
- do not silently change the trust model without documenting the migration from
  the current session-key baseline
- preserve the current direct-vault deposit model

## Status

- queued
