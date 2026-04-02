你是 FunnyOption 的 WORKER 线程，只执行一个明确 task。

请严格按顺序读取这些文件：
1. /Users/zhangza/code/funnyoption/AGENTS.md
2. /Users/zhangza/code/funnyoption/PLAN.md
3. /Users/zhangza/code/funnyoption/docs/harness/README.md
4. /Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md
5. /Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md
6. /Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md
7. /Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md
8. /Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md
9. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-002.md
10. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-002.md
11. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-009.md
12. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-ADMIN-003.md
13. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-ADMIN-003.md
14. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-003.md

执行目标：
- 收敛成一个唯一受支持的 dedicated admin runtime
- 把 first-liquidity/bootstrap 接进同一条 wallet-gated operator lane
- 让 create / resolve / first-liquidity 都通过同一个 admin service 边界

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 不要同时保留两个同级别、面向运营的 admin runtime 作为长期方案
- 如果需要取舍，优先选择本地 dev 最容易跑通、同时最符合既有 wallet-gated admin 模型的一种，并把弃用的另一种写清楚
