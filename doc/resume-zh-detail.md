# 个人技能

## 后端开发

- **Golang**：熟练掌握语言特性与并发模型，具备 Goroutine / Channel 调度与同步设计经验；能够在高并发场景下进行锁竞争分析与优化（包括 sync 原语、细粒度锁、无锁队列、SPSC 环形缓冲区等方案），理解 Aeron 风格的 IdleStrategy（busy-spin → yield → sleep 渐进退避）与 cache-line padding 等低延迟编程范式
- **微服务与中间件**：熟悉 Gin、go-zero、gRPC、WebSocket 等服务形态与 RPC 拆分方式，具备多服务协作、接口版本管理、protobuf 契约演进及错误处理的实践经验
- **数据存储**：熟悉 MySQL、PostgreSQL、Redis，具备分库分表、索引与 SQL 调优经验；能够处理缓存穿透/击穿/雪崩问题，保证缓存与数据库一致性，并具备分布式锁的实际落地经验
- **消息与异步**：熟悉 Kafka、Redis Stream 等消息系统，具备异步解耦、有序消费、ConsumerGroup 多 partition 并行消费、LZ4 压缩批量发布、重试与死信处理、幂等设计与去重机制经验；能够在 at-least-once 语义下保证业务可靠性；了解 Aeron 的 IPC/UDP 传输与 Publication/Subscription 模型及其在金融撮合领域的应用思路
- **运维与可观测性**：熟悉 Linux 环境部署与线上问题排查（日志分析、系统资源监控、慢查询定位等），熟练使用 Docker、Git、Makefile，具备 Prometheus 等监控指标采集与告警体系的实践经验

## 区块链 / 交易基础设施

- **钱包与资金业务**：熟悉多链钱包核心流程，包括充值扫链（eth_getLogs / 逐块扫描）、提现异步队列、归集、热冷钱包划转、链上确认数与 reorg 检测、链上事件到账务/余额系统的映射机制；具备多租户钱包 SaaS 的架构设计与完整落地经验
- **签名与密钥管理**：具备签名服务与密钥托管经验，熟悉 HD 钱包体系（BIP32/BIP44）、助记词/私钥加密存储与内存清零；了解 HSM、MPC 钱包的基本原理及其接入边界
- **EVM 合约接入**：熟悉 Solidity 及 Anchor（Solana），能够结合协议文档与源码完成合约调用封装、事件解析（Transfer/Approval/自定义 event）、交易构造（Legacy/EIP-1559/EIP-4844）及链上状态与后端状态对齐
- **Solana 应用开发**：熟悉 Solana 生态，理解账户模型、交易结构与执行机制、并行执行与 CU（Compute Units）约束；熟悉 SPL Token / Token-2022 的常见交互模式
- **Solana DEX 与协议**：具备链上数据解析与交易构造经验，熟悉 Pump.fun、Raydium（AMM / CLMM / CPMM）等协议的核心机制及集成方式
- **Sol 链 MEV**：熟悉链上交易的 CU 估算、Priority Fee、滑点控制与路径优化；了解 Jito 等 MEV 相关机制及防夹（anti-MEV）策略与产品侧限制

---

# 项目经历

## Seersmarket 预测市场

**Go 后端组长** | 2025.12 – 至今

面向加密货币价格、体育赛事、热点事件等场景，构建 Web3 预测市场平台，覆盖市场创建、订单簿交易、实时行情、限价/市价下单、自动/人工裁决、资金结算、邀请分润与消息通知等完整业务闭环。

**技术栈**：Go、Gin、gRPC、Kafka、PostgreSQL、Redis、WebSocket、Solidity、gnark（Groth16/BN254）

### 高性能链下撮合引擎（借鉴 Aeron 设计理念）

