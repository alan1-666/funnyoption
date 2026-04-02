# 本地持久链模式说明

## 目标

这份文档说明如何把 FunnyOption 的本地开发环境切到一条**持久存在的本地 EVM 链**上运行，而不是继续使用旧的“进程内临时 proof chain”。

开启后，本地环境会获得这些能力：

- 自动启动一条本地 `anvil` 链，默认地址是 `127.0.0.1:8545`
- 自动部署本地测试版 `MockUSDT`
- 自动部署本地测试版 `FunnyVault`
- 自动生成一份链配置覆盖文件，供 `dev-up.sh`、前端和 `chain-service` 使用
- 自动生成三把测试钱包：
  - `operator`：平台运营钱包
  - `buyer`：买方测试用户，对应本地用户 `1001`
  - `maker`：卖方测试用户，对应本地用户 `1002`

这套模式的核心价值是：

- 前端可以真正连到一条稳定存在的本地链
- `chain-service` 会持续监听这条链上的 `FunnyVault` 事件
- 充值步骤不再是“伪造入账”，而是：
  - 钱包发起 `approve`
  - 钱包发起 `deposit`
  - `FunnyVault` 发出 `Deposited(address,uint256)` 事件
  - `chain-service` 监听到事件
  - `account-service` 完成链下余额入账

## 金额精度规则

本地持久链模式下，金额统一按下面这套规则理解：

- 链上 `MockUSDT` / `FunnyVault`
  - 使用 ERC-20 原生精度
  - `USDT = 6` 位小数
  - 例如链上 `100 USDT` 会表现为原始数值 `100000000`
- 链下账户、冻结、结算、前端展示
  - 统一按 `2` 位小数记账和显示
  - 例如 `100 USDT` 在后端内部记为 `10000`
  - 页面展示为 `100.00 USDT`

也就是说：

- 钱包和合约世界看的是 `6` 位 token 精度
- FunnyOption 的撮合、账户、结算和页面世界看的是 `2` 位金额精度

当前前端充值和提现输入也应该只输入到小数点后 `2` 位，例如：

- `100`
- `100.25`
- `0.50`

不建议输入超过 `2` 位小数的金额，因为链下账务不会保留比这更细的精度。

## 适用场景

建议在下面这些场景使用本模式：

1. 你要做“接近真实产品路径”的本地联调
2. 你要让 MetaMask 真正切到本地链
3. 你要完整验证：
   - admin 创建市场
   - 用户钱包登录
   - 用户充值
   - 余额入账
   - 下单撮合
   - 结算更新
4. 你要重复多次调试同一条本地链，而不是每次都重新起临时链

## 和旧模式的区别

旧模式的特点：

- `cmd/local-lifecycle` 会自己起一条进程内模拟链
- 链只为当前命令存在
- 前端和本地 `chain-service` 不直接连这条链

本模式的特点：

- `scripts/dev-up.sh` 会先起一条持久 `anvil`
- 合约部署在这条持久链上
- `chain-service` 直接监听这条链
- 前端也能切到同一条链
- 你可以手动在浏览器里操作，也可以跑自动 lifecycle 脚本

## 一次性前置准备

### 第一步：确认本机依赖

你本机至少要有这些命令：

- `anvil`
- `forge`
- `cast`
- `lsof`

可以分别执行：

```bash
anvil --version
forge --version
cast --version
lsof -v
```

如果前 3 个命令缺失，说明 Foundry 还没装好。

### 第二步：准备本地 env 文件

编辑你的本地环境文件：

- [`.env.local`](/Users/zhangza/code/funnyoption/.env.local)

至少加上这一行：

```bash
FUNNYOPTION_LOCAL_CHAIN_MODE=anvil
```

如果你不写这一行，`dev-up.sh` 就不会进入持久本地链模式，而是继续走原来的外部链配置逻辑。

### 第三步：理解本模式会覆盖哪些链配置

本模式开启后，`scripts/local-chain-up.sh` 会自动生成：

