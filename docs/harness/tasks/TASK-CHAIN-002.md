# TASK-CHAIN-002

## Summary

Replace the current simulated deposit credit in the lifecycle proof with a truthful wallet deposit path that is observed by the running chain listener and credited through the normal chain-service flow.

## Scope

- make the local or testnet-ready deposit path explicit and reproducible for this repo
- ensure `chain-service` can bootstrap a listener-ready configuration for the chosen proof environment
- update the lifecycle proof so the deposit step is driven by an observed vault deposit event instead of a direct `ApplyConfirmedDeposit(...)` call
- record exact env requirements, tx proof, deposit row proof, and balance delta proof in repo docs/worklog
- keep the worker narrowly focused on deposit listener truthfulness; do not widen into withdrawals or claims

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md](/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md)
- [/Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md](/Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-001.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-002.md)

## Owned files

- `internal/chain/service/**`
- `cmd/chain/**`
- `cmd/local-lifecycle/**`
- `scripts/dev-up.sh`
- `docs/operations/local-offchain-lifecycle.md`
- any narrowly required chain env/bootstrap docs under `docs/`

## Acceptance criteria

- the primary lifecycle proof no longer uses a direct confirmed-deposit processor call as the deposit step
- worker demonstrates one real listener-observed deposit credit in the chosen local/testnet proof path
- worklog records:
  - the deposit transaction hash or equivalent on-chain proof reference
  - the resulting `chain_deposits` row or API read proof
  - the resulting balance delta proof
- docs clearly state any required env vars, RPC requirements, and limitations

## Validation

- `cd /Users/zhangza/code/funnyoption && go test ./internal/chain/...`
- one proof run that shows:
  - wallet deposit submitted
  - listener observed it
  - backend credited it
  - user balance increased through the normal read surface

## Dependencies

- `TASK-CHAIN-001` output is the baseline

## Handoff

- return the truthful deposit proof path
- state clearly whether the proof is fully local, shared-testnet, or mixed
- note any remaining chain hardening blockers after deposit truthfulness is restored
