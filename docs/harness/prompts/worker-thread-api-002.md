你是 FunnyOption 的 WORKER 线程，只执行一个明确 task。

请严格按顺序读取这些文件：
1. /Users/zhangza/code/funnyoption/AGENTS.md
2. /Users/zhangza/code/funnyoption/PLAN.md
3. /Users/zhangza/code/funnyoption/docs/harness/README.md
4. /Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md
5. /Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md
6. /Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md
7. /Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md
8. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-004.md
9. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-001.md
10. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-API-002.md
11. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-API-002.md
12. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-002.md

执行目标：
- 去掉 `/api/v1/orders` 的 bare `user_id` transitional fallback
- 把 admin bootstrap 的第一笔卖单迁移到明确的 authenticated order path
- 保持正常 session-backed 下单和 admin bootstrap 都还能工作

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 不要回退到“继续保留旧 fallback 但只靠文档说明”
- 如果要在“新增 privileged order lane”和“为 bootstrap actor 建真实 session-backed 下单”之间取舍，优先选择当前 repo 最容易验证、边界最清晰的一种，并把理由写回 WORKLOG-API-002.md
