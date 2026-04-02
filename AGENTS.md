# FunnyOption Agent Map

This file is the entrypoint, not the encyclopedia.
Read it first, then follow the linked files in order.

## Operating model

- The repository is the source of truth.
- Plans, tasks, handshakes, and worklogs live in versioned files under `docs/harness/`.
- Keep context small: read the minimum set of files needed for the current task.
- Prefer narrow tasks with explicit ownership and acceptance criteria.
- Record decisions in files instead of relying on chat history.

## Mandatory startup order

1. Read [`PLAN.md`](/Users/zhangza/code/funnyoption/PLAN.md)
2. Read [`docs/harness/README.md`](/Users/zhangza/code/funnyoption/docs/harness/README.md)
3. Read the role file for this thread:
   - Commander: [`docs/harness/roles/COMMANDER.md`](/Users/zhangza/code/funnyoption/docs/harness/roles/COMMANDER.md)
   - Worker: [`docs/harness/roles/WORKER.md`](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
4. Read [`docs/harness/PROJECT_MAP.md`](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
5. Read [`docs/harness/THREAD_PROTOCOL.md`](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
6. Read the active task and handshake files assigned to this thread
7. Only then open the relevant code and domain docs

## Where to look next

- Harness index: [`docs/harness/README.md`](/Users/zhangza/code/funnyoption/docs/harness/README.md)
- Master plan: [`PLAN.md`](/Users/zhangza/code/funnyoption/PLAN.md)
- Project map: [`docs/harness/PROJECT_MAP.md`](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- Thread rules: [`docs/harness/THREAD_PROTOCOL.md`](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)

## Domain map

- Order ingress and API: [`internal/api`](/Users/zhangza/code/funnyoption/internal/api)
- Matching core: [`internal/matching`](/Users/zhangza/code/funnyoption/internal/matching)
- Mutable balances and freezes: [`internal/account`](/Users/zhangza/code/funnyoption/internal/account)
- Settlement and payouts: [`internal/settlement`](/Users/zhangza/code/funnyoption/internal/settlement)
- Ledger and reconciliation: [`internal/ledger`](/Users/zhangza/code/funnyoption/internal/ledger)
- Chain listener and claim flow: [`internal/chain`](/Users/zhangza/code/funnyoption/internal/chain)
- WebSocket fanout: [`internal/ws`](/Users/zhangza/code/funnyoption/internal/ws)
- Frontend: [`web`](/Users/zhangza/code/funnyoption/web)
- Contracts: [`contracts`](/Users/zhangza/code/funnyoption/contracts)

## Core docs

- Architecture and order flow: [`docs/architecture/order-flow.md`](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- Ledger boundary: [`docs/architecture/ledger-service.md`](/Users/zhangza/code/funnyoption/docs/architecture/ledger-service.md)
- Direct deposit and session key flow: [`docs/architecture/direct-deposit-session-key.md`](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- SQL schema: [`docs/sql/schema.md`](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- Kafka topics: [`docs/topics/kafka-topics.md`](/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md)

## Planning artifacts

- Active plans: [`docs/harness/plans/active/`](/Users/zhangza/code/funnyoption/docs/harness/plans/active)
- Completed plans: [`docs/harness/plans/completed/`](/Users/zhangza/code/funnyoption/docs/harness/plans/completed)
- Task files: [`docs/harness/tasks/`](/Users/zhangza/code/funnyoption/docs/harness/tasks)
- Thread handshakes: [`docs/harness/handshakes/`](/Users/zhangza/code/funnyoption/docs/harness/handshakes)
- Worklogs: [`docs/harness/worklogs/`](/Users/zhangza/code/funnyoption/docs/harness/worklogs)

## Repo rules

- Do not turn this file into a long handbook.
- Add new knowledge to the right domain doc or harness file, then link it here only if it is a stable entrypoint.
- Prefer append-only worklogs and explicit task status updates over ad hoc chat summaries.
- If a task changes architecture or scope, update the plan and handshake before coding.
- If a task is blocked, record the blocker in the handshake and worklog with next actions.
