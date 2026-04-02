# Worker Thread Prompt: TASK-OFFCHAIN-002

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
7. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-002.md
8. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-002.md
9. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-002.md

执行目标：
- 完成本地 off-chain 回归闭环
- 验证 homepage、detail page、matching、settlement、candle push
- 把可复现的本地验证步骤和 pass/fail matrix 写回 WORKLOG-OFFCHAIN-002.md

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 先验证再假设，不要默认当前实现已经可用
- 如果遇到 blocker，先把 blocker、影响范围、建议下一步写回 worklog，再回传 commander
```
