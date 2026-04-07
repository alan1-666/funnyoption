# TASK-CHAIN-034

## Summary

Land the first frozen-mode trading guard:
once `rollup_freeze_state.frozen = true`, stop new trading writes at API ingress
and matching runtime, and prevent matching restart from restoring stale resting
orders as live book state.

## Scope

- build directly on `TASK-CHAIN-033`
- implement:
  - one API-side rollup-freeze gate for order creation
  - one matching-side rollup-freeze gate for tradability checks
  - one matching restart guard so frozen mode restores no resting orders
  - docs that explicitly call this a frozen-mode runtime truth guard, not full
    escape-hatch or full production-truth switching
- do not implement:
  - Merkle-proof escape claims
  - a global mutable-backend truth switch
  - a new prover / verifier contract lane

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-033.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-033.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-033.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-033.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-033.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-033.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/backend/internal/api/handler/order_handler.go](/Users/zhangza/code/funnyoption/backend/internal/api/handler/order_handler.go)
- [/Users/zhangza/code/funnyoption/backend/internal/matching/service/sql_store.go](/Users/zhangza/code/funnyoption/backend/internal/matching/service/sql_store.go)
- [/Users/zhangza/code/funnyoption/backend/internal/matching/service/consumer.go](/Users/zhangza/code/funnyoption/backend/internal/matching/service/consumer.go)

## Owned files

- `internal/api/handler/**`
- `internal/matching/service/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-034.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-034.md`

## Acceptance criteria

- `/api/v1/orders` rejects new orders once the mirrored rollup freeze state is
  frozen
- matching runtime no longer treats frozen mode as tradable even if a market
  row is still otherwise open
- matching restart restores no resting orders when frozen
- docs stay explicit that this is still not full escape-hatch or full
  production-truth switching

## Validation

- `go test ./internal/chain/service ./internal/api ./internal/api/handler ./internal/matching/service ./internal/rollup`
- `forge test --offline --match-path contracts/test/FunnyRollupCore.t.sol`
- `git diff --check`

## Dependencies

- `TASK-CHAIN-033` completed

## Handoff

- return changed files, frozen-mode runtime contract, validation commands,
  residual limitations, and the recommended next follow-up for broader frozen
  production-truth switching or escape-hatch runtime
