# Matching Engine Benchmark — 2026-04-12

## 测试环境

| 项 | 值 |
|---|---|
| 宿主 | `76.13.220.236` (staging 测试服) |
| VM 规格 | 2 vCPU / 8 GiB RAM / 无 swap |
| 物理 CPU | AMD EPYC 9355P 32-Core @ 2.0GHz (宿主) |
| OS / 内核 | Linux amd64 |
| Go | `go1.26.1 linux/amd64` |
| 运行参数 | `GOMAXPROCS=2 GOMEMLIMIT=5GiB GOGC=50`, `-benchmem`, `count=3` |
| 源码 commit | `83c4eca` (本地 `git archive HEAD`，非服务器已部署版本) |
| 源码位置 | `/tmp/funnyoption-bench/` (服务器) |
| 脚本 | `/tmp/run-matching-bench.sh` (服务器) |
| 原始输出 | `/tmp/matching-bench.out` (服务器) |

## 部署状态 (压测当下)

- `/opt/funnyoption-staging` HEAD = `e86fca4` (`feat(vault): streamline deposit console`, 2026-04-09 12:17)
- 最新 commit `83c4eca` / `3823f1e` (2026-04-12 凌晨) **未同步到服务器**
- `funnyoption-staging-matching-1` 容器 `Created 2026-04-09T08:25`，3 天未重建
- 服务器上未发现 cron / systemd timer 做自动 pull
- `/opt/funnyoption-staging` 工作区含大量 `custody/` 相关的 uncommitted / untracked 改动，直接 `git pull` 会撞冲突
- **结论**：最近的 push 没有触发任何自动部署链路

## 结果汇总 (`83c4eca`, 3 次均值)

| 场景 | ns/op | ops/s (单线程) | allocs | B/op |
|---|--:|--:|--:|--:|
| `DeterministicTradeID` | 31.1 | 32.1 M | 1 | 16 |
| `AddOrder_Fresh` | 270.2 | **3.70 M** | 4 | 256 |
| `PlaceOrder_DeepBook` | 658.5 | **1.52 M** | 9 | 490 |
| `Match_CrossSpread` | 787.1 | **1.27 M** | 9 | 490 |
| `Match_CrossSpread_WithEpoch` | 859.0 | **1.16 M** | 9 | 490 |
| `Match_IOC_SweepBook` | 855.4 | 1.17 M | 9 | 490 |
| `Match_STPSkip` | 5 854.7 | 171 K | 5 | 639 |
| `PlaceOrder_MultiBook100` | 5 922.3 | **169 K** | 6 | 271 |
| `CancelOrders` | 7 813.0 | 128 K | 6 | 407 |
| `Match_InterleavedAddMatch` | 8 333.7 | **120 K (add+match 配对)** | 8 | 308 |
| `PlaceOrder_EmptyBook` | — | OOM skip | — | — |

> `EmptyBook` 每次 iteration 创建一个新 market，30k 个 book 在 2vCPU/8GB VM 上直接 OOM (engine.test RSS 飙到 6 GB 被 oom-killer 干掉)。生产不会出现 30k 全新 market 连续挂单的路径，本次跳过。

## 关键结论

1. **单 book 撮合热路径上限 ≈ 1.2–1.5 M ops/s** (单线程)。撮合引擎本身不会是瓶颈。
2. **Phase 5 epoch + deterministic trade-id 开销仅 ~72 ns (+9%)**  
   `CrossSpread` 787 ns → `CrossSpread_WithEpoch` 859 ns。其中 `DeterministicTradeID` 自身 ~31 ns，其余 ~41 ns 来自 `localSeq` 自增 + 两次写入。
3. **多 book 分发路径 (supervisor → per-book ringbuffer) 掉一个数量级**  
   `MultiBook100` 5.9 µs vs 单 book `DeepBook` 0.66 µs，**9 倍开销**。真实生产负载(成百上千 market)卡点大概率在 supervisor routing 而不是撮合本身。
