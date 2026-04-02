# Worker Thread Prompt: TASK-OFFCHAIN-004

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
7. /Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-004.md
8. /Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-004.md
9. /Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-004.md

执行目标：
- 修复 resolved-market finality
- 确保已 RESOLVED 的 market 不能再接单、不能继续保留 active resting orders、冷启动后不能再恢复出可成交盘口
- 把精确的回归步骤、HTTP 状态、trade / order / freeze 观察结果写回 WORKLOG-OFFCHAIN-004.md

执行规则：
- 只在 task 和 handshake 允许的 scope 内工作
- 优先保证终态正确性，不要顺手扩大到 read-surface cleanup
- 如果发现历史 stale freeze 需要单独修复，先在 worklog 里记录证据和建议，不要在本任务里顺手做大范围回填
```
