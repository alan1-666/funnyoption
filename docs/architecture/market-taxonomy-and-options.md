# 市场分类与选项模型设计

本文定义 FunnyOption 下一步的市场分类与选项存储方案。

目标有两个：

1. 增加一个正式的市场分类表，先内置 `加密` 和 `体育`
2. 增加一个市场选项表，但选项内容不是一行一个 option，而是按“一个市场一份 JSON 选项集”存储，方便后续从 admin 一次性写入多选项

这份设计**先解决 schema 和 API 模型**，并明确和当前二元市场引擎的兼容边界。

## 一、当前现状

当前仓库里：

- `markets` 只有一个 `metadata JSONB`
- 分类只是松散地塞在 `metadata.category`
- 交易、撮合、结算、仓位、资产命名都默认是二元 `YES / NO`

因此：

- “分类”需要先从 `metadata` 升级成正式表结构
- “多选项”可以先存储进后端，但**不能假装当前撮合引擎已经天然支持多选市场**

## 二、推荐表结构

### 1. 市场分类表：`market_categories`

```sql
CREATE TABLE market_categories (
    category_id     BIGSERIAL PRIMARY KEY,
    category_key    VARCHAR(32) NOT NULL UNIQUE,
    display_name    VARCHAR(64) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    status          VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    sort_order      INT NOT NULL DEFAULT 0,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      BIGINT NOT NULL DEFAULT 0,
    updated_at      BIGINT NOT NULL DEFAULT 0
);
```

推荐初始种子：

```text
CRYPTO / 加密 / sort_order=10
SPORTS / 体育 / sort_order=20
```

说明：

- `category_key` 是稳定机器值，例如 `CRYPTO`
- `display_name` 是中文展示名，例如 `加密`
- `metadata` 预留给图标、配色、排序标签等 UI 信息

### 2. 市场主表扩展：`markets.category_id`

在 `markets` 上新增正式外键：

```sql
ALTER TABLE markets
ADD COLUMN category_id BIGINT REFERENCES market_categories(category_id);
```

说明：

- 后续分类的 canonical source 是 `markets.category_id`
- `metadata.category` 暂时保留一段兼容期，用于老数据读面回填

### 3. 市场选项表：`market_option_sets`

不推荐做成一行一个 option 的 `market_options`，因为你已经明确希望“多选项用 JSON 字段一次存进去”。

推荐结构：

```sql
CREATE TABLE market_option_sets (
    market_id        BIGINT PRIMARY KEY REFERENCES markets(market_id) ON DELETE CASCADE,
    option_schema    JSONB NOT NULL,
    version          INT NOT NULL DEFAULT 1,
    created_at       BIGINT NOT NULL DEFAULT 0,
    updated_at       BIGINT NOT NULL DEFAULT 0
);
```

其中 `option_schema` 统一存数组 JSON：

```json
[
  {
    "key": "YES",
    "label": "是",
    "short_label": "是",
    "sort_order": 10,
    "is_active": true,
    "metadata": {
      "color": "#67e8d3"
    }
  },
  {
    "key": "NO",
    "label": "否",
    "short_label": "否",
    "sort_order": 20,
    "is_active": true,
    "metadata": {
      "color": "#5cbef2"
    }
  }
]
```

未来多选市场例子：

```json
[
  { "key": "ARS", "label": "阿森纳", "short_label": "阿森纳", "sort_order": 10, "is_active": true },
  { "key": "DRAW", "label": "平局", "short_label": "平局", "sort_order": 20, "is_active": true },
  { "key": "MCI", "label": "曼城", "short_label": "曼城", "sort_order": 30, "is_active": true }
]
```

## 三、为什么选“独立表 + JSONB”而不是只塞 metadata

如果只塞 `markets.metadata.options`：

- schema 不清晰
- 后续 admin / API 校验边界不明确
- 难以做版本管理
- 不方便表达“市场存在正式选项集”

如果做成 `market_option_sets.option_schema JSONB`：

- 写入仍然是一次性 JSON，符合你的要求
- 数据职责清晰
- 后面可以单独加版本、校验、审计
- 不会把 `markets` 主表变成越来越重的万能 JSON 容器

## 四、API 模型建议

### 1. 创建市场请求

当前 `CreateMarketRequest` 建议扩成：

```json
{
  "title": "今晚英超谁会赢？",
  "description": "以常规时间赛果为准。",
  "category_key": "SPORTS",
  "collateral_asset": "USDT",
  "status": "DRAFT",
  "open_at": 1775200000,
  "close_at": 1775800000,
  "resolve_at": 1775886400,
  "options": [
    { "key": "ARS", "label": "阿森纳", "short_label": "阿森纳", "sort_order": 10, "is_active": true },
    { "key": "DRAW", "label": "平局", "short_label": "平局", "sort_order": 20, "is_active": true },
    { "key": "MCI", "label": "曼城", "short_label": "曼城", "sort_order": 30, "is_active": true }
  ],
  "metadata": {}
}
```