4. `STPSkip` 5.9 µs 比正常撮合还慢 — 每次 match 要循环 pop / push maker 而不是简单穿价。如果生产有高 STP 比例，这是个潜在优化点。
5. Cancel 7.8 µs 比下单慢，和 map 查找 + heap 调整成本一致。

## 原始数据 (per-iteration)

```
BenchmarkAddOrder_Fresh-2               100000    318.8 ns/op    256 B/op    4 allocs/op
BenchmarkAddOrder_Fresh-2               100000    247.7 ns/op    256 B/op    4 allocs/op
BenchmarkAddOrder_Fresh-2               100000    244.0 ns/op    256 B/op    4 allocs/op

BenchmarkPlaceOrder_DeepBook-2          500000    586.1 ns/op    490 B/op    9 allocs/op
BenchmarkPlaceOrder_DeepBook-2          500000    799.9 ns/op    490 B/op    9 allocs/op
BenchmarkPlaceOrder_DeepBook-2          500000    589.4 ns/op    490 B/op    9 allocs/op

BenchmarkMatch_CrossSpread-2            500000    775.9 ns/op    490 B/op    9 allocs/op
BenchmarkMatch_CrossSpread-2            500000    875.7 ns/op    490 B/op    9 allocs/op
BenchmarkMatch_CrossSpread-2            500000    709.6 ns/op    490 B/op    9 allocs/op

BenchmarkMatch_CrossSpread_WithEpoch-2  500000    764.4 ns/op    490 B/op    9 allocs/op
BenchmarkMatch_CrossSpread_WithEpoch-2  500000    909.9 ns/op    490 B/op    9 allocs/op
BenchmarkMatch_CrossSpread_WithEpoch-2  500000    902.7 ns/op    490 B/op    9 allocs/op

BenchmarkMatch_IOC_SweepBook-2          500000    875.2 ns/op    490 B/op    9 allocs/op
BenchmarkMatch_IOC_SweepBook-2          500000    809.0 ns/op    490 B/op    9 allocs/op
BenchmarkMatch_IOC_SweepBook-2          500000    882.0 ns/op    490 B/op    9 allocs/op

BenchmarkPlaceOrder_MultiBook100-2      500000    6523 ns/op     271 B/op    6 allocs/op
BenchmarkPlaceOrder_MultiBook100-2      500000    5628 ns/op     271 B/op    6 allocs/op
BenchmarkPlaceOrder_MultiBook100-2      500000    5616 ns/op     271 B/op    6 allocs/op

BenchmarkMatch_InterleavedAddMatch-2    200000    8310 ns/op     308 B/op    8 allocs/op
BenchmarkMatch_InterleavedAddMatch-2    200000    8943 ns/op     308 B/op    8 allocs/op
BenchmarkMatch_InterleavedAddMatch-2    200000    7748 ns/op     308 B/op    8 allocs/op

BenchmarkCancelOrders-2                 200000    7199 ns/op     407 B/op    6 allocs/op
BenchmarkCancelOrders-2                 200000    7972 ns/op     407 B/op    6 allocs/op
BenchmarkCancelOrders-2                 200000    8268 ns/op     407 B/op    6 allocs/op

BenchmarkMatch_STPSkip-2                500000    6021 ns/op     639 B/op    5 allocs/op
BenchmarkMatch_STPSkip-2                500000    5579 ns/op     639 B/op    5 allocs/op
BenchmarkMatch_STPSkip-2                500000    5964 ns/op     638 B/op    5 allocs/op

BenchmarkDeterministicTradeID-2        5000000   30.80 ns/op      16 B/op    1 allocs/op
BenchmarkDeterministicTradeID-2        5000000   30.31 ns/op      16 B/op    1 allocs/op
BenchmarkDeterministicTradeID-2        5000000   32.28 ns/op      16 B/op    1 allocs/op
```

## 注意事项 / 方法论局限

