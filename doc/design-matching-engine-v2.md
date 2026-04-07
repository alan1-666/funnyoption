# High-Performance Matching Engine V2 — Design Document

> Status: **DRAFT** | Author: zhangza | Date: 2026-04-07

---

## 1. Executive Summary

Redesign the matching engine from the current "Kafka consumer → match → DB persist → Kafka publish" serial loop into an **Aeron-inspired**, **Ring Buffer-isolated**, **zero-IO matching thread** architecture. The goal is to push single-book throughput from ~2,000 orders/sec (current V1, limited by downstream I/O blocking) to **50,000+ orders/sec** per book, with sub-100μs matching latency (p99), while maintaining deterministic replay and near-zero-downtime failover.

### Design Principles (borrowed from Aeron)

| Aeron Concept | Our Adaptation |
|---------------|----------------|
| SPSC Ring Buffer (lock-free) | Go `RingBuffer[T]` with atomic read/write cursors, pre-allocated slots |
| Publication / Subscription | `InputGateway` / `OutputDispatcher` — dedicated I/O goroutines |
| Conductor (lifecycle coordinator) | `BookSupervisor` — manages book worker creation, snapshot, shutdown |
| Idle Strategy (busy-spin → yield → park) | Go `IdleStrategy` with `runtime.Gosched()` → `time.Sleep` backoff |
| Archive & Replay | Kafka retention + snapshot for deterministic restore |
| Cluster (Raft-based HA) | Primary-Standby with deterministic input replay |

---

## 2. Current V1 Architecture & Bottleneck Analysis

### 2.1 Current Flow

```
API → Kafka(order.command) → [matching-service single consumer goroutine]
                                     │
                                     ▼
                              CommandProcessor.HandleOrderCommand
                                     │
                              ┌──────┴──────┐
                              │ Market cache │  ← DB read (5s TTL)
                              └──────┬──────┘
                                     ▼
                              AsyncEngine.Submit
                                     │
                              bookWorker goroutine (ch buffered 2048)
                                     │
                              Engine.PlaceOrder  ← in-memory, fast
                                     │
                              ┌──────┴──────┐
                              │  Result     │
                              └──────┬──────┘
                                     ▼
                              SQLStore.PersistResult  ← DB transaction 🔴
                                     │
                                     ▼
                              PublishJSONBatch        ← Kafka write 🔴
                                     │
                                     ▼
                              CommitMessages          ← Kafka commit 🔴
```

### 2.2 Identified Bottlenecks

| # | Bottleneck | Impact | Current Cost |
|---|-----------|--------|-------------|
| B1 | **串行 consumer loop** — 单 goroutine 处理所有 book 的命令，DB 持久化和 Kafka 发布阻塞后续命令 | 全局吞吐量瓶颈 | ~2-5ms per command |
| B2 | **撮合线程等待 DB** — `PersistResult` 在 consumer 主循环内同步执行 | 撮合延迟被 DB RTT 拉高 | ~1-3ms per persist |
| B3 | **撮合线程等待 Kafka** — `PublishJSONBatch` 同步等待 broker ack | 同上 | ~1-5ms per batch |
| B4 | **JSON 序列化** — 热路径全程 `json.Marshal/Unmarshal` | CPU 开销 + GC 压力 | ~50-200μs per message |
| B5 | **sorted slice 插入** — `OrderBook.Bids/Asks` 用 `sort.Search` + slice shift | 深度大时 O(n) | 可忽略(当前深度浅) |
| B6 | **无反压机制** — bookWorker channel 满则阻塞 consumer | 突发流量下延迟飙升 | buffer=2048, 无监控 |

### 2.3 V1 的正确性优势(需保留)

- ✅ 单写者模型：每个 BookKey 只有一个 goroutine 操作
- ✅ 确定性：相同输入顺序 → 相同 OrderBook 状态
- ✅ Price-time priority 正确实现
- ✅ Self-trade prevention
- ✅ 全局 trade sequence (atomic counter)

---

## 3. V2 Target Architecture

### 3.1 High-Level Topology

