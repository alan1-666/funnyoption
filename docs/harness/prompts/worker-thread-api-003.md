你是 FunnyOption 的 WORKER 线程，只执行一个明确 task。

请严格按顺序读取这些文件：
1. /Users/zhangza/code/funnyoption/AGENTS.md
2. /Users/zhangza/code/funnyoption/PLAN.md
3. /Users/zhangza/code/funnyoption/docs/harness/README.md
4. /Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md
5. /Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md
6. /Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md
7. /Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md
8. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-002.md
9. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-API-003.md
10. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-API-003.md
11. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-003.md

执行目标：
- 给 privileged bootstrap order lane 补 replay / idempotency 保护
- 保持 admin bootstrap 第一次合法下单仍然成功
- 保持普通 session-backed 下单不受影响

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 不要重新引入 bare `user_id` fallback
- 如果需要在“显式 bootstrap nonce”与“更窄的 idempotency key”之间取舍，优先选择当前 repo 最容易验证、边界最清晰的一种，并把理由写回 WORKLOG-API-003.md