- **2 vCPU VM**：数据可和同机多次跑对比，但**不代表**生产多核（无论多 book 并发还是 GC 吞吐都有偏差）。
- **纯引擎层**：不含 Kafka → gateway → ringbuffer → engine → kafka-out 的端到端链路；也不含 DB 落盘。
- **波动较大**：同一 benchmark 3 次跑 ns/op 波动 20% 左右，疑似共享宿主干扰 (`load average ~0.5`) — 做 A/B 对比时需要多 count 平均。
- **`EmptyBook` 不可复现**：需要 supervisor / engine 内部对空 market 懒加载 (或 benchmark 改造) 才能压它。

---

# 第二轮：E2E Kafka 压测 (2026-04-12 晚, 部署修复后)

## 前置修复

部署链路修复完成：

1. **GitHub Actions 侧**：push `83c4eca` 手动触发部署 → 走的是已修好的 entrypoint → 成功
2. **服务器侧 entrypoint 自愈补丁**：`/usr/local/bin/funnyoption-staging-deploy` 在 `require_command git` 后插入 5 行幂等的 `git config --global --add safe.directory "${REPO_PATH}"`，解决 rsync 改 owner 后 root 跑 git 遭遇 dubious-ownership 的问题。老脚本备份 `funnyoption-staging-deploy.bak-20260412`
3. **服务器侧 dirty tree 处理**：`/opt/funnyoption-staging` 上的 51 个 custody 相关 uncommitted 改动 → `git stash push -u -m "pre-deploy salvage 2026-04-12 (server-side custody wip)"`，可恢复

部署确认：`funnyoption-staging-matching-1` Created `2026-04-11T17:27:18Z`，source 跑在 `83c4eca` (含 `3823f1e` 热路径优化)。

## 压测工具

新增 `backend/cmd/kafka-bench/main.go`:
- 直接往 Kafka `funnyoption.staging.order.command` topic 灌 `sharedkafka.OrderCommand`
- 消费 `funnyoption.staging.trade.matched`，按 TakerOrderID 里嵌入的 nanos 算 E2E 延迟
- 跑在 staging compose 网络里: `docker run --rm --network funnyoption-staging_default ... --entrypoint /kafka-bench alpine:3.20 ...`
- 静态 linux/amd64 二进制：`CGO_ENABLED=0 go build`
- 两种模式：
  - **cross-match**（默认）：seed N 层 maker SELL，blast IOC BUY 全穿价成交
  - **`--no-match`**：taker 以远低于最低 ask 的价格下 GTC，纯挂单不成交（为了绕开下面提到的 DB FK 问题）

## 发现 1：dispatcher 落库层有严重的 FK 问题 (非本次任务内 fix，值得单独追)

**症状**：cross-match 路径下，每个 trade 落库都会 hit
```
pq: insert or update on table "trades" violates foreign key constraint "trades_taker_order_id_fkey"
```
- 第一轮 smoke (500 单)：22/500 侥幸成功，478 失败
- 第二轮 5000 单：**4999/5000 全部失败**，只有 1 笔穿过

**定位**：`posttrade.Service.ProcessResult` → `sql_store.PersistResult` 在**同一个事务**里先 `upsertOrder(taker)` 再 `insertTrade(...)`。按理 PG 允许同事务内读自己未 commit 的插入，FK 应该看见 orders 行。但实测大部分 tx 整体回滚，orders 表里也没有 taker 的行（查过 `seq 1/497/498` 均不存在；`seq 0/499` 存在且 FILLED）。

**建议**：另开任务追这个 bug。候选方向：
- 确认 `result.Order` 在 IOC 全填场景下是否总是非 nil（engine.go 里是，但可能有别的路径把它置空）
- 检查 posttrade 是否在 `buildTradeEvent` 之前有别的会 error 的子步骤导致 tx 回滚前 `trades_taker_order_id_fkey` 误报
- 检查 DB FK 是否是 `DEFERRABLE` + posttrade 是否错误地用了 `NOT DEFERRED`
- 如果 dispatcher / posttrade 有并发 worker（我没彻底排除），pin 到单线程再测