- 主导撮合引擎架构设计，采用 **InputGateway → BookEngine → OutputDispatcher** 三级流水线，借鉴 Aeron single-writer 原则实现热路径零 I/O、接近零 GC；每 book 独立 SPSC 无锁环形缓冲区 + Aeron 风格渐进退避 IdleStrategy；**实测单 book 撮合 127 万 ops/s**
- 利用预测市场价格有界的特性，以**固定数组 O(1) 寻址 + bitmap 位运算跳跃 + 侵入式链表 FIFO + slab 对象池**实现价格-时间优先撮合，替代传统红黑树方案；pprof 发现 Snapshot 线性扫描占 83% CPU 后改为 bitmap 遍历，关键路径提速 **7–9x**
- Gateway 层由单 Reader 演进为 **ConsumerGroup per-partition 并行消费**，producer 按 bookKey Hash 分区天然保证 SPSC 不变式；Dispatcher 层采用 **multi-row INSERT + 4 并发 worker 分片保序 + LZ4 压缩批量发布**；通过 pprof 逐层打通后 **E2E 吞吐从 56 提升至 3,400 trades/s（61x），p50 延迟从 43.8s 降至 188ms**
- 实现完整订单类型：GTC / IOC / **FOK（read-only 预检 + 真实撮合两阶段）** / **POST_ONLY**；三种 **STP 自成交保护策略** + **Amend Order**
- 确定性 `TradeID`（全局 sequence + book 级 localSeq）保证 Kafka 重放可复现；备节点 Shadow 模式消费状态但不执行 I/O，支持无缝 HA 切换

### 核心业务实现

- 负责 20+ 核心微服务的规划、开发与治理，覆盖用户鉴权、订单交易、行情服务、裁决/预言机、结算清算、链侧提交、消息推送与分润任务等关键链路
- 设计并落地 Kafka 驱动的核心交易链路，以有序指令串联撮合、账户冻结/解冻、结算、账变与通知流程；热路径不绑多跳同步 RPC，matching 作为 book 唯一写入者，Kafka 分区键与 book key 对齐保证同一 book 全序
- 主导结算引擎设计与实现，抽象传统模式、庄家模式、庄家退出、流拍退款、庄家作恶处罚等复杂分支，沉淀可扩展的策略化结算模型
- 设计并实现复式记账账本：双层分离（Account 管可变余额/冻结、Ledger 维护追加型日志）；每笔 entry 强制借方=贷方；`processedRefs` + `TradeID` 幂等键防 Kafka 重复消费；`BuildReport` 比较负债快照与链上快照输出 `BALANCED` / `DRIFT`

### 钱包 SaaS 衔接与资金链路

- 负责多租户钱包 SaaS 后端接入，打通充值扫链（逐块扫描 + eth_getLogs ERC-20 Transfer event）、提现签名队列、到账确认、风控限额、异常回撤、账务映射等核心能力
- 实现 mirror-accounting 模型：SaaS 管链上托管，平台内部账本镜像 SaaS 的充值/提现事件，实现撮合、账本与资金视图的统一
- 设计多币种充值自动兑换方案：接受 BNB/ETH 等多种代币充值，通过 Binance 实时价格接口自动折合为 USDT 平台余额，兼顾用户体验与内部计价统一
- 实现 parentHash 持久化 + reorg 检测机制，扫描每个区块时校验 parentHash 连续性，发现分叉及时预警；充值到账通过 Kafka → WebSocket 实时推送前端

### Rollup 与链上验证

- 参与 ZK-Rollup 批次编排、链下影子回放、Groth16 证明提交流程与失败自动重试机制建设
- 三阶段提交（record → publish → accept）；超大批次自动切换 DA 旁路（运营商 hash 证明）
- 强制提现 → 冻结 → 逃生舱用户保护完整闭环

### 实时行情与推送

- 构建基于 Binance WebSocket 的实时价格服务，并设计 HTTP fallback 兜底机制
- 基于 Redis + WebSocket 实现价格、订单、成交、市场状态、结算结果、充值到账等多频道高频推送，单连接复用多 topic

---