建议新增字段：

- `category_key`
- `options`

### 2. 市场读模型

`GET /api/v1/markets` 和 `GET /api/v1/markets/:market_id` 的返回建议带出：

```json
{
  "market_id": 1775,
  "title": "今晚英超谁会赢？",
  "category": {
    "category_id": 2,
    "category_key": "SPORTS",
    "display_name": "体育"
  },
  "options": [
    { "key": "ARS", "label": "阿森纳", "short_label": "阿森纳", "sort_order": 10, "is_active": true },
    { "key": "DRAW", "label": "平局", "short_label": "平局", "sort_order": 20, "is_active": true },
    { "key": "MCI", "label": "曼城", "short_label": "曼城", "sort_order": 30, "is_active": true }
  ]
}
```

### 3. 市场列表过滤

`ListMarketsRequest` 可以加：

- `category_key`
- 后续如有需要再加 `category_id`

## 五、与当前交易引擎的兼容边界

这是这次设计里最重要的一点。

当前交易引擎仍然是**二元 outcome 模型**：

- 下单请求只有一个 `outcome string`
- matching book key 依赖 `market_id + outcome`
- settlement 用 `resolved_outcome`
- 仓位资产命名是 `POSITION:{market_id}:{OUTCOME}`

所以本次设计建议分两步走。

### Phase 1：先支持“分类 + 选项集存储”

这一阶段可以做：

- 新增 `market_categories`
- 新增 `markets.category_id`
- 新增 `market_option_sets.option_schema`
- API 能创建、读取、列出分类与选项集

但交易侧仍然限制：

- 可交易市场只能是二元选项
- 且选项 key 必须是 `YES / NO`

也就是说：

- `SPORTS` 分类可以先建
- 也可以先存三选项 JSON
- 但如果 options 不是 `YES / NO` 二元结构，就只能停留在 `DRAFT`，不能进入 `OPEN`

### Phase 2：真正支持多选市场交易

这个阶段才会涉及：

- order / trade / position / settlement 全链路改造成 N 选项
- 前端下单面板与 K 线切换改成动态 option 集合
- first-liquidity / resolution / payout 规则改成多选模型

这一阶段明显比“加表”大很多，应该单独立 task。

## 六、建议的校验规则

### 分类校验

- `category_key` 必须存在于 `market_categories`
- 只允许引用 `ACTIVE` 分类

### 选项 JSON 校验

`option_schema` 至少要满足：

- 必须是 JSON array
- 长度 >= 2
- 每个元素必须有：
  - `key`
  - `label`
  - `sort_order`
  - `is_active`
- `key` 在单个市场内必须唯一
- `label` 不能为空

### 交易兼容校验

如果市场要进入 `OPEN`：

- 当前版本强制 `len(options) == 2`
- 两个 key 必须是 `YES` / `NO`

否则：

- 返回 `400`
- 或要求市场只能保留在 `DRAFT`

## 七、数据迁移建议

### 1. 新建分类种子

插入：

- `CRYPTO / 加密`
- `SPORTS / 体育`

### 2. 回填老市场分类

从 `markets.metadata.category` 回填：

- `crypto` / `Crypto` -> `CRYPTO`
- `sports` / `Sports` -> `SPORTS`
- 其他未知值先回填到 `CRYPTO` 或保留 `NULL`，根据运营需要决定

### 3. 给老市场补默认二元选项集

所有老市场补一份默认 option_schema：

```json
[
  { "key": "YES", "label": "是", "short_label": "是", "sort_order": 10, "is_active": true },
  { "key": "NO", "label": "否", "short_label": "否", "sort_order": 20, "is_active": true }
]
```

这样不会破坏现有撮合和结算链路。

## 八、推荐落地顺序

### 第一步：schema

- migration 新增 `market_categories`
- migration 给 `markets` 加 `category_id`
- migration 新增 `market_option_sets`
- seed `CRYPTO` 和 `SPORTS`
- backfill 老市场分类与默认二元选项

### 第二步：API

- `CreateMarketRequest` 支持 `category_key + options`
- `MarketResponse` / `ListMarkets` 带出 `category + options`
- `ListMarketsRequest` 支持 `category_key`

### 第三步：交易保护

- market open / first-liquidity / order create 继续只接受 `YES / NO` 二元市场
- 非二元 option_schema 的市场可存储，但禁止 open/trade

## 九、最终建议

如果目标是“先把后端模型设计对，再不打爆现有交易引擎”，推荐的最终方案是：

- `market_categories`：正式分类表
- `markets.category_id`：市场归属分类
- `market_option_sets.option_schema JSONB`：每个市场一份选项集
- 现阶段只允许 `YES / NO` 选项集进入可交易状态
- 真正多选交易放到下一阶段单独做

这是风险最小、扩展性也最好的路径。
