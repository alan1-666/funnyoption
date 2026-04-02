# TASK-HARNESS-001

## Summary

Set up a harness-style operating model for FunnyOption so new agent threads can start from files instead of long chat history.

## Scope

- slim repo entry instructions
- add role files
- add plan / task / handshake / worklog structure
- add reusable templates
- add starter prompts for commander and worker threads

## Inputs to read

- [`/Users/zhangza/code/funnyoption/AGENTS.md`](/Users/zhangza/code/funnyoption/AGENTS.md)
- [`/Users/zhangza/code/funnyoption/PLAN.md`](/Users/zhangza/code/funnyoption/PLAN.md)
- [`/Users/zhangza/code/funnyoption/docs/harness/README.md`](/Users/zhangza/code/funnyoption/docs/harness/README.md)
- [`/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md`](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [`/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md`](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)

## Owned files

- `AGENTS.md`
- `PLAN.md`
- `docs/harness/**`

## Acceptance criteria

- repo has a slim entry map
- commander and worker roles are clearly separated
- plan, task, handshake, and worklog locations are explicit
- a new thread can be started with a prompt that points to files in order

## Validation

- manually verify links and read order
- ensure the file tree is concise and navigable

## Dependencies

- none

## Handoff

- commander should use this framework for all future multi-thread work
