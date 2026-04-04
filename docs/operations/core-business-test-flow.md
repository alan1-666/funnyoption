# FunnyOption 核心业务测试流程

这份文档用于在**换新会话**或**新同事接手测试**时，快速按同一套流程把 FunnyOption 的核心业务链路重新跑一遍。

它不替代所有底层文档，而是把“怎么测、按什么顺序测、每一步成功时该看到什么、失败时先查哪里”收敛到一个入口。

## 0. 当前业务边界

当前 FunnyOption 测试环境已经具备这些核心能力：

- 运营后台钱包登录
- 运营后台创建市场
- 运营后台发首发流动性
- 用户端钱包连接
- 用户端 session 授权
- 用户充值 USDT 类 ERC20 到 Vault
- chain-service 监听充值事件并完成链下入账
- 用户下单
- 撮合成交
- 运营后台结算市场
- settlement / account / ledger 更新终态数据
- 用户端查看仓位、挂单、历史结算

当前仍需记住的产品边界：

- 交易引擎当前主链路仍是**二元 YES / NO 市场**
- 多选项市场的 `options` 已经可以存储，但**非二元市场不应直接以 `OPEN` 状态进入交易**
- 当前测试环境充值资产以 **MockUSDT / USDT 类 ERC20** 为准，**tBNB 只用于 gas，不是交易本金**
- 站内信、用户自建市场等入口目前是 UI 占位，不作为核心验收链路

## 1. 先选测试环境

### 1.1 本地环境

适合：

- 开发时快速验证改动
- 用本地 Anvil 链跑完整充值 / 结算链路
- 直接看本地数据库和服务日志

核心入口：

- 用户端：`http://127.0.0.1:3000`
- 运营后台：`http://127.0.0.1:3001`
- API 健康检查：`http://127.0.0.1:8080/healthz`

启动方式：

```bash
/Users/zhangza/code/funnyoption/scripts/dev-up.sh
```

如果你已经启用了 `FUNNYOPTION_LOCAL_CHAIN_MODE=anvil`，脚本会自动拉起本地 Anvil、部署 `MockUSDT` 和 `FunnyVault`，并生成三把测试钱包。

本地链详细说明见：

- [/Users/zhangza/code/funnyoption/docs/operations/local-persistent-chain.md](/Users/zhangza/code/funnyoption/docs/operations/local-persistent-chain.md)

### 1.2 测试服务器环境

适合：

- 用 BSC Testnet 跑“接近线上部署形态”的完整验证
- 给运营钱包和普通测试钱包做跨端联调

当前 staging 入口：