```
                         ┌─────────────────────────────────────────────────┐
                         │            Matching Engine Process              │
                         │                                                 │
  Kafka                  │  ┌──────────────┐   ┌──────────────────────┐   │
  order.command ────────►│  │ Input Gateway │   │   Book Supervisor    │   │
  (partitioned by book)  │  │ (IO goroutine)│   │ (lifecycle manager)  │   │
                         │  └──────┬───────┘   └──────────────────────┘   │
                         │         │                                       │
                         │         ▼                                       │
                         │  ┌──────────────┐                               │
                         │  │ Input Ring   │  SPSC, pre-allocated          │
                         │  │ Buffer       │  capacity: 64K slots          │
                         │  └──────┬───────┘                               │
                         │         │                                       │
                         │         ▼                                       │
                         │  ┌──────────────────────┐                       │
                         │  │  Matching Thread      │  pure computation    │
                         │  │  (single goroutine)   │  no I/O, no alloc   │
                         │  │                       │  no locks, no syscall│
                         │  └──────┬───────────────┘                       │
                         │         │                                       │
                         │         ▼                                       │
                         │  ┌──────────────┐                               │
                         │  │ Output Ring  │  SPSC, pre-allocated          │
                         │  │ Buffer       │  capacity: 64K slots          │
                         │  └──────┬───────┘                               │
                         │         │                                       │
                         │         ▼                                       │
                         │  ┌────────────────┐                             │
                         │  │Output Dispatcher│  (IO goroutine)            │
                         │  │                │                             │
                         │  │ ┌─ Kafka batch publisher                     │
                         │  │ ├─ DB batch persister (async)                │
                         │  │ └─ Metrics emitter                           │
                         │  └────────────────┘                             │
                         │                                                 │
                         └─────────────────────────────────────────────────┘
                                      │
                                      ▼
                                   Kafka
                           (order.event, trade.matched,
                            quote.depth, quote.ticker,
                            position.changed)
```

### 3.2 Per-Book Sharding Model

```
Kafka Topic: order.command (N partitions)
  partition 0 ──► BookEngine[BTC:YES]   ← 独立 Input RB + Matching Thread + Output RB
  partition 1 ──► BookEngine[BTC:NO]
  partition 2 ──► BookEngine[ETH:YES]
  ...
  partition N ──► BookEngine[DOGE:NO]

每个 BookEngine 是完全独立的 goroutine 集合:
  - 1 Input Gateway goroutine
  - 1 Matching goroutine
  - 1 Output Dispatcher goroutine
  共 3 goroutines per active book
```

**分区策略**：Kafka producer 用 `BookKey()` 作为 partition key，保证同一 book 的所有命令落在同一 partition。Matching service 通过 Kafka consumer group 自动分配 partition，一个 matching 实例可以服务多个 book。

### 3.3 Ring Buffer Design (Aeron SPSC 风格)

```go
// 核心思路：单生产者单消费者，无锁，cache-line padding 防 false sharing
type RingBuffer[T any] struct {
    _        [64]byte        // cache line padding
    write    atomic.Uint64   // 只被 producer 修改
    _        [56]byte        // padding between write and read
    read     atomic.Uint64   // 只被 consumer 修改
    _        [56]byte
    mask     uint64          // capacity - 1, capacity 必须是 2 的幂
    slots    []T             // pre-allocated, 大小 = capacity
}

func (rb *RingBuffer[T]) TryPublish(item T) bool {
    w := rb.write.Load()
    r := rb.read.Load()
    if w - r >= rb.mask + 1 {
        return false // full — 反压信号
    }
    rb.slots[w & rb.mask] = item
    rb.write.Store(w + 1) // release semantics via Store
    return true
}

func (rb *RingBuffer[T]) TryConsume() (T, bool) {
    r := rb.read.Load()
    w := rb.write.Load()
    if r >= w {
        var zero T
        return zero, false // empty
    }
    item := rb.slots[r & rb.mask]
    rb.read.Store(r + 1)
    return item, true
}

// 批量消费 — 减少 atomic 操作次数
func (rb *RingBuffer[T]) DrainTo(dst []T, maxItems int) int {
    r := rb.read.Load()
    w := rb.write.Load()
    available := int(w - r)
    if available == 0 {
        return 0
    }
    n := min(available, maxItems)
    for i := 0; i < n; i++ {
        dst[i] = rb.slots[(r + uint64(i)) & rb.mask]
    }
    rb.read.Store(r + uint64(n))
    return n
}
```

**为什么不用 Go channel？**

| 维度 | `chan T` | `RingBuffer[T]` |
|------|---------|-----------------|
| 延迟 | ~100-500ns (mutex + futex) | ~10-30ns (atomic only) |
| 批量消费 | 不支持 | `DrainTo` 一次读 N 条 |
| 反压信号 | 阻塞或丢弃 | `TryPublish` 返回 false |
| 内存分配 | 运行时按需 | 预分配，零 GC |
| Cache 利用 | 差 (链表 + 指针) | 连续内存，预取友好 |

### 3.4 Idle Strategy (Aeron 风格)

撮合线程在无消息时的等待策略直接影响 CPU 利用和尾延迟：

```go
type IdleStrategy struct {
    spinCount   int  // phase 1: busy spin (最低延迟)
    yieldCount  int  // phase 2: runtime.Gosched()
    parkNanos   int  // phase 3: time.Sleep (让出 CPU)
    state       int
    counter     int
}

func (s *IdleStrategy) Idle(workCount int) {
    if workCount > 0 {
        s.counter = 0
        s.state = 0
        return
    }
    s.counter++
    switch s.state {
    case 0: // spin
        if s.counter > s.spinCount {
            s.state = 1
            s.counter = 0
        }
    case 1: // yield
        runtime.Gosched()
        if s.counter > s.yieldCount {
            s.state = 2
            s.counter = 0
        }
    case 2: // park
        time.Sleep(time.Duration(s.parkNanos))
    }
}
```

