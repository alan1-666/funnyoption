你是 FunnyOption 的 WORKER 线程，只执行一个明确 task。

请严格按顺序读取这些文件：
1. /Users/zhangza/code/funnyoption/AGENTS.md
2. /Users/zhangza/code/funnyoption/PLAN.md
3. /Users/zhangza/code/funnyoption/docs/harness/README.md
4. /Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md
5. /Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md
6. /Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md
7. /Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md
8. /Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md
9. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-004.md
10. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-API-001.md
11. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-API-001.md
12. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-001.md

执行目标：
- 按 Gin 最佳实践整理 API service
- 把路由按模块拆开，不再把所有东西混在一个 `RegisterRoutes`
- 补齐中间件层的限流和清晰鉴权边界

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 不要回退或绕开 `TASK-ADMIN-004` 选定的 backend auth 模型
- 如果限流策略需要取舍，优先保证 session create、order write、claim、operator/admin write 这些敏感路径先有可验证的保护