- [`.run/dev/local-chain.env`](/Users/zhangza/code/funnyoption/.run/dev/local-chain.env)

里面会覆盖成类似这些值：

- `FUNNYOPTION_CHAIN_RPC_URL=http://127.0.0.1:8545`
- `FUNNYOPTION_CHAIN_ID=31337`
- `FUNNYOPTION_CHAIN_NAME=anvil`
- `FUNNYOPTION_NETWORK_NAME=local`
- `FUNNYOPTION_CHAIN_CONFIRMATIONS=0`
- `FUNNYOPTION_VAULT_ADDRESS=<自动生成>`
- `FUNNYOPTION_COLLATERAL_TOKEN_ADDRESS=<自动生成>`
- `FUNNYOPTION_OPERATOR_WALLETS=<operator 地址>`

也就是说，你不需要手动填本地部署后的 vault 地址和 token 地址，脚本会帮你生成。

## 启动整套本地环境

### 第一步：运行一键启动脚本

执行：

```bash
/Users/zhangza/code/funnyoption/scripts/dev-up.sh
```

这是整个流程最关键的入口。

当它检测到 `FUNNYOPTION_LOCAL_CHAIN_MODE=anvil` 时，会先做本地链准备，再启动业务服务。

### 第二步：脚本内部会发生什么

按真实实现来看，[local-chain-up.sh](/Users/zhangza/code/funnyoption/scripts/local-chain-up.sh) 会依次做这些事：

1. 读取 [`.env.local`](/Users/zhangza/code/funnyoption/.env.local)
2. 检查本地链相关命令是否存在：
   - `anvil`
   - `forge`
   - `cast`
   - `lsof`
3. 检查本地 `8545` 端口是否已被别的进程占用
4. 如果之前已经有它自己管理的 `anvil` 在运行，就复用
5. 如果没有，就启动一个新的 `anvil`
6. 用 Foundry 编译合约
7. 部署：
   - [MockUSDT.sol](/Users/zhangza/code/funnyoption/contracts/src/MockUSDT.sol)
   - [FunnyVault.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyVault.sol)
8. 给买方和 maker 钱包打本地 ETH，确保它们有 gas
9. 给 operator、buyer、maker 三个钱包 mint 本地 USDT
10. 把最终生成的链配置写到：
    - [`.run/dev/local-chain.env`](/Users/zhangza/code/funnyoption/.run/dev/local-chain.env)
11. 把测试钱包材料写到：
    - [`.run/dev/local-chain-wallets.env`](/Users/zhangza/code/funnyoption/.run/dev/local-chain-wallets.env)

### 第三步：业务服务会继续启动

本地链准备好后，`dev-up.sh` 会继续启动 FunnyOption 的业务服务，包括：

- `account`
- `matching`
- `ledger`
- `settlement`
- `chain`
- `api`
- `ws`
- `web`
- `admin`

其中最关键的是：

- `chain-service` 现在会直接读取本地链配置，监听你刚部署出来的 `FunnyVault`
- 前端也会拿到同一套本地链参数，方便自动切链

## 启动完成后要检查什么

### 第一步：看脚本输出

正常情况下，你会在终端看到类似这些信息：

- 本地 `anvil` 已启动
- 本地 `MockUSDT` 已部署
- 本地 `FunnyVault` 已部署
- 生成的 env 文件路径
- 生成的钱包文件路径

### 第二步：确认生成文件存在

重点检查这两个文件：

- [`.run/dev/local-chain.env`](/Users/zhangza/code/funnyoption/.run/dev/local-chain.env)
- [`.run/dev/local-chain-wallets.env`](/Users/zhangza/code/funnyoption/.run/dev/local-chain-wallets.env)

前者是“链配置”，后者是“测试钱包”。

### 第三步：确认 API 和前端已起来

你可以手动验证：

```bash
curl http://127.0.0.1:8080/healthz
```

前端和 admin 默认地址：

