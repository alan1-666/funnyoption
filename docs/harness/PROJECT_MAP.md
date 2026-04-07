# Project Map

## Product shape

FunnyOption is a prediction-market MVP with:

- off-chain order entry and sequencing
- Kafka-centered matching
- mutable balance snapshots in `account`
- append-only evidence in `ledger`
- settlement and payout events
- BSC testnet vault integration
- Next.js user frontend with wallet/session UX
- a transitional operator surface in `web/app/admin` that is planned to move into a dedicated admin service

## Code paths by concern

### API and product entry

- [`backend/cmd/api`](/Users/zhangza/code/funnyoption/backend/cmd/api)
- [`backend/internal/api`](/Users/zhangza/code/funnyoption/backend/internal/api)

### Matching and market data

- [`backend/cmd/matching`](/Users/zhangza/code/funnyoption/backend/cmd/matching)
- [`backend/internal/matching/engine`](/Users/zhangza/code/funnyoption/backend/internal/matching/engine)
- [`backend/internal/matching/service`](/Users/zhangza/code/funnyoption/backend/internal/matching/service)
- [`backend/internal/shared/kafka`](/Users/zhangza/code/funnyoption/backend/internal/shared/kafka)

### Account snapshot and freezes

- [`backend/cmd/account`](/Users/zhangza/code/funnyoption/backend/cmd/account)
- [`backend/internal/account/service`](/Users/zhangza/code/funnyoption/backend/internal/account/service)
- [`backend/proto/account/v1/account.proto`](/Users/zhangza/code/funnyoption/backend/proto/account/v1/account.proto)

### Ledger and reconciliation

- [`backend/cmd/ledger`](/Users/zhangza/code/funnyoption/backend/cmd/ledger)
- [`backend/internal/ledger/service`](/Users/zhangza/code/funnyoption/backend/internal/ledger/service)

### Settlement

- [`backend/cmd/settlement`](/Users/zhangza/code/funnyoption/backend/cmd/settlement)
- [`backend/internal/settlement/service`](/Users/zhangza/code/funnyoption/backend/internal/settlement/service)

### Chain integration

- [`backend/cmd/chain`](/Users/zhangza/code/funnyoption/backend/cmd/chain)
- [`backend/internal/chain/service`](/Users/zhangza/code/funnyoption/backend/internal/chain/service)
- [`contracts/src/FunnyVault.sol`](/Users/zhangza/code/funnyoption/contracts/src/FunnyVault.sol)

### Realtime fanout

- [`backend/cmd/ws`](/Users/zhangza/code/funnyoption/backend/cmd/ws)
- [`backend/internal/ws/service`](/Users/zhangza/code/funnyoption/backend/internal/ws/service)

### Frontend

- [`web/app`](/Users/zhangza/code/funnyoption/web/app)
- [`web/components`](/Users/zhangza/code/funnyoption/web/components)
- [`web/lib`](/Users/zhangza/code/funnyoption/web/lib)

### Admin/operator surface

- current transitional shell: [`web/app/admin`](/Users/zhangza/code/funnyoption/web/app/admin)
- target architecture: a dedicated admin service, not a long-term route inside the public web app

## Reference docs

- Order path: [`/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md`](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- Ledger boundaries: [`/Users/zhangza/code/funnyoption/docs/architecture/ledger-service.md`](/Users/zhangza/code/funnyoption/docs/architecture/ledger-service.md)
- Direct deposit / session key: [`/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md`](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- Schema: [`/Users/zhangza/code/funnyoption/docs/sql/schema.md`](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- Topics: [`/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md`](/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md)

## Startup and local runtime

- One-click dev: [`/Users/zhangza/code/funnyoption/scripts/dev-up.sh`](/Users/zhangza/code/funnyoption/scripts/dev-up.sh)
- Stop stack: [`/Users/zhangza/code/funnyoption/scripts/dev-down.sh`](/Users/zhangza/code/funnyoption/scripts/dev-down.sh)
- Status: [`/Users/zhangza/code/funnyoption/scripts/dev-status.sh`](/Users/zhangza/code/funnyoption/scripts/dev-status.sh)
- Local env: [`/Users/zhangza/code/funnyoption/.env.local`](/Users/zhangza/code/funnyoption/.env.local)
- Core business test flow: [`/Users/zhangza/code/funnyoption/docs/operations/core-business-test-flow.md`](/Users/zhangza/code/funnyoption/docs/operations/core-business-test-flow.md)
- Persistent local chain: [`/Users/zhangza/code/funnyoption/docs/operations/local-persistent-chain.md`](/Users/zhangza/code/funnyoption/docs/operations/local-persistent-chain.md)

## Guidance

- Read the code path that owns the state you are changing.
- If a task spans more than one concern, the handshake must explicitly list the owned files.
- If a task changes messaging or ledger semantics, read both the architecture doc and the code path before editing.
- If a task touches admin/operator UX, decide first whether it belongs in the dedicated admin service target rather than the public `web` app shell.
