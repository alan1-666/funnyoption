# TASK-API-001

## Summary

Refactor the API service toward Gin best practices by splitting route registration by module, adding explicit middleware-based auth layering, and applying rate limiting to sensitive write paths.

## Scope

- replace the current one-file mixed route registration with module-oriented route registration inside `internal/api`
- define explicit route groups for at least:
  - public health/meta
  - public reads
  - session/wallet auth
  - trade writes
  - operator/admin writes
- add a middleware structure that is clearer than the current ad hoc `Recovery + Logger + CORS` stack
- add rate limiting to sensitive endpoints such as session creation, order entry, claim submission, and privileged admin/operator writes
- align auth boundaries with current product semantics:
  - public reads remain readable
  - user trade writes require the existing session / signature model
  - operator/admin writes respect the hardened operator model from `TASK-ADMIN-004`
- keep the task focused on API service structure and boundary enforcement; do not widen into unrelated product redesign

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md](/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-004.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-API-001.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-API-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-001.md)

## Owned files

- `internal/api/server.go`
- `internal/api/handler/**`
- `internal/api/**` newly introduced middleware/router files if needed
- `internal/shared/auth/**` if narrowly required
- related API docs and runbooks

## Acceptance criteria

- route registration is split by module or concern instead of one mixed `RegisterRoutes` block
- middleware stack is explicit and documented, with at least recovery, logging, CORS, and rate limiting arranged intentionally
- sensitive API write paths have rate limiting
- auth layering is clearer in the router structure and enforced by middleware or equivalent route-group boundary logic
- focused tests or validation prove:
  - public reads still work
  - session-backed trade writes still work
  - protected operator/admin writes still require the intended auth
  - rate limiting denies or throttles repeated abusive calls on at least one sensitive path

## Validation

- `go test ./internal/api/...`
- one local proof for a normal public read
- one local proof for an authorized write path
- one local proof for a rate-limited sensitive endpoint

## Dependencies

- `TASK-ADMIN-004` output is the baseline

## Handoff

- return the chosen router/module structure
- state which endpoints are rate limited and how
- state the final auth boundary model in the API service after the refactor