**推荐参数**：`spinCount=100, yieldCount=10, parkNanos=1000` (1μs)
- 活跃市场: 几乎不离开 spin phase，延迟 ~10ns
- 冷门市场: 快速退化到 park，CPU 占用 <1%

### 3.5 消息编码：从 JSON 到 FlatBuffer/Binary

热路径的 JSON 开销可观。V2 在**进程内**使用定长二进制编码：

```go
// 进程内 Ring Buffer 消息格式 — 固定大小，零分配
type MatchCommand struct {
    Type       uint8     // 1: PlaceOrder, 2: CancelOrder, 3: Snapshot
    MarketID   int64
    Outcome    [8]byte   // "YES\0\0\0\0\0" or "NO\0\0\0\0\0\0"
    OrderID    [48]byte  // fixed-length order ID
    UserID     int64
    Side       uint8     // 1: BUY, 2: SELL
    OrderType  uint8     // 1: LIMIT, 2: MARKET
    TimeInForce uint8    // 1: GTC, 2: IOC
    Price      int64     // 价格以 cent 为单位，整数
    Quantity   int64
    Nonce      int64
    Timestamp  int64     // 纳秒级时间戳
}
// sizeof = ~128 bytes, cache-line friendly

type MatchResult struct {
    Type        uint8    // 1: OrderAccepted, 2: Trade, 3: OrderRejected, 4: DepthUpdate
    MarketID    int64
    // ... 按类型解释后续字段
    Payload     [256]byte // union-style, 根据 Type 解析
}
```

**外部通信(Kafka)仍用 JSON** — 下游服务不需要改变。编解码边界在 Input Gateway 和 Output Dispatcher。

---

## 4. Component Detail Design

### 4.1 Input Gateway

```
Kafka Consumer ──► JSON Decode ──► Build MatchCommand ──► RingBuffer.TryPublish
                   (IO goroutine)
```

**职责**：
- 消费 Kafka `order.command` 分区
- JSON 反序列化 → `MatchCommand` 二进制结构
- 发布到 Input Ring Buffer
- 如果 Ring Buffer 满(反压)：记录指标，短暂 yield 后重试
- **不做任何业务逻辑**

**Kafka commit 策略**：
- 延迟提交(deferred commit)：Input Gateway 记录每个消息的 offset
- 当 Output Dispatcher 确认该消息的结果已 persist，才提交对应 offset
- 保证 at-least-once delivery + 精确重放能力

### 4.2 Matching Thread

```
loop:
    batch = InputRB.DrainTo(buf, 64)  // 批量读，减少 atomic 次数
    for cmd in batch:
        result = engine.Process(cmd)
        OutputRB.TryPublish(result)
    idleStrategy.Idle(batch.len)
```

**约束**：
- **禁止任何 I/O**：不读 DB、不写 Kafka、不做 HTTP 调用
- **禁止 `sync.Mutex`**：Ring Buffer 是唯一的同步机制
- **禁止 `make`/`new` 在热路径**：预分配所有结构
- **禁止 `log` 在热路径**：错误通过结果消息传递，由 Output Dispatcher 记录
- **允许 `atomic` 操作**：全局 trade sequence counter

**引擎内部改进**：

| 当前 V1 | V2 方案 | 理由 |
|---------|--------|------|
| `[]DepthLevel` + `sort.Search` | 侵入式红黑树或跳表 | O(log n) 插入/删除，无 slice shift |
| `map[string]*Order` | `map[string]*Order` (保留) | Go map 对小 key 已经够快 |
| `json.Marshal` per trade | 二进制 `MatchResult` 直接写 OutputRB | 零序列化开销 |
| `book.Snapshot(5)` 每次 | 增量 depth diff | 减少快照开销 |

### 4.3 Output Dispatcher

```
loop:
    batch = OutputRB.DrainTo(buf, 256)  // 大批量读
    for result in batch:
        classify(result) → append to topic-specific batch buffers
    flush all batch buffers:
        - Kafka: PublishJSONBatch (已有, 可复用)
        - DB:    Batch INSERT trades + UPSERT orders (async, 可延迟)
        - WS:    depth/ticker push (通过 Kafka → ws-service)
    idleStrategy.Idle(batch.len)
```

**关键设计**：
- DB 持久化从**同步阻塞**变为**异步批量**
- 可配置 `flushIntervalMs` (默认 10ms) 和 `maxBatchSize` (默认 256)
- DB 写入失败不阻塞撮合，但触发告警 + 进入重试队列
- Kafka 写入同理

**持久化一致性保证**：
- 撮合结果先写 Output Ring Buffer（已确定），后异步持久化
- 如果 persist 失败，服务重启后从 Kafka 重放，确定性引擎会产生相同结果
- 这就是为什么确定性是一切的基础

