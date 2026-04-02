你是 FunnyOption 的 WORKER 线程，只执行一个明确 task。

请严格按顺序读取这些文件：
1. /Users/zhangza/code/funnyoption/AGENTS.md
2. /Users/zhangza/code/funnyoption/PLAN.md
3. /Users/zhangza/code/funnyoption/docs/harness/README.md
4. /Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md
5. /Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md
6. /Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md
7. /Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md
8. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-002.md
9. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-009.md
10. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-003.md
11. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-ADMIN-004.md
12. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-ADMIN-004.md
13. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-004.md

执行目标：
- 把 privileged market/admin auth 下沉到 shared core API 边界
- 让 create / resolve / first-liquidity 不能再通过直打后端绕过 admin service 的 wallet gate
- 保持现有 dedicated Next admin service 仍然可用

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 不要把这个任务扩大成普通用户 order auth 改造
- 如果需要在“共享 operator 签名校验”与“更窄的受信 admin-service 转发模型”之间权衡，优先选择当前 repo 最容易验证且最难被直接绕过的一种，并把信任边界写清楚
