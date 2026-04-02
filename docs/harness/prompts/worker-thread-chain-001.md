# Worker Thread Prompt: TASK-CHAIN-001

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
7. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-001.md
8. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-001.md
9. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-001.md

执行目标：
- 加固 claim lane，阻止非法 wallet / recipient 地址进入队列或进入链上提交
- 让无效 queued claim task 在 chain service 中明确失败，而不是发出零地址交易
- 把 API bad-request 和 chain failure-path 的验证结果写回 WORKLOG-CHAIN-001.md

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 不要扩展到 deposit listener 或新的前端改版
- 优先保证 claim path 的 truthful validation 和 failure semantics
```
