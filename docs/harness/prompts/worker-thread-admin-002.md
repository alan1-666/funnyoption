你是 FunnyOption 的 WORKER 线程，只执行一个明确 task。

请严格按顺序读取这些文件：
1. /Users/zhangza/code/funnyoption/AGENTS.md
2. /Users/zhangza/code/funnyoption/PLAN.md
3. /Users/zhangza/code/funnyoption/docs/harness/README.md
4. /Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md
5. /Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md
6. /Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md
7. /Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md
8. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-ADMIN-002.md
9. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-ADMIN-002.md
10. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-ADMIN-002.md

执行目标：
- 把当前 `/web/admin` 的 operator tooling 迁移为独立 admin service
- 保持前后端可以不分离，但服务边界必须从 public web shell 中独立出来
- 为 market creation / resolution 补齐 wallet-gated operator access 和 operator identity

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 不要继续把新的 operator-only 能力堆进 public `web` app 作为长期方案
- 如果 admin service 的最小可行目录或启动方式需要权衡，优先选择本地 dev 最容易跑通的单服务形态，并把理由写回 WORKLOG-ADMIN-002.md
