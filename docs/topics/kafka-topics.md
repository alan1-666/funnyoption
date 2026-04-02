# Kafka Topics

## Command topics

- `funnyoption.order.command`
  - produced by `api`
  - consumed by `matching`
  - key: `market_id:outcome`

## Event topics

- `funnyoption.order.event`
  - accepted / rejected / cancelled / partially-filled / filled
- `funnyoption.trade.matched`
  - canonical trade stream
- `funnyoption.position.changed`
  - position delta for downstream accounting
- `funnyoption.quote.depth`
  - depth snapshots or incremental depth updates
- `funnyoption.quote.ticker`
  - latest trade, best bid, best ask
- `funnyoption.market.event`
  - market state changes such as open, close, resolve
- `funnyoption.settlement.completed`
  - resolved payout events for downstream account and ledger
- `funnyoption.chain.deposit`
  - confirmed vault deposit credits for downstream ledger evidence
- `funnyoption.chain.withdrawal`
  - confirmed vault withdrawal queue debits for downstream ledger evidence

## First-stage consumer map

- `matching`
  - consumes `order.command`
  - produces `order.event`, `trade.matched`, `position.changed`, `quote.depth`, `quote.ticker`
- `account`
  - consumes `order.event`, `trade.matched`, `settlement.completed`
- `settlement`
  - consumes `position.changed`, `market.event`
  - produces `settlement.completed`
- `ledger`
  - consumes `trade.matched`, `settlement.completed`, `chain.deposit`, `chain.withdrawal`
- `ws`
  - consumes `quote.depth`, `quote.ticker`
- `chain`
  - persists vault deposits / withdrawals, calls `account.CreditBalance` and `account.DebitBalance`
  - produces `chain.deposit`, `chain.withdrawal`
