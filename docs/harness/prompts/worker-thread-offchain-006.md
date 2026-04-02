# Worker Thread Prompt: TASK-OFFCHAIN-006

Use this prompt to open the next execution thread:

```text
你是 FunnyOption 的 WORKER 线程，只执行一个明确 task。

请严格按顺序读取这些文件：
1. /Users/zhangza/code/funnyoption/AGENTS.md
2. /Users/zhangza/code/funnyoption/PLAN.md
3. /Users/zhangza/code/funnyoption/docs/harness/README.md
4. /Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md
5. /Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md
6. /Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md
7. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-006.md
8. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-006.md
9. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-006.md

执行目标：
- 修复 homepage / detail / control 的 SSR 读取层，让 API 故障不再伪装成空数据
- 明确区分 empty state、not found、API unavailable
- 把 degraded-path 验证结果写回 WORKLOG-OFFCHAIN-006.md

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 不要扩展到 chain hardening 或本地 DB 清理
- 优先保证“诚实呈现错误”，不要继续返回会误导 operator 的空数组/空对象
```
