# Order Flow And Matching Center

## Core principle

Matching is the system's state arbitration center, not just a utility module.

- only one ordered writer can decide fills for the same order book
- the matching hot path must avoid synchronous multi-service RPC hops
- queue ordering is part of the correctness model, not just a performance optimization

## Hot path

```text
Client
  -> api-service
  -> risk/account pre-trade checks
  -> freeze balance
  -> publish Kafka command: order.command
  -> matching-service consumes ordered command
  -> match engine produces:
       - order.accepted / order.rejected
       - trade.matched
       - quote.depth
       - quote.ticker
  -> downstream consumers update account, ledger, positions, push WS
```

## Why not direct gRPC for order ingress

- synchronous RPC adds tail latency on the hottest path
- matching correctness depends on a single ordered intake channel
- Kafka makes back-pressure, replay, audit, and consumer decoupling easier
- API and matching can scale independently without turning matching into a multi-writer service

## Matching boundary

Matching is responsible for:

- consuming ordered order commands
- maintaining in-memory order books
- applying price-time priority
- producing deterministic trade and order events

Matching is not responsible for:

- user authentication
- balance freezing
- risk checks
- final ledger settlement
- websocket fanout

## Current V1 settlement note

- `api` pre-freezes quote collateral before publishing `order.command`
- freeze metadata travels with the command and order event
- `ledger` consumes `trade.matched` and writes the first append-only cash-leg entry
- `matching` also emits maker-side order updates so `account` can release or close older resting freezes
- position-leg accounting is deferred to the next iteration

## Ordering strategy

- one book key is `market_id:outcome`
- Kafka partition key should use the same book key
- one partition should be consumed by only one active matching worker at a time
- horizontal scaling is done by sharding books, not by allowing multiple writers on one book