### 4.4 Book Supervisor

**生命周期管理**：
```go
type BookSupervisor struct {
    books     map[string]*BookEngine
    mu        sync.RWMutex
    sequence  atomic.Uint64  // 全局 trade sequence
}

type BookEngine struct {
    bookKey         string
    inputRB         *RingBuffer[MatchCommand]
    outputRB        *RingBuffer[MatchResult]
    engine          *Engine          // 纯内存撮合
    inputGateway    *InputGateway    // Kafka → InputRB
    outputDispatch  *OutputDispatcher // OutputRB → Kafka + DB
    idle            *IdleStrategy
}
```

**Supervisor 职责**：
- 按 Kafka partition assignment 动态创建/销毁 BookEngine
- Rebalance 时的优雅迁移：drain Ring Buffer → snapshot → 交接
- 定期 snapshot 用于 Standby 快速恢复
- 健康监控：Ring Buffer 水位、延迟指标、heartbeat

---

## 5. 高可用：Primary-Standby with Deterministic Replay

### 5.1 方案概述

```
Kafka Topic: order.command (partition P)
    │
    ├──► Server A (Primary)    → 消费 + 撮合 + 输出 Trade ✅
    ├──► Server B (Standby)    → 消费 + 撮合 + 不输出 ❌ (shadow mode)
    └──► Server C (Standby)    → 消费 + 撮合 + 不输出 ❌

    三台消费相同输入，维护相同 OrderBook 状态
    只有 Primary 的 Output Dispatcher 实际写入 Kafka/DB
```

### 5.2 为什么可行

**确定性保证**：
- 撮合引擎是纯函数：`f(OrderBook_state, Command) → (OrderBook_state', Result)`
- 没有外部 I/O、没有随机性、没有时间依赖(时间戳来自命令)
- 相同的命令序列 → 相同的 OrderBook 最终状态 → 相同的 Trade 序列

**所需前提**：
1. 所有节点从 Kafka 获取**完全相同顺序**的输入
2. 撮合逻辑中无 `time.Now()`、无 `rand`、无并发非确定性
3. Trade sequence 来自全局递增计数器，由 Primary 分配

### 5.3 故障切换协议

```
正常运行:
  Primary   → consume + match + OUTPUT (write Kafka/DB)
  Standby-1 → consume + match + DISCARD output
  Standby-2 → consume + match + DISCARD output

Primary 故障检测 (Kafka consumer group rebalance / 自定义心跳):
  1. Kafka rebalance 触发，Standby-1 获得 partition assignment
  2. Standby-1 的 OutputDispatcher 从 DISCARD 切换为 ACTIVE
  3. Standby-1 成为新 Primary
  4. 因为 OrderBook 已在 Standby-1 内存中保持一致，切换接近零延迟

Primary 恢复:
  1. 旧 Primary 重启
  2. 从新 Primary 请求 BookSnapshot (gRPC)
  3. 从 snapshot 对应的 Kafka offset 开始重放
  4. 追上后加入为新 Standby
```

### 5.4 Fencing (防脑裂)

- **Kafka consumer group** 天然保证每个 partition 只有一个 active consumer
- Output Dispatcher 在写入 Kafka 时附带 `epoch_id`(每次 rebalance 递增)
- 下游服务(account/settlement)验证 `epoch_id` 单调递增，拒绝旧 epoch

---

## 6. 增量 Depth 推送(替代全量快照)

### 6.1 当前问题

V1 每次撮合后执行 `book.Snapshot(5)` 并发送完整 5 档深度。对于高频市场，这意味着大量重复数据。

### 6.2 V2 方案

```go
type DepthDiff struct {
    Side    uint8   // BID / ASK
    Price   int64
    NewQty  int64   // 0 表示该档位消失
    Action  uint8   // 1: INSERT, 2: UPDATE, 3: DELETE
}
```

**推送策略**：
- 每次 match 后，引擎计算受影响的价位 diff
- Output Dispatcher 累积 diff 并按 `snapshotIntervalMs`(默认 100ms) 合并发送
- 下游 ws-service 维护本地 depth 镜像，apply diff
- 新连接首次获取全量快照，之后只接收 diff

**带宽节约**：典型一次 fill 只影响 1-2 个价位，diff 消息比全量快照小 ~80%。

---

## 7. Metrics & Observability

### 7.1 关键指标

| 指标 | 采集点 | 含义 |
|------|--------|------|
| `matching.input_rb.water_level` | InputGateway | Input Ring Buffer 使用率 |
| `matching.output_rb.water_level` | OutputDispatcher | Output Ring Buffer 使用率 |
| `matching.latency.match_ns` | Matching Thread | 单次 PlaceOrder 耗时(纳秒) |
| `matching.latency.e2e_us` | Input→Output | 从 InputRB 写入到 OutputRB 写入的端到端延迟 |
| `matching.throughput.orders_per_sec` | Matching Thread | 每秒处理命令数 |
| `matching.throughput.trades_per_sec` | Output Dispatcher | 每秒生成 Trade 数 |
| `matching.idle.spin_ratio` | IdleStrategy | spin phase 占比(反映负载) |
| `matching.persist.batch_size` | OutputDispatcher | DB 批量写入大小 |
| `matching.persist.lag_ms` | OutputDispatcher | 持久化延迟(Output → DB commit) |

