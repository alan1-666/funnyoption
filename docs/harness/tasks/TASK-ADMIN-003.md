# TASK-ADMIN-003

## Summary

Converge the admin/operator surface to one supported dedicated runtime and extend wallet-gated operator access to the explicit first-liquidity/bootstrap flow.

## Scope

- choose one supported dedicated admin runtime in `admin/` and make that the single operator entrypoint for local dev and docs
- remove, demote, or clearly deprecate the duplicate admin runtime shape so the repo no longer presents two competing operator surfaces
- move the explicit first-liquidity/bootstrap flow behind the same wallet-gated operator boundary already used for create/resolve
- keep explicit operator identity visible for privileged actions
- keep the public `web` shell free of operator-only tooling beyond migration pointers
- do not widen into shared core-API auth for every privileged backend endpoint unless it is strictly required to complete the admin-service convergence

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md](/Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-009.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-009.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-ADMIN-003.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-ADMIN-003.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-003.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-003.md)

## Owned files

- `admin/**`
- `web/app/admin/**`
- `scripts/dev-up.sh`
- `docs/operations/local-offchain-lifecycle.md`
- related admin runtime docs

## Acceptance criteria

- the repo documents one supported dedicated admin runtime, not two competing runtime shapes
- create, resolve, and first-liquidity/bootstrap all run through that same admin runtime boundary
- first-liquidity/bootstrap is wallet-gated with the same operator signature / allowlist model as the other admin actions, or the worker documents a narrower equivalent model and why it is sufficient
- unauthorized bootstrap requests are denied
- local startup docs and `dev-up.sh` point to the same supported admin runtime

## Validation

- build or start proof for the chosen admin runtime
- one authorized proof that create or first-liquidity succeeds through the supported runtime
- one unauthorized proof that first-liquidity/bootstrap is denied

## Dependencies

- `TASK-OFFCHAIN-009` output is the baseline

## Handoff

- return the chosen admin runtime shape
- state which runtime was deprecated or removed
- note any remaining deeper backend-auth gaps after runtime convergence
