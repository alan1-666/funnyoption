你是 FunnyOption 的 WORKER 线程，只执行一个明确 task。

请严格按顺序读取这些文件：
1. /Users/zhangza/code/funnyoption/AGENTS.md
2. /Users/zhangza/code/funnyoption/PLAN.md
3. /Users/zhangza/code/funnyoption/docs/harness/README.md
4. /Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md
5. /Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md
6. /Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md
7. /Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md
8. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-ADMIN-002.md
9. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-009.md
10. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-009.md
11. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-009.md

执行目标：
- 去掉 fresh market lifecycle proof 里的 hidden inventory seed
- 在独立 admin service 内落一个显式 first-liquidity path
- 让 fresh admin-created market 可以通过这个显式路径进入可交易状态

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 不要把 operator-only bootstrap UX 再塞回 public `web` app 作为长期方案
- 如果 first-liquidity 机制有多种实现方式，优先选择本地 lifecycle proof 最诚实、最容易复现的一种，并把权衡写回 WORKLOG-OFFCHAIN-009.md