### 7.2 告警阈值

| 条件 | 级别 | 动作 |
|------|------|------|
| `input_rb.water_level > 80%` | WARN | 反压开始，关注上游 |
| `input_rb.water_level > 95%` | CRITICAL | 可能丢单，立即扩容 |
| `match_ns p99 > 100μs` | WARN | 检查 book depth |
| `persist.lag_ms > 1000` | WARN | DB 慢，检查连接池 |
| `persist.lag_ms > 5000` | CRITICAL | 考虑降级为只写 Kafka |

---

## 8. Migration Path (V1 → V2)

### Phase 1: Ring Buffer 隔离 (1-2 weeks)

**目标**：撮合线程不再直接等待 DB 和 Kafka

```
V1:  Consumer → Match → DB → Kafka → Commit  (串行)
Ph1: Consumer → InputRB → Match → OutputRB → [DB+Kafka async] → Commit
```

**具体改动**：
1. 实现 `RingBuffer[T]` (SPSC, padding, atomic)
2. 将 `CommandProcessor.HandleOrderCommand` 拆分：
   - Input Gateway: JSON decode → `MatchCommand` → `InputRB.TryPublish`
   - Matching Loop: `InputRB.DrainTo` → `engine.PlaceOrder` → `OutputRB.TryPublish`
   - Output Dispatcher: `OutputRB.DrainTo` → `PersistResult` + `PublishJSONBatch`
3. 保留现有 `Engine` 和 `OrderBook` 实现
4. 保留 JSON 编码(进程内也用 JSON)，先拿架构收益

**预期收益**：
- 撮合延迟从 ~2-5ms 降至 ~50-200μs
- 吞吐量提升 3-5x (不再被 DB/Kafka 阻塞)

### Phase 2: Binary Encoding + Idle Strategy (1 week)

**目标**：消除热路径的 JSON 开销

1. 实现 `MatchCommand` / `MatchResult` 二进制结构
2. Input Gateway: JSON→Binary 转换
3. Output Dispatcher: Binary→JSON 转换(发往 Kafka)
4. 实现 `IdleStrategy` (spin/yield/park)

**预期收益**：
- 撮合延迟降至 ~10-50μs
- CPU 使用降低 ~30-40% (JSON 序列化开销消除)

### Phase 3: Per-Book Full Isolation (1-2 weeks)

**目标**：每个 book 完全独立的 goroutine 集合

1. 从 `AsyncEngine` (共享 worker map) 迁移到 `BookSupervisor` (独立 BookEngine)
2. 每个 BookEngine 有自己的 Input/Output Ring Buffer
3. Kafka partition 与 BookEngine 1:1 映射
4. 全局 trade sequence 保持 atomic counter

**预期收益**：
- 多 book 完全并行，无 mutex 竞争
- 单 book 吞吐量达到 50K+ orders/sec

### Phase 4: OrderBook 数据结构优化 (1-2 weeks)

**目标**：exchange-core2 级别的 OrderBook 实现

1. 将 `[]DepthLevel` + `sort.Search` 替换为 `[100]*Bucket` 直接数组索引（预测市场价格 1-99）
2. 将 `[]*Order` per-level 替换为侵入式双向链表（`DirectOrder.prev/next`）
3. 维护 `bestAsk`/`bestBid` 直接指针，O(1) 最优价格访问
4. 实现 `OrderPool`（slab + free list），热路径零 GC
5. 实现增量 depth diff
6. 全局 trade sequence 改为 `bookKey + localSeq` 确定性复合 ID

### Phase 5: Primary-Standby HA (2-3 weeks)

**目标**：接近零停摆的故障切换

1. 实现 shadow mode: Standby 消费相同输入但不输出
2. 实现 BookSnapshot gRPC 接口
3. 实现 fencing (epoch_id)
4. 实现 Standby 追赶协议(snapshot + replay)

---

## 9. Capacity Planning

### 9.1 单 BookEngine 资源

| 资源 | 值 | 说明 |
|------|-----|------|
| goroutines | 3 | Input + Match + Output |
| Input RB 内存 | ~8 MB | 64K × 128B MatchCommand |
| Output RB 内存 | ~16 MB | 64K × 256B MatchResult |
| OrderBook 内存 | ~10-50 MB | 取决于 resting order 数量 |
| CPU (活跃) | ~0.5 core | Matching Thread 占主要 |

### 9.2 集群规模(目标)