## 多链聚合钱包 SaaS 系统

**Go 后端开发** | 2023.05 – 2025.10

面向集团内多个 Web3 项目（如一元夺宝、Seers、哈希游戏等）提供统一的钱包基础设施能力，支持多租户隔离、多链地址管理、充值入账、提现签名、自动归集与账务映射等核心链路。

**技术栈**：Go、PostgreSQL、Redis、gRPC、Docker、HD Wallet (BIP32/BIP44)、PKCS#11

### 核心架构

- 负责五大微服务核心架构设计与开发：`api-gateway`（路由 + 鉴权 + 幂等）、`wallet-core`（提现编排 + 地址管理）、`chain-gateway`（多链 RPC 适配 + 交易构造）、`sign-service`（签名隔离 + HSM/本地双模式）、`scan-service`（充值扫链 + outbox 派发 + reorg 协调）
- 主导签名服务重构，搭建 gRPC signer + custody provider + backend provider 分层架构，将密钥派生、签名执行与业务系统解耦；实现空 token 拒绝与 rate-limit map 自动清理
- 手写 BIP32 层级确定性密钥派生（secp256k1 硬化/非硬化子密钥 + ed25519 硬化派生），不依赖第三方 HD 库；master seed 采用 scrypt KDF + AES-256-GCM 加密后存储于 LevelDB，nonce/salt 随机生成
- 设计 `hsm.Backend` 接口抽象密钥存储层：当前 SoftwareBackend（LevelDB 加密存储）为主，预留 CloudHSMBackend（PKCS#11）接口便于生产切换；每个租户独立 seed slot，slot ID 经 SHA-256 hash 防 injection
- 签名流程中私钥仅在签名瞬间从 seed 派生，签名完成后 `defer zeroString` 立即清零内存，最小化私钥暴露窗口
- 实现多租户 HD 钱包模型，统一 `key_id` 与派生路径语义，支持 ECDSA(secp256k1) 与 EdDSA(ed25519) 两类签名算法，覆盖 Ethereum、BSC、Polygon、Arbitrum、Solana、Tron 等多链网络

### 提现与归集

- 设计提现异步队列与 EVM stuck tx 自动加速机制，支持 QUEUED → PROCESSING → BROADCASTED → CONFIRMED 状态流转及 nonce replacement / speed-up 处理
- 实现冷热钱包两层归集模型，支持按热钱包余额阈值动态路由至 hot/cold 账户，提升资金归集效率与风险隔离能力

### 扫链与可靠性

- 实现 chain-gateway EVM 适配器中基于 `eth_getLogs` 的 ERC-20 Transfer event 批量扫描，支持按合约地址或收款地址 topic 过滤，替代逐笔 receipt 轮询，大幅提升扫链效率
- 实现 parentHash 持久化与 reorg 检测，扫描区块时记录 blockHash 链并校验 parentHash 连续性，旧记录自动修剪
- RPC 调用层添加指数退避重试（3 次 / 300ms base / 4s max），过滤 429/502/503/504/timeout 等可恢复错误
- outbox 派发添加 `FOR UPDATE SKIP LOCKED` 行级锁防多实例并发；幂等层改为 `INSERT ... ON CONFLICT DO UPDATE ... RETURNING` 原子操作消除双写竞态
- 项目通知与归集触发的下游失败降级为 non-fatal，不阻塞主 outbox 处理流

### 基础设施

- 统一集中化 Schema 管理：抽取 5 个服务的 inline DDL 至版本化 SQL 迁移文件 + 共享迁移运行器（schema_migrations 表追踪）
- 金额字段从 TEXT 升级为 NUMERIC(78,0)，支持 uint256 全精度范围内的数据库端算术
- Makefile 集成 build / vet / test / vuln（govulncheck）/ proto 等目标
- Docker Compose 生产配置外部化所有凭据，HTTP Server 显式超时，gRPC 可选 TLS，EVM gas 参数环境变量化