- 用户端：`http://127.0.0.1:3000`
- Admin：`http://127.0.0.1:3001`

## 三把测试钱包分别是什么

生成的钱包文件：

- [`.run/dev/local-chain-wallets.env`](/Users/zhangza/code/funnyoption/.run/dev/local-chain-wallets.env)

里面会有三种身份。

### 1. operator 钱包

它代表“平台运营侧钱包”，不是普通用户钱包。

用途：

- admin 创建市场
- admin 首发流动性
- admin 结算市场

特点：

- 会自动进入 `FUNNYOPTION_OPERATOR_WALLETS`
- 也是默认的本地链 operator key

### 2. buyer 钱包

它对应本地用户 `1001`。

用途：

- 钱包登录
- session 授权
- 充值
- 买单
- claim payout

### 3. maker 钱包

它对应本地用户 `1002`。

用途：

- 作为对手盘
- 卖单
- 持有显式 first-liquidity inventory

## 怎么把钱包导入 MetaMask

### 第一步：打开钱包文件

打开：

- [`.run/dev/local-chain-wallets.env`](/Users/zhangza/code/funnyoption/.run/dev/local-chain-wallets.env)

你会看到类似这些字段：

- `FUNNYOPTION_LOCAL_CHAIN_OPERATOR_PRIVATE_KEY`
- `FUNNYOPTION_LOCAL_CHAIN_BUYER_PRIVATE_KEY`
- `FUNNYOPTION_LOCAL_CHAIN_MAKER_PRIVATE_KEY`

### 第二步：导入到 MetaMask

在 MetaMask 中选择“导入账户”，把这些测试私钥逐个导入。

建议至少导入两把：

- `operator`
- `buyer`

如果你要手动验证双边下单，再导入：

- `maker`

### 第三步：切到本地链

现在前端会收到与本地链一致的配置，包含：

- `chainId = 31337`
- `rpcUrl = http://127.0.0.1:8545`
- 原生币名 `ETH`

如果前端触发 `wallet_addEthereumChain`，MetaMask 应该能切到这条本地链。

## 手动验证完整链路

下面是一条推荐的“人工联调路径”。

### 步骤 1：用 operator 打开 admin

访问：

- `http://127.0.0.1:3001`

使用 `operator` 钱包连接。

你现在是在“平台运营侧”视角。

### 步骤 2：admin 创建市场

在 admin 里创建一个新市场。

这一步实际会走平台 operator lane，最终进入共享 API 的市场创建接口，而不是普通用户下单接口。

建议你创建时确认这些字段是合理的：

- 标题
- 描述
- 开盘时间
- 收盘时间
- 结算时间
- collateral 资产

成功后，记下新的 `market_id`。

### 步骤 3：admin 首发流动性

在新市场上执行 first-liquidity。

这里的含义不是“偷偷给市场一个隐藏种子单”，而是显式给 maker 用户发 inventory，再通过受保护的 admin lane 发第一笔 bootstrap order。

这一步的目的是让后续用户买单时，市场里真的有可撮合对手盘。

### 步骤 4：buyer 在用户端连接钱包

访问：

- `http://127.0.0.1:3000`

用 `buyer` 钱包连接。

这一步会建立浏览器里的钱包身份。

### 步骤 5：buyer 授权 session

用户端交易并不是每次都让主钱包直接签所有订单，而是先做一次 session 授权。

这一层的好处是：

- 钱包只在关键时刻弹出
- 后续订单可以由 session key 签名

完成后，后端会生成一个有效 session，后续下单使用这条 session。

### 步骤 6：buyer 充值

现在不是直接调 HTTP 写数据库，而是真正走链上存款路径：

1. `buyer` 钱包先对 `MockUSDT` 执行 `approve`
2. 然后对 `FunnyVault` 执行 `deposit`
3. `FunnyVault` 发出 `Deposited(address,uint256)` 事件
4. [listener.go](/Users/zhangza/code/funnyoption/internal/chain/service/listener.go) 里的 `DepositListener` 轮询链上日志
5. 监听器拿到钱包地址后，会去查有没有活动中的 wallet session
6. 找到对应用户后，写入 `chain_deposits`
7. 再通过 processor 调用 account 入账
8. 用户链下余额增加