| 场景 | 活跃 Book 数 | 实例数 | 配置 |
|------|-------------|--------|------|
| 当前 staging | ~10 | 1 | 2 core / 2 GB |
| 生产 V2 早期 | ~100 | 2 | 4 core / 8 GB |
| 生产 V2 满载 | ~1000 | 4-8 | 8 core / 16 GB |

### 9.3 理论吞吐量上限

```
单 BookEngine:
  撮合纯计算: ~10-50ns per order (内存操作)
  Ring Buffer 读写: ~20ns per operation
  合计: ~50-100ns per order
  理论上限: 10M-20M orders/sec (CPU bound)

实际预期 (含 idle, batch, 调度):
  单 book: 50K-200K orders/sec
  集群 (100 books × 4 instances): 5M-20M orders/sec aggregate
```

---

## 10. Design Decisions (Finalized)

### D1. 反压策略：Pause Partition

**决策**：Ring Buffer 满时，Input Gateway **暂停 Kafka partition 消费**（stop polling），而非在 gateway 内部死循环重试。

**为什么不是 "gateway 里 sleep + retry"**：
- 死等重试会让 Input Gateway goroutine 占住 CPU 做无用功，且无法响应 shutdown 信号
- Kafka consumer 的 `max.poll.interval.ms` 会超时触发 rebalance，导致 partition 丢失

**实现方式**：
```go
// Input Gateway 核心循环
for {
    if inputRB.IsFull() {
        consumer.Pause(partition)          // 暂停拉取，Kafka 端不 commit
        idleStrategy.Idle(0)               // 短暂让出 CPU
        if !inputRB.IsFull() {
            consumer.Resume(partition)      // 水位下降后恢复
        }
        continue
    }
    msg := consumer.Poll()                 // 正常拉取
    cmd := decode(msg)
    inputRB.TryPublish(cmd)
}
```

**效果**：
- Kafka broker 自动反压 producer（API 层的 PublishJSON 会变慢）
- Ring Buffer 内存有严格上界（64K × 128B = 8MB）
- 语义最简单：不丢消息、不拒绝、不溢出到磁盘

---

### D2. DB 异步持久化失败恢复：Kafka Canonical Log + 幂等下游

**决策**：DB 写入失败时，不 panic，不重试到死。标记 offset 未 commit，服务重启后从 Kafka 重放。

**前提条件（必须满足）**：

| 条件 | 当前状态 | 需要补的 |
|------|---------|---------|
| Kafka 是 canonical log，消息不可变 | ✅ `order.command` 已是 source of truth | — |
| Trade identity 确定性生成 | ⚠️ 当前用 atomic counter | 改用 `bookKey + localSeq`（见 D4） |
| Order event ID 确定性 | ✅ `order_id + status` | — |
| Account 服务幂等 | ⚠️ 部分幂等（freeze idempotent key） | 需要补全 trade 消费幂等 |
| Ledger 服务幂等 | ✅ `event_id` 去重 | — |
| Settlement 服务幂等 | ✅ `event_id` 去重 | — |

**关键补充**：

```
replay 安全公式:
  ∀ downstream consumer:
    consume(event) 是幂等的
    ⟺ 相同 event_id 第二次消费 = no-op

replay 时撮合引擎:
  相同输入序列 → 相同 (trade_id, order_event) 输出
  ⟺ trade_id 由确定性函数生成，不依赖 wall clock 或 random
```

如果上述任何一条不满足，replay 会导致 account/ledger 被"打重"。这是最需要在 Phase 1 之前验证的不变式。

---

### D3. 跨 book 操作：V2 不支持，API 层拒绝

**决策**：不支持组合单（cross-book atomic order）。

**理由**：
- 单写者模型是撮合引擎正确性和性能的根基
- 跨 book 原子操作需要两阶段提交或全局序列化，会把吞吐量打回到 V1 甚至更差
- 预测市场场景下，用户买 YES 和 NO 是在同一 market 的不同 book，但不需要原子关联

**实现**：API 层在 `order.Service.SubmitOrder` 中校验，如果请求包含跨 book 语义（当前不存在这种 API），直接返回 400。

---

### D4. Trade Identity：放弃全局递增整数，改用确定性复合 ID

**决策**：不用 `instance prefix + atomic counter`，也不用中心化 sequence server。改为 **`bookKey + localSeq`** 复合 ID。

**为什么 instance prefix 不好**：
- 多实例 failover 后 prefix 变了，下游无法判断"这是 replay 还是新 trade"
- 用户看到的 trade ID 不连续、不可排序，体验差
- prefix 本身需要协调分配（又一个中心点）

**为什么中心化 sequence server 不好**：
- 网络往返 ~0.5-1ms，直接抵消 Ring Buffer 带来的 μs 级收益
- 成为新的单点故障和延迟瓶颈
- 违反 "撮合线程不做 I/O" 的核心原则

**V2 方案**：

