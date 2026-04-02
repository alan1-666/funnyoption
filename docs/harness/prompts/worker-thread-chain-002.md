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
9. /Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md
10. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-002.md
11. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-002.md
12. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-002.md

执行目标：
- 把当前 lifecycle proof 里的 simulated deposit 替换成真实 listener-driven deposit credit proof
- 让 worker 明确写出 proof environment、复现命令、tx/deposit/balance 证据

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 不要悄悄回退到直接调用 ApplyConfirmedDeposit(...) 作为主 proof
- 如果真实 listener proof 做不到，先把具体 blocker、缺失 env、受影响链路写回 WORKLOG-CHAIN-002.md，再回传 commander