**对压测的影响**：cross-match 跑不了 consumer 端的 trade 延迟 ——`posttrade.Service.ProcessResult` 在 DB 失败后直接 `return err`，根本走不到 `s.publisher.PublishJSONBatch`，所以失败的 trade 不会上 Kafka。绕路方法：用 `--no-match` 模式，或按 pipeline stats 日志里的 counter 推算真实吞吐。

## 发现 2：E2E 吞吐的真正瓶颈是 dispatcher 落库，不是撮合引擎

用 `--no-match` 模式跑 5000 和 10000 单，观察 matching 容器 pipeline stats 日志里的 counter 变化：

| bench | 送达时间 | send t'put | gw_received 增量 | sv_matched 增量 | gw_paused 增量 | disp_dispatched 增量/s (稳态) |
|---|---|---:|---:|---:|---:|---:|
| no-match 5000 c=8 | 1.93s | 2 593/s | 5 020 | 5 020 | 0 | **~68/s** (稳态 drain 速率) |
| cross 5000 c=8 | 1.93s | 2 591/s | 5 020 | 5 020 | 0 | 立即等量 (全部 FK 回滚、快失败) |
| no-match 10000 c=32 | 1.53s | **6 547/s** | 10 020 | 10 020 | **10 713** | ~70/s |

**关键洞察**：

1. **Client send rate (c=32) 可以飙到 ~6 500 orders/s** —— 单 client、8 goroutine 到 32 goroutine 线性放大 (c=1 → 372, c=8 → 2 591, c=32 → 6 547)
2. **Gateway + engine 路径的 Kafka 消费速率 ~500–1 000 orders/s** —— 从日志 10s 粒度 delta 推算。这个是 **Kafka → JSON decode → gateway route → engine PlaceOrder** 的总和
3. **Engine 自身几乎没有负载**：in-process bench 我们测到 1.27 M ops/s，而 E2E 路径只有 ~500 ops/s —— 说明 99.96% 的开销不在 engine，而在 **Kafka fetch + JSON decode + single-goroutine gateway 串行路径**
4. **Dispatcher 成功落库路径只有 ~68-72 orders/s** —— 被 PG tx fsync + `PersistResult` 里的 `upsertOrder * (1+affected) + insertTrade * N + rollup append` 拖死。这是 E2E 最慢的环节
5. **观察到 backpressure**：c=32 高速发时 `gw_paused=10 713`，意味着 input ring buffer (size 8 192) 溢出了，gateway 开始 idle-spin 等待消费侧

## 发现 3：latency 样本（仅限落库成功的那部分）

| 来源 | n | mean | p50 | p90 | p95 | p99 | max |
|---|---:|---:|---:|---:|---:|---:|---:|
| smoke 500 c=1 (cross) | 22 | 16.9 ms | 16.2 ms | 17.8 ms | 18.4 ms | 20.2 ms | 25.1 ms |

这 22 个样本是 cross-match 路径下**侥幸穿过 FK 的那批**，延迟包括：bench 客户端发 Kafka → redpanda → matching gateway 消费 → engine PlaceOrder → dispatcher 落库 → Kafka publish → bench consumer 收到。**16 ms 左右的 p50 是真实 E2E 延迟**（包含 DB fsync）。

`--no-match` 模式因为没有 trade，根本不会走到 `trade.matched` topic，consumer 看不到任何消息 → 无法算延迟。如果想在修掉 FK 之前拿到更大的 latency 样本，可以：
- 加 `--pg-dsn`，在 bench 开跑前先把 taker 的 orders 行预插进去（绕 FK）
- 或者直接改用 `order.event` topic 的 NEW 事件时间戳来算 gateway 到 dispatcher 的延迟（不含 publish 回传）

## 产物与位置

- 源码: `backend/cmd/kafka-bench/main.go` (in-tree, 已加 `--no-match` flag)
- 服务器二进制: `/tmp/kafka-bench` (静态 linux/amd64, 6.8 MB)
- 服务器源码 tarball: `/tmp/funnyoption-bench/` (包含 backend + bench 源码)
- Entrypoint 补丁: `/usr/local/bin/funnyoption-staging-deploy` (已打，旧版 `.bak-20260412`)
- 部署日志: `/tmp/deploy-2026-04-12-83c4eca.log`
- Bench 原始输出: 各次 `docker run` stdout，未单独保存（如果需要回看可以重跑）