你在产品侧应该能看到：

- deposit 记录出现
- USDT 可用余额增加

### 步骤 7：buyer 下单

选择刚才创建的市场，发一笔买单。

这里实际发生的是：

1. 前端构造订单 intent
2. 使用 session key 对订单签名
3. API 校验 session 和签名
4. API 调 `account` 做预冻结
5. API 把订单命令发到 Kafka
6. `matching` 消费订单命令

### 步骤 8：matching 撮合

如果前面 first-liquidity 已经成功，这时市场里应该有可成交的对手盘。

撮合发生后，你应该看到：

- 订单状态更新
- trade 记录生成
- 余额 / 仓位变化
- 相关事件继续流向 account / ledger / ws / settlement 所需链路

### 步骤 9：admin 结算市场

回到 admin，用 `operator` 钱包执行市场结算。

这一步会把市场推到终态，并触发后续 payout 逻辑。

### 步骤 10：检查终态数据

结算后，重点看这些结果：

- 市场状态是否是 `RESOLVED`
- payout 是否已生成
- 订单是否进入正确终态
- 持仓是否按结算结果更新
- 用户余额是否反映结算收益

## 自动化验证：运行 lifecycle 脚本

如果你不想手动点完整流程，可以跑：

```bash
/Users/zhangza/code/funnyoption/scripts/local-lifecycle.sh
```

这个脚本本身很薄，它做的事主要是：

1. 先加载：
   - [`.env.local`](/Users/zhangza/code/funnyoption/.env.local)
   - [`.run/dev/local-chain.env`](/Users/zhangza/code/funnyoption/.run/dev/local-chain.env)
2. 然后执行：

```bash
go run ./cmd/local-lifecycle
```

### 这个自动脚本会做什么

在持久本地链模式下，[persistent_env.go](/Users/zhangza/code/funnyoption/cmd/local-lifecycle/persistent_env.go) 里的逻辑会被启用。

也就是说，它会：

1. 读取当前链配置
2. 连接本地 RPC：`http://127.0.0.1:8545`
3. 读取本地 `FunnyVault` 地址
4. 读取本地 `MockUSDT` 地址
5. 为 buyer 创建真实 session
6. 为 maker 创建真实 session
7. 用 buyer 钱包先发 `approve`
8. 再发 `deposit`
9. 等待链上交易 receipt 成功
10. 再等待运行中的 `chain-service` 真正监听到 deposit
11. 确认 deposit 已出现在读接口里
12. 显式发 first-liquidity
13. 发卖单
14. 发买单
15. 等待 trade 生成
16. 触发市场结算
17. 等待 payout 和终态读接口正确返回
18. 最后输出一份 JSON 总结

### 自动脚本输出里重点看什么

重点关注这些字段：

- `proof_environment.mode`
- `market_id`
- `deposit_id`
- `deposit_tx_hash`
- `deposit_block_number`
- `deposit_vault_address`
- `trade_id`
- `market.status`
- `market.resolved_outcome`
- `payout.amount`

如果这些值都正常，就说明“创建市场 -> 充值 -> 下单 -> 撮合 -> 结算”这条链基本跑通了。

## 怎么确认 chain listener 真的在工作

这点很重要，因为本模式的关键就是“不是伪造入账”。

你可以从两层看。

### 第一层：看业务结果

如果 deposit 成功后，API 的 deposit 列表和余额列表都更新了，说明 listener 至少把事件处理通了。

### 第二层：看代码路径

当前监听逻辑在：

- [listener.go](/Users/zhangza/code/funnyoption/internal/chain/service/listener.go)

核心行为是：

1. 按配置的 `vaultAddress` 轮询日志
2. 只关心：
   - `Deposited(address,uint256)`
   - `WithdrawalQueued(bytes32,address,uint256,address)`
