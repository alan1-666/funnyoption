# Thread Protocol

## File set per active task

Every active worker thread should have:

- one task file in `docs/harness/tasks/`
- one handshake file in `docs/harness/handshakes/`
- one worklog file in `docs/harness/worklogs/`

## Startup sequence for a worker thread

1. Read `AGENTS.md`
2. Read `PLAN.md`
3. Read `roles/WORKER.md`
4. Read `PROJECT_MAP.md`
5. Read this file
6. Read the assigned task file
7. Read the assigned handshake file
8. Read the latest relevant worklog
9. Read only the domain docs and code linked by the task

## Startup sequence for a commander thread

1. Read `AGENTS.md`
2. Read `PLAN.md`
3. Read `roles/COMMANDER.md`
4. Read `PROJECT_MAP.md`
5. Read this file
6. Read the active plan and open tasks
7. Update plan, task routing, and handshakes

## Handshake contract

Handshake files should contain:

- task id
- owner thread / role
- scope
- exact files and modules to touch
- dependencies on other tasks
- input docs to read
- expected output
- blockers and open questions
- handoff notes back to commander

## Worklog contract

Worklogs are append-only.
Each entry should record:

- timestamp
- thread id or nickname
- what was read
- what was changed
- what was validated
- blockers / next actions

## Commander -> worker handoff

Commander must provide:

- one task file path
- one handshake file path
- explicit success criteria
- file ownership boundaries
- dependency ordering if another worker is involved

## Worker -> commander return

Worker should return with:

- task status
- changed files
- tests / validation performed
- remaining risk
- required follow-up tasks