## 后续 TODO

- [x] ~~**阻塞**: 追 `trades_taker_order_id_fkey` 的根因~~ → **已修复** (slice aliasing bug)
- [x] ~~把 stash 整理成 feature branch~~ → `salvage/server-custody-wip` 服务器本地分支已创建
- [x] ~~给 bench 加 `--pg-dsn` + pre-insert~~ → 已实现，cross-match 5000/5000 全部成功
- [x] ~~push 一次 commit 验证 CI/CD~~ → 全链路打通（known_hosts / safe.directory / stash 三个问题均修复）
- [x] ~~前后对比~~ → 见下方对比表
- [x] ~~pprof 采样~~ → Snapshot 是 83% CPU 热点，已修复
- [x] ~~EmptyBook OOM~~ → bench 改为 fresh engine/iteration，GC 回收正常
- [ ] E2E gateway fan-out / dispatcher 落库优化 — 后续迭代
- [ ] CI/CD SSH 偶发 timeout — GitHub runner IP 不固定，需要防火墙放行或改用 self-hosted runner

---

# 修复总结 (2026-04-12)

## 本次修复的 7 个 Commits

| # | Commit | 类型 | 说明 |
|---|---|---|---|
| 1 | `d244cb2` | **fix** | **P0: 修复 trade 数据被后续 PlaceOrder 覆写** — `engine.match()` 的 `tradesBuf` reusable slice 通过 channel 传给 dispatcher 时被下一次 PlaceOrder 覆写，导致 `trades_taker_order_id_fkey` FK 100% 失败。修复：return 前 copy 出独立 slice |
| 2 | `1ddcdd1` | **perf** | Gateway 批量 Kafka commit — `CommitInterval=200ms` + `MaxWait=10ms` + 256 条批量提交，消费速率 460→980/s (2x) |
| 3 | `fb9307b` | **feat** | 新增 `backend/cmd/kafka-bench` E2E 压测工具 — 直接往 Kafka 灌 OrderCommand，消费 trade.matched 测延迟 |
| 4 | `8073995` | **docs** | 压测报告 + 前后对比数据 |
| 5 | `a685820` | **feat** | kafka-bench 加 `--pg-dsn` pre-insert — 确定性 OrderID + 预插 taker 行 + sentAt[] 内存延迟追踪 |
| 6 | `0e2f559` | **perf** | **Snapshot 用 bitmap 遍历替代线性扫描** — InterleavedAddMatch 7600→1036 ns (7.3x)，CancelOrders 7600→818 ns (9.3x) |
| 7 | `20169a6` | **fix** | EmptyBook benchmark OOM — 改为 fresh engine/iteration，GC 回收 |

## CI/CD 修复（服务器侧，非 git commit）

| 问题 | 根因 | 修复 |
|---|---|---|
| `repo checkout not found or not a git work tree` | `/opt/funnyoption-staging` 被 rsync 改成 `501:staff` owner → git dubious-ownership 拒绝 | 在 `/usr/local/bin/funnyoption-staging-deploy` 加 5 行幂等 `safe.directory` 自愈补丁 |
| `tracked git changes exist` | 服务器上 51 个 custody 相关 uncommitted 改动 | `git stash push -u` → 创建 `salvage/server-custody-wip` 分支保留 |
| SSH `install -m 700 -d ~/.ssh` exit 1 | `STAGING_SSH_KNOWN_HOSTS` secret 为空 → `ssh-keyscan` 从 runner 扫服务器被防火墙挡 | 配置 `STAGING_SSH_KNOWN_HOSTS` secret 为服务器的 3 种 host key |

## 最终 Benchmark 对比 (同机 2vCPU EPYC, GOMAXPROCS=2, count=3 均值)

### 三代对比: e86fca4 → 83c4eca → 20169a6