```go
// Trade ID = bookKey + per-book monotonic sequence
// 格式: "BTC:YES:00000042"
type TradeIdentity struct {
    BookKey  string  // partition-scoped, 同一 partition 内唯一
    LocalSeq uint64  // per-book 单调递增，确定性（相同输入 → 相同序号）
}

func (id TradeIdentity) String() string {
    return fmt.Sprintf("%s:%08d", id.BookKey, id.LocalSeq)
}
```

**确定性保证**：
- `LocalSeq` 是 per-book 的 counter，只被该 book 的 matching goroutine 操作
- 相同的命令序列 → 相同的 `LocalSeq` → Standby replay 产生完全一致的 trade identity
- 下游幂等消费用 `TradeIdentity.String()` 作为 dedup key

**排序需求**：如果业务上需要"跨 book 全局时间排序"，用 `(timestamp, bookKey, localSeq)` 三元组排序，不需要全局递增整数。

---

### D5. OrderBook 数据结构：参考 exchange-core2 的 ART + 侵入式双向链表

**参考**：[exchange-core2 OrderBookDirectImpl](https://github.com/exchange-core/exchange-core/blob/master/src/main/java/exchange/core2/core/orderbook/OrderBookDirectImpl.java)

exchange-core2 的 Direct 实现是工业级 OrderBook 的标杆设计，核心思路：

```
askPriceBuckets: LongAdaptiveRadixTreeMap<Bucket>   ← 价格 → 桶
bidPriceBuckets: LongAdaptiveRadixTreeMap<Bucket>   ← 价格 → 桶
orderIdIndex:    LongAdaptiveRadixTreeMap<DirectOrder> ← orderId → 订单

Bucket {
    tail:     *DirectOrder      // 指向该价位最新的订单
    volume:   int64             // 该价位总剩余量
    numOrders: int              // 该价位订单数
}

DirectOrder {
    orderId, price, size, filled, uid, timestamp
    parent:  *Bucket            // 所属桶
    prev:    *DirectOrder       // 全局链表 prev（更差价格方向）
    next:    *DirectOrder       // 全局链表 next（更优价格方向）
}

bestAskOrder → 直接指针，O(1) 获取最优卖价
bestBidOrder → 直接指针，O(1) 获取最优买价

撮合时: 从 bestAskOrder/bestBidOrder 开始，沿 prev 链遍历
插入时: ART tree 找到或创建 Bucket，append 到 bucket.tail 链表末尾
删除时: 双向链表 O(1) 摘除 + ART tree 删除空桶
```

**关键设计亮点**：

| 特性 | exchange-core2 做法 | 收益 |
|------|-------------------|------|
| 价格索引 | Adaptive Radix Tree (ART) | O(k) lookup (k=key 字节长度，固定)，比红黑树 cache 更友好 |
| 同价位队列 | 侵入式双向链表 | O(1) 追加、O(1) 删除，无数组 shift |
| 全局遍历 | 跨桶的 prev/next 链 | 撮合遍历不需要"找下一个价位"，直接跟指针 |
| Best price | 直接指针 `bestAskOrder` / `bestBidOrder` | O(1)，不需要 tree.min() |
| 内存管理 | ObjectsPool (pre-allocated) | 热路径零 GC |

**Go 适配方案**：

```go
// 价格桶索引：Go 没有 ART 库，用 B-tree (google/btree) 或自实现 radix tree
// 对于预测市场（价格范围 1-99 cents），用 [100]*Bucket 直接数组索引更快 ← O(1)
type PriceBuckets struct {
    buckets [100]*Bucket  // 价格 1-99，index = price
    // 如果需要支持更大范围，fallback 到 btree
}

// 侵入式双向链表：直接移植
type DirectOrder struct {
    OrderID   [48]byte
    Price     int64
    Size      int64
    Filled    int64
    UserID    int64
    Timestamp int64
    Side      uint8

    parent *Bucket
    prev   *DirectOrder  // toward worse prices
    next   *DirectOrder  // toward better prices
}

type Bucket struct {
    Tail      *DirectOrder
    Volume    int64
    NumOrders int32
}

type OrderBook struct {
    askBuckets   PriceBuckets
    bidBuckets   PriceBuckets
    orderIndex   map[string]*DirectOrder  // orderId → order
    bestAsk      *DirectOrder
    bestBid      *DirectOrder
    pool         *OrderPool               // pre-allocated
}
```

**预测市场的特殊优化**：价格范围固定为 1-99 cents（二元市场），可以用 `[100]*Bucket` 数组做 O(1) 价格定位，比 ART/B-tree 还快。这是 exchange-core2 通用设计之上的场景优化。

**OrderPool（零 GC）**：
```go
type OrderPool struct {
    orders []DirectOrder  // pre-allocated slab
    free   []int          // free list indices
}

func (p *OrderPool) Get() *DirectOrder {
    if len(p.free) == 0 {
        // 扩容（非热路径，允许 alloc）
        p.grow()
    }
    idx := p.free[len(p.free)-1]
    p.free = p.free[:len(p.free)-1]
    return &p.orders[idx]
}

func (p *OrderPool) Put(order *DirectOrder) {
    *order = DirectOrder{} // zero out
    idx := int((uintptr(unsafe.Pointer(order)) - uintptr(unsafe.Pointer(&p.orders[0]))) / unsafe.Sizeof(DirectOrder{}))
    p.free = append(p.free, idx)
}
```

---

## 11. 与 Aeron / exchange-core2 设计的对比总结

| 参考设计 | 我们的适配 | 差异原因 |
|---------|-----------|---------|
| Aeron SPSC Ring Buffer (`sun.misc.Unsafe`) | Go `atomic.Uint64` + cache line padding | Go 无 Unsafe，atomic 已足够 |
| Aeron Media Driver (shared memory IPC) | 进程内 Ring Buffer (同一进程) | 不需要跨进程 IPC |
| Aeron Archive (persistent stream) | Kafka retention + topic replay | Kafka 是 canonical log |
| Aeron Cluster (Raft consensus) | Kafka consumer group + deterministic replay | 更简单，利用现有 Kafka 基础设施 |
| Aeron `IdleStrategy` | 直接移植 (spin/yield/park) | 1:1 移植 |
| Aeron `Flyweight` (zero-copy decode) | Fixed-size `MatchCommand` struct | 类似思路 |
| LMAX Disruptor (exchange-core2 依赖) | 我们的 SPSC Ring Buffer | Disruptor 是 MPMC，我们只需 SPSC |
| exchange-core2 `LongAdaptiveRadixTreeMap` | `[100]*Bucket` 直接数组 (预测市场优化) | 价格范围固定 1-99，O(1) > O(k) |
| exchange-core2 侵入式双向链表 | 直接移植到 Go | 核心思路完全适用 |
| exchange-core2 `ObjectsPool` | Go `OrderPool` (slab + free list) | 同样目标：热路径零 GC |
| exchange-core2 `bestAskOrder/bestBidOrder` | 直接指针，O(1) best price | 直接移植 |
| exchange-core2 全局 trade sequence | 放弃全局递增，改用 `bookKey:localSeq` | 多实例 + 确定性 replay 的要求 |

---

## Appendix A: 当前代码到 V2 的映射

| 当前文件 | V2 新模块 | 改动 |
|---------|----------|------|
| `service/consumer.go` (CommandProcessor) | `transport/input_gateway.go` | 拆分为纯 I/O + decode，反压用 pause partition |
| `engine/engine.go` (AsyncEngine) | `supervisor/book_supervisor.go` | 取代 worker map |
| `engine/engine.go` (Engine.PlaceOrder) | `engine/engine.go` (保留, 优化) | 内部结构优化 |
| `model/order_book.go` (OrderBook) | `model/order_book_direct.go` | exchange-core2 风格重写 |
| `model/depth_level.go` (DepthLevel) | `model/bucket.go` + `model/direct_order.go` | 侵入式链表 + 直接指针 |
| `service/sql_store.go` (PersistResult) | `transport/output_dispatcher.go` | 异步批量化 |
| (新增) | `ringbuffer/ringbuffer.go` | SPSC Ring Buffer (cache-line padded) |
| (新增) | `transport/idle_strategy.go` | Aeron Idle Strategy (spin/yield/park) |
| (新增) | `protocol/commands.go` | Binary MatchCommand/MatchResult |
| (新增) | `model/order_pool.go` | Slab allocator + free list, 零 GC |
| (新增) | `model/trade_identity.go` | `bookKey:localSeq` 确定性复合 ID |
| (新增) | `ha/standby.go` | Primary-Standby 确定性 replay |

---

## Appendix B: Benchmark Plan

在开发各 phase 之前和之后运行基准测试：

```go
// benchmark: 空 book 单次 PlaceOrder
func BenchmarkPlaceOrder_EmptyBook(b *testing.B) { ... }

// benchmark: 1000 resting orders, 新 order 撮合
func BenchmarkPlaceOrder_DeepBook(b *testing.B) { ... }

// benchmark: Ring Buffer SPSC throughput
func BenchmarkRingBuffer_SPSC(b *testing.B) { ... }

// benchmark: 端到端 InputRB → Match → OutputRB
func BenchmarkE2E_RingBufferPipeline(b *testing.B) { ... }

// benchmark: JSON vs Binary encoding
func BenchmarkEncoding_JSON_vs_Binary(b *testing.B) { ... }
```

**目标基线**：
| 指标 | V1 当前 | V2 Phase1 | V2 Phase3 |
|------|---------|-----------|-----------|
| PlaceOrder latency (p50) | ~2ms | ~100μs | ~20μs |
| PlaceOrder latency (p99) | ~10ms | ~500μs | ~100μs |
| Single book throughput | ~2K/s | ~10K/s | ~50K/s |
| E2E order→trade latency | ~5-10ms | ~1ms | ~200μs |
