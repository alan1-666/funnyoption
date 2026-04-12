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

- [ ] **阻塞**: 追 `trades_taker_order_id_fkey` 的根因 —— 这是一个 posttrade/sql_store 的真实 bug，cross-match 路径几乎 100% 失败率，生产风险等级很高
- [ ] 把 `-serverside custody wip` 那个 stash 整理成一个 feature branch 推到 origin，让服务器的 /opt/funnyoption-staging 恢复 clean state
- [ ] 给 bench 加 `--pg-dsn` + pre-insert orders 行，让 cross-match 路径能拿到大样本 latency
- [ ] push 一次 commit 到 main 验证 GitHub Actions 能走通整条链路（我修了服务器端，但没实际 push 过东西触发 runner）
- [ ] 把 cmd/kafka-bench 补个 README 或注释说明怎么跑在 staging compose 网络里
- [ ] E2E gateway 的单 goroutine decode 路径明显是瓶颈，考虑 fan-out 或换更快的 JSON 库 (goccy/go-json 已经用上了但仍 bottleneck)
- [x] `83c4eca` vs `e86fca4` in-process 前后对比 — 见下方对比表
## 前后对比: `3823f1e feat(matching): optimize hot path` 效果

同一服务器 (2vCPU EPYC, GOMAXPROCS=2) 背靠背跑，排除负载差异。

| Benchmark | e86fca4 ns/op | 83c4eca ns/op | Delta | Allocs | B/op |
|---|--:|--:|---|---|---|
| `PlaceOrder_DeepBook` | 1 438 | **632** | **-56% (2.3x faster)** | 16→9 | 1376→490 |
| `Match_CrossSpread` | 1 633 | **800** | **-51% (2.0x faster)** | 16→9 | 1376→490 |
| `Match_CrossSpread_WithEpoch` | 1 621 | **749** | **-54% (2.2x faster)** | 16→9 | 1376→490 |
| `Match_InterleavedAddMatch` | **1 809** | 7 631 | **+322% (4.2x 回退)** | 16→8 | 1200→308 |
| `CancelOrders` | **936** | 7 597 | **+711% (8.1x 回退)** | 7→6 | 423→407 |

**解读**：
- 简单匹配场景（DeepBook、CrossSpread）**提速 2x+**，allocs 从 16 降到 9，B/op 从 1376 降到 490 — 这是 `OrderBookDirect` (bucket linked-list) 替换旧 map-based 结构的收益
- InterleavedAddMatch 和 CancelOrders **出现严重回退** (4-8x 慢)，尽管 allocs/B 更低。疑似 Cancel 在新的 linked-list 结构上变成 O(N) 查找，InterleavedAddMatch 因为每对 add+match 都涉及 cancel-by-fill，被同样的 O(N) 拖慢
- **建议**：查 `OrderBookDirect.RemoveDirectOrder` 路径是否做了线性扫描，考虑加 order-id→node 的 map 做 O(1) cancel

- [ ] 多 book 路径 profile — 确认 MultiBook100 的 5.9 µs 是 map lookup 还是 ringbuffer CAS
- [ ] 用 `pprof` 采样一次 `InterleavedAddMatch`，看 alloc / GC 占比
- [ ] `EmptyBook` 不可复现 —— 需要 supervisor/engine 内部对空 market 懒加载 (或 benchmark 改造) 才能压它