| Benchmark | e86fca4 (旧) | 83c4eca (hot-path优化) | **20169a6 (全部修复)** | 总提速 (vs e86fca4) |
|---|--:|--:|--:|---|
| `AddOrder_Fresh` | — | 270 ns | **333 ns** | — |
| `PlaceOrder_DeepBook` | 1 438 ns | 658 ns | **834 ns** | 1.7x |
| `Match_CrossSpread` | 1 633 ns | 787 ns | **789 ns** | **2.1x** |
| `CrossSpread_WithEpoch` | 1 621 ns | 859 ns | **674 ns** | **2.4x** |
| `IOC_SweepBook` | — | 855 ns | **876 ns** | — |
| **`InterleavedAddMatch`** | 1 809 ns | ~~7 631 ns~~ (回退) | **1 036 ns** | **1.7x** |
| **`CancelOrders`** | 936 ns | ~~7 597 ns~~ (回退) | **818 ns** | **1.1x** |
| `MultiBook100` | — | 5 922 ns | **817 ns** | — |
| `STPSkip` | — | 5 855 ns | **1 680 ns** | — |
| `EmptyBook` | OOM | OOM | **218 µs** (含 book 创建) | ✅ 可跑 |
| `DeterministicTradeID` | — | 31 ns | **35 ns** | — |

> `83c4eca` 的 InterleavedAddMatch/CancelOrders 回退是因为 `Snapshot()` 用 O(maxPrice) 线性扫描。`20169a6` 用 bitmap 遍历修复后，**全面超越 e86fca4**。

### E2E Kafka 压测最终数据 (cross-match 5000 单, `--pg-dsn` pre-insert)

| 指标 | 值 |
|---|---|
| orders sent | 5 000 |
| trades observed | **5 000 (100%)** |
| disp_errors | **0** |
| send throughput | 2 287 orders/s (c=8) |
| matching throughput | 56 trades/s (dispatcher DB limited) |
| **latency p50** | 43.8 s (队列排队延迟，非引擎延迟) |
| latency p99 | 1m26.9s |

> 延迟大是因为引擎瞬间处理完 5000 单，但 dispatcher 只有 ~56/s 的 DB fsync 速度。真正的引擎撮合延迟 sub-ms。

### E2E 吞吐分层分析

```
Client (c=32)  →  6 547 orders/s
    ↓ Kafka produce
Gateway        →  ~980 orders/s   ← batch commit 优化后 2x
    ↓ ringbuffer route
Engine         →  ~1M+ ops/s      ← 不是瓶颈
    ↓ output channel
Dispatcher     →  ~56 trades/s    ← DB fsync，是 E2E 瓶颈
    ↓ Kafka publish
Consumer       →  观测到的 trade
```

## pprof 发现

**InterleavedAddMatch CPU 分布 (优化前)**:
- `OrderBookDirect.Snapshot` — **83%** (线性扫描 10000 个 bucket slot)
- `Engine.match` — 5%
- GC — 8%

**修复后**: Snapshot 使用 `FirstBidBucket/NextBidBucket` bitmap 跳跃，O(limit) 而非 O(maxPrice)

**MultiBook100 CPU 分布**:
- `Snapshot` — 71%
- `getOrCreateBook` (map lookup) — 8%
- `NewOrderBookDirect` — 7%

## 残留问题

1. **CI/CD SSH timeout** — GitHub Actions runner IP 不固定，偶发连不上服务器 port 22。解法：防火墙放行 GitHub Actions IP 段，或改用 self-hosted runner
2. **Dispatcher 落库 ~56/s** — E2E 真正瓶颈。每笔 cross-match trade 需要 upsert 2 行 orders + insert 1 行 trades + rollup append，受 PG fsync 限制。优化方向：批量 persist、async write-ahead、或 WAL-only 模式
3. **`salvage/server-custody-wip` 未 push 到 origin** — 服务器没有 GitHub push 权限，需要从本地 fetch 后 push
4. **Gateway 进一步优化** — 当前单 goroutine fetch 已提到 ~980/s，如需更高可考虑 fan-out decode + 多 partition 并行消费
