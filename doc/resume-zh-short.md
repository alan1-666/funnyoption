# 个人技能

## 后端开发

- **Golang**：熟练掌握并发模型（Goroutine / Channel / sync 原语），具备高并发场景下的锁竞争分析与优化经验，熟悉 SPSC 无锁队列、Aeron 风格 IdleStrategy、cache-line padding 等低延迟编程范式
- **微服务与中间件**：熟悉 Gin、go-zero、gRPC、WebSocket，具备多服务协作、protobuf 契约演进及接口版本管理经验
- **数据存储**：熟悉 MySQL、PostgreSQL、Redis，具备索引调优、缓存一致性保障与分布式锁落地经验
- **消息与异步**：熟悉 Kafka 有序消费、ConsumerGroup 多 partition 并行消费、LZ4 压缩批量发布、幂等设计、重试与死信处理；了解 Aeron IPC/UDP 传输模型及其在金融撮合领域的应用
- **运维与可观测性**：熟悉 Docker、Linux 线上排查、Prometheus 监控告警

## 区块链 / 交易基础设施

- **钱包与资金业务**：熟悉多链钱包核心流程（充值扫链 / 提现 / 归集 / 热冷划转 / reorg 检测），具备多租户钱包 SaaS 架构设计与落地经验
- **签名与密钥管理**：熟悉 HD 钱包（BIP32/BIP44）、签名服务分层架构；了解 HSM / MPC 钱包原理
- **EVM 合约接入**：熟悉 Solidity，能完成合约调用封装、事件解析、EIP-1559 交易构造及链上链下状态对齐
- **Solana 生态**：熟悉账户模型、SPL Token、Raydium / Pump.fun 等 DEX 协议集成；了解 CU 估算、Jito MEV 及 anti-MEV 策略

---

# 项目经历

## Seersmarket 预测市场 — Go 后端组长 | 2025.12 – 至今

Web3 预测市场平台，覆盖订单簿交易、实时行情、裁决、结算、资金托管等完整闭环。
**技术栈**：Go、Gin、gRPC、Kafka、PostgreSQL、Redis、WebSocket、Solidity

**【撮合引擎 — 借鉴 Aeron 设计理念】**
- 主导撮合引擎架构设计，三级流水线 + SPSC 无锁环形缓冲区 + bitmap 位运算价位跳跃，热路径零 I/O、接近零 GC；**实测单 book 撮合 127 万 ops/s**
- 支持 GTC / IOC / FOK（两阶段预检）/ POST_ONLY + 三种 STP 自成交保护 + Amend Order；确定性 TradeID 支持 Kafka 重放复现；Shadow 模式 HA 切换
- 主导 E2E 全链路性能调优：通过 pprof 逐层定位瓶颈，从引擎层 bitmap 优化、DB 批量落盘、Kafka 异步提交与 LZ4 压缩到多 partition 并行消费逐级打通；**最终吞吐 61 倍提升（56 → 3,400 trades/s），p50 延迟从 43.8s 降至 188ms**

**【核心业务】**
- 负责 20+ 微服务规划开发，设计 Kafka 驱动的交易链路（撮合 → 冻结 → 结算 → 账变），matching 作为 book 唯一写入者，分区键对齐保证全序
- 主导结算引擎，抽象传统/庄家/流拍/处罚等多模式策略化结算；设计复式记账账本（借方=贷方强校验 + TradeID 幂等 + 负债对账报告）
- 负责钱包 SaaS 接入，实现 mirror-accounting 模型统一撮合与资金视图；设计多币种充值自动兑换方案（Binance 实时价格 → USDT 折合）
- 参与 ZK-Rollup 批次编排（record → publish → accept 三阶段提交 + Groth16 证明 + 失败自动重试）
- 构建 Binance WebSocket 实时价格服务 + Redis/WS 多频道高频推送

---

## 多链聚合钱包 SaaS 系统 — Go 后端开发 | 2023.05 – 2025.10

面向集团内多个 Web3 项目提供统一钱包基础设施，支持多租户隔离、多链地址管理、充值提现归集。
**技术栈**：Go、PostgreSQL、gRPC、Docker、HD Wallet (BIP32/BIP44)

- 负责五大微服务架构设计（api-gateway / wallet-core / chain-gateway / sign-service / scan-service），主导签名服务重构为 gRPC signer + custody provider 分层架构，密钥派生与业务解耦
- 实现多租户 HD 钱包模型，支持 ECDSA / EdDSA 双算法，覆盖 Ethereum、BSC、Polygon、Arbitrum、Solana、Tron 等多链；设计提现异步队列与 stuck tx 自动加速
- 实现 eth_getLogs 批量 ERC-20 扫链、parentHash reorg 检测、RPC 指数退避重试、outbox 行级锁防并发重处理、幂等原子化等可靠性增强
- 实现冷热钱包两层归集模型，按热钱包余额阈值动态路由
