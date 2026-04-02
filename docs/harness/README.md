# Harness Index

This directory turns the repo into a navigable map for short-context agent threads.

## Read order

1. [`/Users/zhangza/code/funnyoption/AGENTS.md`](/Users/zhangza/code/funnyoption/AGENTS.md)
2. [`/Users/zhangza/code/funnyoption/PLAN.md`](/Users/zhangza/code/funnyoption/PLAN.md)
3. Role file:
   - [`roles/COMMANDER.md`](/Users/zhangza/code/funnyoption/docs/harness/roles/COMMANDER.md)
   - [`roles/WORKER.md`](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
4. [`PROJECT_MAP.md`](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
5. [`THREAD_PROTOCOL.md`](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
6. Task, handshake, and worklog files for the current thread

## Directory roles

- `roles/`: operating instructions by thread type
- `plans/active/`: current orchestrated plans
- `plans/completed/`: archived plans
- `tasks/`: scoped implementation or research tasks
- `handshakes/`: per-thread contracts and dependency notes
- `worklogs/`: append-only working journals
- `prompts/`: starter prompts for new threads
- `templates/`: reusable file templates

## Design goals

- keep the top-level prompt small
- make plans first-class
- let new threads start from files instead of chat history
- preserve decisions and blockers in the repo
