# TASK-ADMIN-002

## Summary

Extract the current `/admin` operator tooling into a dedicated admin service, while keeping frontend and backend allowed to remain coupled inside that service, and require wallet-gated operator access with explicit operator identity for market actions.

## Scope

- define one dedicated admin-service runtime and move operator tooling toward that boundary instead of growing the public `web` shell
- keep frontend and backend allowed to remain coupled inside the admin service; do not force a frontend/backend split as part of this task
- migrate or mirror the current operator flows out of the public app shell
- require operator wallet/session authorization before market creation or resolution actions can run
- keep the public app and `/control` free of operator-only controls
- surface operator identity or denial state clearly in the admin UI

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-ADMIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-ADMIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-002.md)

## Owned files

- `admin/**` as the preferred new dedicated service root
- `web/app/admin/**` as migration source or temporary redirect shell
- `web/components/admin-market-ops.tsx`
- `web/components/market-studio.tsx`
- `web/components/trading-session-provider.tsx`
- `web/lib/session-client.ts`
- `scripts/dev-up.sh`
- any narrowly required API/session config files

## Acceptance criteria

- a dedicated admin service exists as the target operator entrypoint instead of relying on the public `web` app shell
- frontend and backend may remain coupled inside that admin service
- the public `web` app no longer acts as the long-term privileged operator surface
- operator wallet/session state is explicit in the UI
- unauthorized users cannot create or resolve markets from the admin surface
- docs/worklog state the current admin runtime model, local startup shape, and any remaining limitations

## Validation

- build or start proof for the new admin service runtime
- one UI proof for authorized admin state
- one UI or API proof for unauthorized denial state

## Dependencies

- `TASK-CHAIN-002` output is the baseline

## Handoff

- return the chosen admin-service shape, the admin access model, and any remaining backend auth gaps
