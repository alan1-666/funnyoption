# Worker Thread Prompt: TASK-OFFCHAIN-007

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
7. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-007.md
8. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-007.md
9. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-007.md

执行目标：
- 修正本地 stale freeze cleanup SQL，让释放后的 freeze 行把 `remaining_amount` 归零
- 更新 runbook，使说明和 SQL 行为一致
- 把 rollback 验证结果写回 WORKLOG-OFFCHAIN-007.md

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 不要修改 `internal/**` 运行时代码
- 这是本地 DB 工具链修正，不要扩大成 productized reconciliation
```
