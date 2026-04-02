# Worker Thread Prompt: TASK-OFFCHAIN-003

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
7. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-003.md
8. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-003.md
9. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-003.md

执行目标：
- 清理 homepage、market detail、control 的 read/query surfaces
- 减少误导性的 fallback/mock 路径，并让 operator visibility 更贴近真实本地状态
- 输出 homepage / detail / control 的 pass/fail matrix，并把剩余 gap 写回 WORKLOG-OFFCHAIN-003.md

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 先读 `TASK-OFFCHAIN-002` 和 `TASK-OFFCHAIN-004` 的回归结果，再决定哪些 fallback 可以安全移除
- 不要顺手扩展到 chain hardening；如果发现需要后续链路任务，只记录证据和建议
```
