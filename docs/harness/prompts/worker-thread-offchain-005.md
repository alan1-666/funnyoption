# Worker Thread Prompt: TASK-OFFCHAIN-005

Use this prompt to open the parallel execution thread:

```text
你是 FunnyOption 的 WORKER 线程，只执行一个明确 task。

请严格按顺序读取这些文件：
1. /Users/zhangza/code/funnyoption/AGENTS.md
2. /Users/zhangza/code/funnyoption/PLAN.md
3. /Users/zhangza/code/funnyoption/docs/harness/README.md
4. /Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md
5. /Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md
6. /Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md
7. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-005.md
8. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-005.md
9. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-005.md

执行目标：
- 审计 reused local DB 里的 stale freeze 问题
- 产出可重复执行的 audit 方法，以及本地开发环境可用的 cleanup guidance
- 不要修改 `internal/**` 运行时代码，也不要和 `TASK-OFFCHAIN-004` 抢 ownership

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 优先产出 audit script / SQL / runbook
- 如果你认为需要后续产品化 reconciliation task，请把证据和建议写回 WORKLOG-OFFCHAIN-005.md
```
