# TASK-CHAIN-001

## Summary

Harden the claim lane so malformed wallet or recipient addresses, or invalid payout payloads, cannot enter the queue through the API or turn into zero-address on-chain submissions inside the chain service.

## Scope

- reject malformed claim requests at the API boundary before a `CLAIM` task is created
- validate queued claim payloads again inside the chain claim processor before building/signing a transaction
- ensure invalid queued claim tasks move to a truthful failed state instead of attempting a zero-address submission
- add or update tests for both API bad-request handling and chain failure-path handling

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md](/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-008.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-008.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-008.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-001.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-001.md)

## Owned files

- `internal/api/dto/order.go`
- `internal/api/handler/order_handler.go`
- `internal/api/handler/order_handler_test.go`
- `internal/api/handler/sql_store.go`
- `internal/chain/service/claim_processor.go`
- `internal/chain/service/claim_processor_test.go`

## Acceptance criteria

- `POST /api/v1/payouts/:event_id/claim` rejects malformed wallet or recipient addresses with a truthful `400`
- chain claim submission validates wallet address, recipient address, and payout amount before signing
- invalid queued claim tasks are marked failed and do not send a transaction
- tests cover at least one API invalid-address case and one chain invalid-task failure case
- worker records verification evidence in the worklog

## Validation

- `cd /Users/zhangza/code/funnyoption && go test ./internal/api/... ./internal/chain/...`
- one API-level proof that an invalid claim request now returns `400`
- one chain-level proof that an invalid queued claim task is marked failed without producing a tx submission

## Dependencies

- `TASK-OFFCHAIN-008` output is the baseline

## Handoff

- return the hardened claim-lane behavior
- note whether any further chain hardening blockers remain after claim validation is fixed
