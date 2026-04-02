# Worker Thread Prompt: TASK-OFFCHAIN-008

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
7. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-008.md
8. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-008.md
9. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-008.md

执行目标：
- 修正本地 API collection contract，让空列表接口返回 `{"items":[]}` 而不是 `{"items":null}`
- 至少覆盖 `trades` 和 `chain-transactions` 这两个已经影响 homepage/detail/control 的端点
- 把 empty-collection 的验证结果写回 WORKLOG-OFFCHAIN-008.md

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 不要扩展到 chain 执行逻辑或新的前端改版
- 优先做一致性的共享修复，避免只补单个 endpoint
```