3. deposit 事件到来后，从日志里解析：
   - wallet 地址
   - amount
   - tx hash
   - log index
   - block number
4. 通过活动 session 反查用户
5. 调 processor 落库并触发入账

也就是说，如果一个钱包没有活动 session，即使它往 vault 里打了 deposit，当前实现也会跳过，不给用户入账。

## 常见问题

### 1. `forge build` 失败

最常见原因是第一次安装 Foundry 编译器时，需要联网下载 `solc 0.8.24`。

表现通常是：

- `forge build` 卡住
- 或报下载失败
- 或出现 `429 Too Many Requests`

解决思路：

1. 稍后重试一次
2. 在正常终端里单独跑一次 `forge build`
3. 等 Foundry 把编译器装好后，再重新执行 `dev-up.sh`

### 2. `8545` 端口被占用

如果本机已经有别的链进程占用了 `8545`，`local-chain-up.sh` 会直接失败，不会强行覆盖。

你需要先停掉旧进程，再重启：

```bash
/Users/zhangza/code/funnyoption/scripts/dev-down.sh
```

然后再跑：

```bash
/Users/zhangza/code/funnyoption/scripts/dev-up.sh
```

### 3. 钱包能 deposit，但系统没入账

优先检查这几项：

1. `buyer` 是否已经完成 session 授权
2. `chain-service` 是否正常运行
3. `FUNNYOPTION_VAULT_ADDRESS` 是否来自最新生成的 `local-chain.env`
4. `buyer` 是否把交易打到了正确的本地链和正确的 vault 地址

因为当前 listener 会用“活动 session -> user_id”做归属映射，所以没有活动 session 的钱包不会被记账。

### 4. 前端没自动切到本地链

优先检查：

1. `dev-up.sh` 是否成功写出了新的前端 env
2. 前端是否是本次启动后重新起来的
3. MetaMask 是否允许添加新链

### 5. 为什么还是看到旧链配置

通常是以下原因：

1. 没开 `FUNNYOPTION_LOCAL_CHAIN_MODE=anvil`
2. `dev-up.sh` 没完整跑完
3. 前端或服务还在使用旧进程

最稳的方式是：

```bash
/Users/zhangza/code/funnyoption/scripts/dev-down.sh
/Users/zhangza/code/funnyoption/scripts/dev-up.sh
```

## 停止环境

执行：

```bash
/Users/zhangza/code/funnyoption/scripts/dev-down.sh
```

这会把业务服务和受管的本地 `anvil` 一起停掉。

本地链的 pid 文件在：

- [`.run/dev/anvil.pid`](/Users/zhangza/code/funnyoption/.run/dev/anvil.pid)

## 推荐的实际使用顺序

如果你要最稳地跑一遍，我建议按这个顺序：

1. 在 [`.env.local`](/Users/zhangza/code/funnyoption/.env.local) 写入 `FUNNYOPTION_LOCAL_CHAIN_MODE=anvil`
2. 执行 [dev-up.sh](/Users/zhangza/code/funnyoption/scripts/dev-up.sh)
3. 确认 [local-chain.env](/Users/zhangza/code/funnyoption/.run/dev/local-chain.env) 和 [local-chain-wallets.env](/Users/zhangza/code/funnyoption/.run/dev/local-chain-wallets.env) 已生成
4. 导入 `operator` 和 `buyer` 到 MetaMask
5. 打开 admin，创建市场并首发流动性
6. 打开用户端，用 `buyer` 做 session 授权和充值
7. 下单撮合
8. 回 admin 结算
9. 检查终态数据
10. 如果想一键验证，再跑一次 [local-lifecycle.sh](/Users/zhangza/code/funnyoption/scripts/local-lifecycle.sh)

## 安全提醒

- 这里只能使用本地测试钱包
- 这些私钥只允许用于本地开发
- 不要把这些私钥拿去测试网或主网复用
- 不要把本地钱包材料提交进 git