- 用户端：[https://funnyoption.xyz](https://funnyoption.xyz)
- 运营后台：[https://admin.funnyoption.xyz](https://admin.funnyoption.xyz)

当前链配置：

- 网络：`BSC Testnet`
- Chain ID：`97`
- 浏览器：[https://testnet.bscscan.com](https://testnet.bscscan.com)
- MockUSDT：`0x0ADa04558decC14671D565562Aeb8D1096F71dDc`
- FunnyVault：`0x7665d943c62268d27ffcbed29c6a8281f7364534`
- 运营钱包白名单地址：`0xC421d5Ff322e4213A913ec257d6b4458af4255c6`

部署和 nginx 说明见：

- [/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md](/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md)

## 2. 测试身份准备

至少准备两类钱包。

### 2.1 运营钱包

用途：

- 登录 `admin`
- 创建市场
- 发首发流动性
- 结算市场

要求：

- 钱包地址必须在 `FUNNYOPTION_OPERATOR_WALLETS` 白名单里
- 在 BSC Testnet 上要有一点 `tBNB` 付 gas

### 2.2 普通用户钱包

用途：

- 登录用户端
- 创建 session
- approve + deposit
- 下单
- 查看仓位 / 挂单 / 历史结算

要求：

- 钱包里要有一点 `tBNB` 付 gas
- 钱包里要有可充值的 `MockUSDT` / USDT 类 ERC20

### 2.3 测试资产规则

当前金额精度规则是：

- 链上 ERC20 使用 `6` 位小数
- 链下账户、冻结、结算和前端展示统一使用 `2` 位小数
- `price` 仍按“分”为整数处理，比如 `58` 表示 `58¢`
- `quantity` 仍按“份额”为整数处理

也就是说：

- 链上 `100 USDT` 对应原始 token 数值 `100000000`
- 链下 `100 USDT` 对应后端记账值 `10000`
- 页面显示为 `100.00 USDT`
- 二元市场中，`1` 份赢家仓位结算应得到 `1.00 USDT`，也就是 `100` 个链下记账单位

金额规则的架构说明见：

- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)

## 3. 端到端主链路测试

这部分是最重要的“核心业务测试主流程”。建议严格按顺序跑。

## 3.1 检查服务和页面是否活着

### 操作

本地环境：

```bash
curl http://127.0.0.1:8080/healthz
```

测试环境：

```bash
curl https://funnyoption.xyz/healthz
```

然后分别打开：

- 用户端首页
- 用户端任一市场详情页
- 用户端个人页
- 运营后台首页

### 预期结果

- `healthz` 返回 `status=ok`
- 首页能看到市场列表
- 市场详情页能看到 YES / NO 价格和下单面板
- 个人页能看到余额卡和三个 tab：`仓位`、`开的订单`、`历史结算`
- admin 页能看到运营钱包登录状态、创建市场表单、首发流动性表单和市场/成交读面

### 失败时先查

- 本地：`/Users/zhangza/code/funnyoption/.logs/dev/`
- staging：服务器 nginx 和容器日志
- 如果 HTTPS 页面白屏，优先查浏览器 Console 有没有 hydration error 或 ws / api mixed-content 问题

## 3.2 运营后台创建市场

### 操作

1. 打开 admin
2. 用**运营钱包**连接
3. 在“创建市场”里填写：
   - 标题
   - 分类：`加密` 或 `体育`
   - 状态
   - 开始 / 关闭 / 结算时间
   - 抵押资产
   - 封面图 URL
   - 来源 URL / 来源名称
   - 选项
4. 如果只是普通二元市场，使用 `YES / NO`
5. 如果选了多选项模板，市场状态先保持 `DRAFT`
6. 提交创建

### 预期结果

- admin 返回创建成功
- 新市场出现在后台市场列表
- 新市场出现在用户端首页和对应详情页
- 分类和选项能被正确读出来
- 如果传了封面图，首页卡片和详情页能看到封面

### 失败时先查

- 运营钱包是否在 allowlist 里
- admin 当前钱包网络是否是目标链
- 市场状态和选项是否违反“当前 OPEN 只允许二元 YES / NO”的规则
- `POST /api/v1/markets` 是否被 API 侧 operator 校验拒绝

## 3.3 运营后台发首发流动性

### 背景

新市场刚创建后，如果没有对手盘，普通用户下第一笔单可能只会挂单，不一定立刻成交。

当前做法是由 admin 先给 maker 用户发一份显式首发库存，并挂出第一笔 bootstrap 卖单。

### 操作

1. 在 admin 的“首发流动性”区域选择刚创建的市场
2. 填写：
   - maker user id
   - outcome
   - price
   - quantity
3. 提交

推荐用当前已准备好的 maker 用户：

- `user_id = 1002`

### 预期结果

- 返回 `first_liquidity_id`
- 返回一笔 bootstrap `order_id`
- 订单状态进入 `QUEUED` 或 `NEW`
- 用户端市场详情页能看到对应 YES / NO 价格变化
- 后台成交/订单读面能看到这笔首发挂单

### 失败时先查

- maker 用户是否有足够可用 USDT 余额
- maker 用户是否已有 active session
- 这笔 bootstrap order 是否被 replay / semantic uniqueness 策略拦截
- 如果返回 `409 insufficient available balance`，先给 maker 用户充值

## 3.4 用户端钱包连接和 session 授权

### 操作

1. 打开用户端首页
2. 点击右上角 `Connect`
3. 选择普通用户钱包
4. 如果钱包不在目标链，先切到：
   - 本地：`Anvil Local / 31337`
   - staging：`BSC Testnet / 97`
5. 完成钱包连接
6. 首次交易前完成 session 授权签名

### 预期结果

- 右上角从 `Connect` 变成余额 / 头像入口
- 点余额能进入个人页
- 下单面板不再停在“未授权”状态
- 不应该出现“刷新页面就反复弹钱包授权”的体验

### 失败时先查

- 浏览器里是否有 MetaMask 或兼容钱包插件
- 钱包当前网络是否正确
- 浏览器 local session 是否来自旧 chain id
- `POST /api/v1/sessions` 是否验签失败

## 3.5 用户充值并确认链下入账

### 操作

1. 打开个人页或资产页入口
2. 使用普通用户钱包
3. 输入充值金额，比如 `20.00`
4. 先点 `Approve`
5. 等钱包交易确认
6. 再点 `Deposit`
7. 等待 chain-service 监听事件并入账

### 预期结果

- 钱包交易发生在目标链上
- `Deposit` 不再因为 `InsufficientAllowance()` 回滚
- 用户端余额增加，比如 `20.00 USDT`
- 充值列表能看到新记录
- 后端 `chain_deposits` 有对应 deposit 行
- `account_balances.available` 增加

### 失败时先查

- 是否只充值了原生币 `tBNB`，而不是 MockUSDT / USDT 类 ERC20
- 是否忘了先 `approve`
- `approve` 授权额度是否低于本次 `deposit` 金额
- chain-service 是否连着正确的 vault 地址
- 链上 token decimals 和链下 accounting decimals 是否被错误理解

## 3.6 用户下单并撮合

### 操作

1. 打开一个 `OPEN` 市场详情页
2. 确认该市场已有首发对手盘
3. 在右侧下单面板选择 `是` 或 `否`
4. 填写价格和份额
5. 用 session key 提交订单

推荐先测一笔容易成交的单，比如：

- `BUY YES @ 58 x 5`

如果当前簿上已有 `SELL YES @ 58`，这笔单应当直接撮合。

### 预期结果

- 用户下单返回成功
- `orders` 里买单进入 `FILLED` 或 `PARTIALLY_FILLED`
- bootstrap 卖单被部分吃掉后进入 `PARTIALLY_FILLED`，或全部吃掉后进入 `FILLED`
- `trades` 里新增一条成交
- 市场详情页 YES / NO 概率、成交额、成交笔数发生更新
- 个人页“仓位”出现赢家/持仓记录
- 个人页“开的订单”能看到仍未完全成交的挂单

### 失败时先查

- 市场是否已经 `RESOLVED` 或非 `OPEN`
- 用户 USDT 可用余额是否不足
- 用户 session 是否过期或 nonce / replay 校验失败
- 是否误走了已经删除的 bare `user_id` 下单旧口子
- matching / kafka / ws 是否正常消费和广播

## 3.7 运营后台结算市场

### 操作

1. 打开 admin
2. 选择要结算的市场
3. 选择最终结果，比如 `YES`
4. 提交结算

### 预期结果

- 市场状态变为 `RESOLVED`
- `resolved_outcome` 正确写入
- 未成交的剩余挂单被终止，不能再继续成交
- 已有持仓被 settlement 扫描并结算
- 赢家用户生成 payout 记录
- 赢家余额增加
- 输家对应持仓进入已结算终态
- 用户端市场详情页和个人页读面同步更新

### 关键金额验收

二元市场当前结算规则是：

- `1` 份赢家仓位 = `1.00 USDT`
- 后端内部记账为 `100`
- 所以 `40` 份赢家仓位应得到 `4000`

如果你看到 payout 只等于 `40`，那就是历史上已经修过的错误结算公式回归了，需要立刻查 `internal/settlement/service/processor.go`。

### 失败时先查

- 运营钱包是否有 operator 权限
- 结算请求是否被 operator proof 校验拒绝
- settlement-service 是否正常消费 `market.event`
- account/ledger 是否写终态失败
- 如果 RESOLVED 后还能继续下单，优先查 API market status gate 和 matching 冷启动恢复过滤

## 3.8 用户端查看仓位、开的订单、历史结算

### 操作

1. 打开用户端个人页
2. 查看顶部余额卡
3. 在下方三个 tab 间切换：
   - `仓位`
   - `开的订单`
   - `历史结算`
4. 使用顶部搜索框过滤市场标题
5. 点钱包地址旁的复制按钮，确认可以复制地址
6. 点二维码按钮，确认可以弹出收款/地址二维码

### 预期结果

- 顶部余额和地址展示正确
- 地址能复制
- 二维码弹层能打开和关闭
- `仓位` tab 能看到当前持仓
- `开的订单` tab 能看到未完成订单
- `历史结算` tab 能看到已结算 payout / settlement 记录
- 页面不再展示重复的 `链上用户 / 0x...` 和多余说明文案

### 失败时先查

- profile API 是否正常返回默认头像/头像配置
- 个人页是否把内部 asset key 例如 `POSITION:...` 直接暴露出来
- 历史结算为空时是否是当前钱包确实没有 payout，而不是接口异常被误渲染成空态

## 4. 一键自动化测试入口

如果只是想快速跑一遍“本地完整生命周期”，优先用这个脚本：

```bash
/Users/zhangza/code/funnyoption/scripts/local-lifecycle.sh
```

它会自动覆盖这些步骤：

- 创建市场
- 创建两个 session
- 充值并等待 listener 入账
- 发首发流动性
- 下单撮合
- 结算市场
- 等待 payout / 仓位 / 市场终态更新
- 输出 JSON summary

更详细的自动化说明见：

- [/Users/zhangza/code/funnyoption/docs/operations/local-lifecycle-runbook.md](/Users/zhangza/code/funnyoption/docs/operations/local-lifecycle-runbook.md)
- [/Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md](/Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md)

## 5. Postman / API 手动验证入口

如果要绕开 UI，直接测 API，可以导入：

- [/Users/zhangza/code/funnyoption/docs/postman/funnyoption-api.postman_collection.json](/Users/zhangza/code/funnyoption/docs/postman/funnyoption-api.postman_collection.json)

建议 API 验证顺序：

1. `GET /healthz`
2. `GET /api/v1/markets`
3. `POST /api/v1/sessions`
4. `POST /api/v1/orders`
5. `GET /api/v1/orders`
6. `GET /api/v1/trades`
7. `GET /api/v1/balances`
8. `GET /api/v1/positions`
9. admin/operator 创建市场
10. admin/operator 首发流动性
11. admin/operator resolve market
12. `GET /api/v1/payouts`

注意：

- 普通用户下单必须走 session-backed order lane
- 运营侧市场创建、结算、首发流动性必须带 operator proof
- 不要重新依赖 bare `user_id` 写订单旧口子

## 6. 浏览器冒烟建议

如果要用 Playwright / 浏览器快速巡检页面，建议至少打开这几个页面：

- 首页：`/`
- 任意市场详情页：`/markets/{market_id}`
- 个人页：`/portfolio`
- 运营后台：`/`

页面冒烟重点看：

- 控制台有没有新 JS runtime error
- 是否有 React hydration mismatch
- 是否有 `favicon.ico 404`
- HTTPS 下有没有 ws / api mixed-content
- 首页、详情页、个人页、后台页面是否都能首屏正常渲染
- 搜索框、tab 切换、二维码弹层、钱包连接按钮这些基础交互是否可用

如果 Playwright 环境没有 MetaMask 插件，那么“真实钱包弹窗签名”这一步需要在你本机真实浏览器里手动验。

## 7. 常见问题速查

### 7.1 打开页面后钱包网络不对

先看 MetaMask 当前链是不是：

- 本地：`31337`
- staging：`97`

如果不是，优先让前端/后台触发切链，或者手动在钱包里切到目标链。

### 7.2 充值点了 Deposit，但余额没涨

先查四件事：

1. 这次交易是不是发到了当前目标链
2. 是否已经对 Vault 做过足够额度的 `approve`
3. 是否真的充值了 MockUSDT / USDT 类 ERC20，而不是 tBNB
4. chain-service 是否监听的是当前 vault 地址

如果链上交易 revert 且错误是 `InsufficientAllowance()`，基本就是授权额度不够，先重新 `approve`。

### 7.3 页面金额显示成 100000000 USDT 这种大数字

这通常是把链上 `6` 位原始 token 单位直接当人类金额显示了。

当前正确规则应该是：

- 链上 token amount：`6` 位小数
- 链下 accounting amount：`2` 位小数
- 页面显示：固定 `2` 位小数

新代码已经按这个规则处理；如果老数据本来就是旧错账，旧记录不会自动变干净。

### 7.4 首发流动性返回余额不足

如果看到：

```text
409 insufficient available balance
```

先检查 maker 用户有没有 USDT 可用余额。

当前 staging 已经准备过一个 maker 用户：

- `user_id = 1002`

如果这条账户余额又被用完，需要先给它重新充值。

### 7.5 市场 RESOLVED 后还能继续下单

这是严重回归，优先查：

- `internal/api/handler/order_handler.go` 里的 market status gate
- `internal/matching/service/sql_store.go` 的冷启动挂单恢复是否只加载 `OPEN` 市场
- `internal/settlement/service/processor.go` 是否在结算时终止剩余 active orders

### 7.6 结算后赢家 payout 金额不对

当前二元市场应该是：

- `quantity = 10`
- 赢家 `payout_amount = 1000`
- 页面显示 `10.00 USDT`

如果 payout 直接等于 `10`，说明结算公式又把 `settled_quantity` 当成金额了。

优先查：

- `internal/settlement/service/processor.go`
- `internal/shared/assets/assets.go`

### 7.7 后台页面能打开，但别的钱包也能进

要区分两件事：

- 页面能打开：这是正常的，任何人都能访问 URL
- 能执行敏感操作：只能是 `FUNNYOPTION_OPERATOR_WALLETS` 白名单里的钱包

验收时重点看“非白名单钱包是否真的无法创建市场 / 结算市场 / 发首发流动性”。

## 8. 下个会话接手时建议先读什么

如果下个会话只想继续做业务测试/联调，建议按这个顺序读：

1. [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
2. [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
3. 本文档
4. [/Users/zhangza/code/funnyoption/docs/operations/local-persistent-chain.md](/Users/zhangza/code/funnyoption/docs/operations/local-persistent-chain.md)
5. [/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md](/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md)
6. 如果要看签名/充值/结算规则，再读：
   - [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
   - [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
   - [/Users/zhangza/code/funnyoption/docs/architecture/market-taxonomy-and-options.md](/Users/zhangza/code/funnyoption/docs/architecture/market-taxonomy-and-options.md)

## 9. 交接时最重要的当前认知

目前最值得保留给下个会话的结论是：

- 主业务链路已经能跑通：建市场、发首发流动性、用户充值、下单、撮合、结算、读面更新
- admin 已经是独立服务，不再把运营能力挂在用户端页面里
- 用户端和后台都已经接入钱包身份和链配置
- staging 已经连到 BSC Testnet，并部署了测试版 `MockUSDT` 和 `FunnyVault`
- 当前最容易继续扩展的方向是：
  - 真正的多选项市场交易链路
  - 用户自建市场
  - 站内信
  - 真实测试网 USDT / 自动换币充值
  - 更完整的 Playwright + 钱包插件 E2E

换会话后，如果只想先验业务，不想重新读一大堆历史聊天，**就从本文档按 3.x 主流程直接跑起**。
