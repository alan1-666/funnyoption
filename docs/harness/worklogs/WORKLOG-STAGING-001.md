# WORKLOG-STAGING-001

### 2026-04-03 20:35 Asia/Shanghai

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/deploy/staging-bsc-testnet.md`
  - `docs/operations/core-business-test-flow.md`
- changed:
  - created a staging E2E validation task and handshake for the deployed `funnyoption.xyz` / `admin.funnyoption.xyz` environment
  - recorded the current deployed domains in the staging deploy runbook
- validated:
  - `.secrets` is not tracked by git
  - `.secrets` is ignored by `.gitignore`
  - this worker can run in parallel with `TASK-CICD-001` because it owns only its worklog and should not edit workflow/deploy files
- blockers:
  - none yet; worker may still need a funded non-operator user wallet to complete the user-session + user-order leg on staging
- next:
  - launch a worker against `TASK-STAGING-001`

### 2026-04-03 21:23 Asia/Shanghai

- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-STAGING-001.md`
  - `docs/harness/handshakes/HANDSHAKE-STAGING-001.md`
  - `docs/harness/worklogs/WORKLOG-STAGING-001.md`
  - `docs/operations/core-business-test-flow.md`
  - `docs/deploy/staging-bsc-testnet.md`
  - `docs/harness/worklogs/WORKLOG-API-004.md`
  - `admin/lib/operator-auth.ts`
  - `admin/lib/operator-server.ts`
  - `admin/app/api/operator/markets/route.ts`
  - `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts`
  - `admin/app/api/operator/markets/[marketId]/resolve/route.ts`
  - `web/lib/session-client.ts`
  - `web/lib/chain.ts`
  - `web/app/portfolio/page.tsx`
  - `web/components/portfolio-shell.tsx`
  - `internal/shared/auth/session.go`
  - `internal/api/dto/operator_auth.go`
  - `internal/api/dto/market.go`
  - `internal/api/dto/order.go`
  - `internal/api/handler/order_handler.go`
- changed:
  - appended this staging E2E result entry only
- validated:
  - API health probe:
    - `GET https://funnyoption.xyz/healthz` => `{"env":"staging","service":"api","status":"ok"}`
  - browser smoke via Playwright:
    - `https://funnyoption.xyz/`
    - `https://funnyoption.xyz/markets/1775222028803`
    - `https://funnyoption.xyz/portfolio`
    - `https://admin.funnyoption.xyz/`
  - end-to-end staging script:
    - temporary script path: `/tmp/funnyoption-staging-e2e.mjs`
    - script did not print or write any private key plaintext
    - operator key was read from `.secrets` in-process only
    - generated a fresh non-operator taker wallet in memory and funded it from the operator wallet
    - created a session-backed taker order lane for a random user id
    - created a new market through the admin API, issued first liquidity, verified same-terms duplicate bootstrap rejection, matched a taker buy against the bootstrap sell, placed a second resting taker order, resolved the market, and polled user positions/orders/payouts/balance until terminal state

#### pass/fail matrix

| Check | Result | Evidence |
| --- | --- | --- |
| staging API healthz | PASS | `GET https://funnyoption.xyz/healthz` returned `status=ok` |
| user web homepage smoke | PASS | Playwright snapshot showed market list and `Connect` entry on `https://funnyoption.xyz/` |
| admin web homepage smoke | PASS | Playwright snapshot showed operator gate, market creation, first-liquidity, settlement, account snapshot, and trade/market read panels on `https://admin.funnyoption.xyz/` |
| admin operator auth + create market | PASS | signed `POST https://admin.funnyoption.xyz/api/operator/markets` created `market_id=1775222028803` |
| admin first-liquidity bootstrap | PASS | signed `POST https://admin.funnyoption.xyz/api/operator/markets/1775222028803/first-liquidity` returned `first_liquidity_id=liq_1775222029115_18d8ec99ef3e`, `order_id=ord_bootstrap_55fe1f888436a20cb974c1b2c193308b`, `order_status=QUEUED` |
| same-terms second bootstrap rejection | PASS | second signed call with fresh `requestedAt` returned `409 {"error":"issued first-liquidity liq_1775222029643_ccae65f30883 but failed to queue the first sell order: operator bootstrap order already accepted",...}` |
| user session authorization | PASS | `POST https://funnyoption.xyz/api/v1/sessions` created `session_id=sess_d4a076b28d70464125d88d130cb30688` for `user_id=502346`, wallet `0x155B6d24e13f586543ec6cAdce18B84AF68775ba` |
| user approve + deposit + chain credit | PASS | tx `0x0ea12a3acaad522510ede155f50e7a63c1049e4356eb1f21546bb3e27896b6f0` approved Vault, tx `0x4a6cba2992e5ac56df84cbd5809d94822bc9159a023b1e70b95abe36b4a4e993` deposited, API returned `deposit_id=dep_2a39afe98d2e37ebf4c5c51ddbe1eaea`, `available=500`, `frozen=0` |
| user matched order | PASS | session-signed `POST https://funnyoption.xyz/api/v1/orders` created `order_id=ord_1775222030363_01d24da67a49`; trade read returned `trade_id=trd_2` matched against `maker_order_id=ord_bootstrap_55fe1f888436a20cb974c1b2c193308b`, `price=58`, `quantity=1` |
| user resting order read | PASS | second taker order `ord_1775222030818_a222c6522c7b` appeared as `NEW`, `remaining_quantity=1`, `freeze_amount=57` before settlement |
| admin resolve market | PASS | signed `POST https://admin.funnyoption.xyz/api/operator/markets/1775222028803/resolve` resolved YES; market read returned `status=RESOLVED`, `resolved_outcome=YES` |
| user position/order/payout API reads | PASS | `GET /api/v1/positions?user_id=502346&market_id=1775222028803` returned YES position `quantity=1`, `settled_quantity=1`; `GET /api/v1/orders?...` returned one `FILLED` order and one `CANCELLED` order; `GET /api/v1/payouts?...` returned `event_id=evt_settlement_1775222028803_502346_YES`, `payout_amount=100`, `status=COMPLETED`; final balance `available=542`, `frozen=0` |
| market detail UI readback | PASS | Playwright snapshot on `https://funnyoption.xyz/markets/1775222028803` showed `已结算`, YES `100¢`, NO `0¢`, `累计成交额 0.58 USDT`, `成交笔数 1`, `挂单数量 0`, `赔付进度 2/2`, `结算结果 是` |
| admin readback for new market/trade | PASS | Playwright snapshot on `https://admin.funnyoption.xyz/` showed latest resolved market `#1775222028803 · Staging E2E Smoke 1775222027422`, one trade row `#1775222028803 · 是`, `1 份`, `58¢`, and settlement panel `最近已结算 #1775222028803 ... 是` |
| user `/portfolio` personalized browser readback for the random taker | FAIL | Playwright snapshot on `https://funnyoption.xyz/portfolio` still rendered wallet `0xc421d5ff322e4213a913ec257d6b4458af4255c6`, balance `0`, and operator-owned positions instead of random taker `user_id=502346`; `web/app/portfolio/page.tsx` server-fetches balances/positions/orders/payouts with default `user_id=1001` and `PortfolioShell` only refreshes profile from `session.userId`, not the position/order/payout collections |
| duplicate bootstrap side-effect atomicity | FAIL | duplicate bootstrap request correctly returned 409, but also issued `first_liquidity_id=liq_1775222029643_ccae65f30883`; maker position for `market_id=1775222028803` increased from `YES quantity=1` to `2`, and maker USDT balance changed from `available=960` to `958` |
| first-liquidity collateral unit scaling | FAIL | `internal/api/handler/order_handler.go` currently debits `req.Quantity` and returns `collateral_debit=req.Quantity`; staging evidence above shows two `qty=1` first-liquidity issuances only reduced maker USDT from `960` to `958`, while one winning taker position paid `100`, so maker inventory is materially under-collateralized |

#### core ids and tx hashes

| Kind | Value |
| --- | --- |
| market_id | `1775222028803` |
| taker_user_id | `502346` |
| taker_wallet | `0x155B6d24e13f586543ec6cAdce18B84AF68775ba` |
| taker_session_id | `sess_d4a076b28d70464125d88d130cb30688` |
| fund_tbnb_tx | `0x92094b23078a391697a6f98d1de893dfdd0490a716b3d79dce6b91c237e0ec8a` |
| fund_usdt_tx | `0xa9224a1303d08fc06bc28dea71958754a8dd0e7976a4d4c75a4c73e5df95a1be` |
| approve_tx | `0x0ea12a3acaad522510ede155f50e7a63c1049e4356eb1f21546bb3e27896b6f0` |
| deposit_tx | `0x4a6cba2992e5ac56df84cbd5809d94822bc9159a023b1e70b95abe36b4a4e993` |
| deposit_id | `dep_2a39afe98d2e37ebf4c5c51ddbe1eaea` |
| first_liquidity_id | `liq_1775222029115_18d8ec99ef3e` |
| duplicate_first_liquidity_id | `liq_1775222029643_ccae65f30883` |
| bootstrap_order_id | `ord_bootstrap_55fe1f888436a20cb974c1b2c193308b` |
| user_filled_order_id | `ord_1775222030363_01d24da67a49` |
| user_open_order_id | `ord_1775222030818_a222c6522c7b` |
| trade_id | `trd_2` |
| payout_event_id | `evt_settlement_1775222028803_502346_YES` |

- proof snippets:
  - market read after create:
    - `GET https://funnyoption.xyz/api/v1/markets/1775222028803` => `status=OPEN`, `category_key=CRYPTO`, options `YES/NO`
  - duplicate bootstrap response:
    - `409 {"error":"issued first-liquidity liq_1775222029643_ccae65f30883 but failed to queue the first sell order: operator bootstrap order already accepted","first_liquidity_id":"liq_1775222029643_ccae65f30883","operator_wallet_address":"0xc421d5ff322e4213a913ec257d6b4458af4255c6"}`
  - trade read:
    - `GET https://funnyoption.xyz/api/v1/trades?market_id=1775222028803&limit=20` => `trd_2`, taker `ord_1775222030363_01d24da67a49`, maker `ord_bootstrap_55fe1f888436a20cb974c1b2c193308b`, `YES @ 58 x 1`
  - user final reads:
    - `GET https://funnyoption.xyz/api/v1/positions?user_id=502346&market_id=1775222028803&limit=20` => YES `quantity=1`, `settled_quantity=1`
    - `GET https://funnyoption.xyz/api/v1/orders?user_id=502346&market_id=1775222028803&limit=20` => `ord_1775222030363_01d24da67a49` `FILLED`, `ord_1775222030818_a222c6522c7b` `CANCELLED`
    - `GET https://funnyoption.xyz/api/v1/payouts?user_id=502346&market_id=1775222028803&limit=20` => `evt_settlement_1775222028803_502346_YES`, `payout_amount=100`, `status=COMPLETED`
    - `GET https://funnyoption.xyz/api/v1/balances?user_id=502346&limit=20` => USDT `available=542`, `frozen=0`
- blockers:
  - no hard environment blocker for the API/business path; staging API, admin API, BSC Testnet RPC, chain deposit listener, matching, and settlement all completed the full flow
  - browser-wallet connect/sign flows were not executed through MetaMask because the headless Playwright runner has no wallet extension; operator and taker auth were validated through signed admin/user API requests instead
- follow-up:
  - `internal/api` owner: fix first-liquidity collateral scaling so paired inventory issuance debits accounting collateral in the same 2-decimal unit convention as payouts, not raw share quantity
  - `admin` + `internal/api` owner: make duplicate bootstrap handling atomic/idempotent before issuing extra first-liquidity inventory, or move the duplicate-semantic precheck ahead of inventory mutation
  - `web` owner: make `/portfolio` balance/positions/orders/payout collections use the connected session user instead of hard-coded `user_id=1001`, or add a client refresh path aligned with `session.userId`
- next:
  - hand this matrix and the regression follow-ups back to commander

### 2026-04-04 17:07 Asia/Shanghai

- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-STAGING-001.md`
  - `docs/harness/handshakes/HANDSHAKE-STAGING-001.md`
  - `docs/harness/worklogs/WORKLOG-STAGING-001.md`
  - `docs/operations/core-business-test-flow.md`
  - `docs/deploy/staging-bsc-testnet.md`
  - `docs/harness/worklogs/WORKLOG-API-004.md`
  - `admin/lib/operator-auth.ts`
  - `admin/lib/operator-server.ts`
  - `admin/app/api/operator/markets/route.ts`
  - `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts`
  - `admin/app/api/operator/markets/[marketId]/resolve/route.ts`
  - `web/lib/session-client.ts`
  - `internal/api/dto/order.go`
  - `internal/api/dto/market.go`
  - `internal/api/routes_reads.go`
  - `internal/api/handler/order_handler.go`
  - `contracts/src/FunnyVault.sol`
  - `/tmp/funnyoption-staging-e2e.mjs`
  - `/Users/zhangza/.codex/skills/playwright/SKILL.md`
- changed:
  - added `scripts/staging-concurrency-orders.mjs`
  - appended this staging retest result only
- validated:
  - script syntax and patch hygiene:
    - `node --check scripts/staging-concurrency-orders.mjs`
    - `git diff --check -- scripts/staging-concurrency-orders.mjs`
  - script command and bounded concurrency parameters:
    - `node scripts/staging-concurrency-orders.mjs --users 2 --seller-users 1 --orders-per-user 1 --concurrency 2 --bootstrap-price 58 --match-price 58 --poll-timeout-ms 60000`
    - generated taker users: `2`
    - preseed seller users: `1`
    - orders per user: `1`
    - max parallel user order pipelines: `2`
    - bootstrap/match price: `58¢ / 58¢`
    - bootstrap quantity: `1`
    - per-user deposit / token funding / gas funding: `5.00 USDT / 10.00 USDT / 0.03 tBNB`
    - the script loads the operator key from `/Users/zhangza/code/funnyoption/.secrets` or `FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY` in-process only, and does not print private key plaintext

#### pass/fail matrix

| Check | Result | Evidence |
| --- | --- | --- |
| staging API healthz | PASS | `GET https://funnyoption.xyz/healthz` returned `status=ok` through the script matrix |
| user web homepage smoke | PASS | Playwright snapshot on `https://funnyoption.xyz/` showed `Connect 点击完成钱包授权`, category tabs, and the new market card linking to `/markets/1775293473345` |
| user market detail smoke | PASS | Playwright snapshot on `https://funnyoption.xyz/markets/1775293473345` showed title `Staging Concurrency`, YES/NO `50¢`, `挂单数量 1`, and the order panel in `未连接` state |
| admin web homepage smoke | PASS | Playwright snapshot on `https://admin.funnyoption.xyz/` showed operator gate, market creation, first-liquidity, settlement, account snapshot, trade/market read panels, and selected market `#1775293473345` |
| admin operator auth + create market | PASS | script-created `market_id=1775293473345` via signed `POST https://admin.funnyoption.xyz/api/operator/markets` |
| admin first-liquidity bootstrap | PASS | `first_liquidity_id=liq_1775293474007_ce099d24c7b1`, `bootstrap_order_id=ord_bootstrap_38f2f75c014d7f0a09b192a1a2cfac41`, maker order readback `status=NEW`, `remaining_quantity=1` |
| same-terms second bootstrap rejection | PASS | second signed bootstrap call with fresh `requestedAt` returned `409` and body `{"error":"issued first-liquidity liq_1775293474898_9da626825742 but failed to queue the first sell order: operator bootstrap order already accepted",...}` |
| duplicate bootstrap side-effect atomicity | FAIL | despite the `409`, maker YES position changed `1 -> 2`, maker USDT available changed `1105 -> 1104`, and response exposed `duplicate_first_liquidity_id=liq_1775293474898_9da626825742` |
| first-liquidity collateral unit scaling | FAIL | maker USDT available changed `1106 -> 1105` for `quantity=1`, but one full YES/NO pair should debit `100` accounting units; script anomaly `first-liquidity-collateral-unit-mismatch` |
| user wallet/session/deposit setup | FAIL | generated `user_id=1074720`, wallet `0x7dD78D95C6a3ACeD695B6b3B349e391f1516A2Ea`, session created, tx `0x4129a4db5f66760ca8374a1dbe3df94652552df9768500ff0d49ec9654733a6c` succeeded on-chain at block `99674293`, but `/api/v1/deposits` and `/api/v1/balances` stayed empty and the script timed out with `wait deposit credited user=1074720 timeout after 60000ms; last=null` |
| concurrent burst order submit + matching | FAIL_BLOCKED | not reached because the first generated user's deposit never appeared in the off-chain reads; script aggregate remained `submitted_orders=0`, `success_orders=0`, `failed_orders=0`, `matched_trade_count=0`, `remaining_open_orders_after_match=0`, latency summary all `0` |
| post-match consistency checks (duplicate-fill / overfill / negative-balance / stale-freeze) | NOT_REACHED | the script implements these checks after final market/order/freeze snapshots, but this run stopped in deposit setup before the burst phase |
| admin resolve + user settlement/payout reads | FAIL_BLOCKED | not reached because user deposit setup failed before order placement |
| user `/portfolio` personalized browser readback | FAIL | Playwright snapshot on `https://funnyoption.xyz/portfolio` still rendered wallet `0xc421d5ff322e4213a913ec257d6b4458af4255c6`, balance `0`, and old user-1001-like positions/orders instead of the generated taker user |

#### script aggregate result

| Metric | Value |
| --- | --- |
| status | `FAIL` |
| submitted_orders | `0` |
| success_orders | `0` |
| failed_orders | `0` |
| matched_trade_count | `0` |
| matched_quantity | `0` |
| remaining_open_orders_after_match | `0` |
| remaining_open_orders_after_resolve | `0` |
| latency_summary_ms | `{count:0,min_ms:0,p50_ms:0,p95_ms:0,p99_ms:0,max_ms:0,avg_ms:0}` |
| anomalies | `duplicate-bootstrap-side-effect-position`, `duplicate-bootstrap-side-effect-balance`, `first-liquidity-collateral-unit-mismatch` |

#### core ids and tx hashes

| Kind | Value |
| --- | --- |
| market_id | `1775293473345` |
| operator_wallet | `0xC421d5Ff322e4213A913ec257d6b4458af4255c6` |
| first_liquidity_id | `liq_1775293474007_ce099d24c7b1` |
| duplicate_first_liquidity_id | `liq_1775293474898_9da626825742` |
| bootstrap_order_id | `ord_bootstrap_38f2f75c014d7f0a09b192a1a2cfac41` |
| generated_seller_user_id | `1074720` |
| generated_seller_wallet | `0x7dD78D95C6a3ACeD695B6b3B349e391f1516A2Ea` |
| fund_tbnb_tx | `0x6f476fdf1b9247a23bd5799923f13dc2e51c188354ffaa8e23d5e9dfc551e6fa` |
| fund_usdt_tx | `0x028a6eaefcab6d8ce5161006da48f84bdcb565fba2babd63bacb09c5d7973d8a` |
| approve_tx | `0xa5d3ec773978b8570e61f512a3946f63f0a34643de7faba7b52f730cd45e1608` |
| deposit_tx | `0x4129a4db5f66760ca8374a1dbe3df94652552df9768500ff0d49ec9654733a6c` |
| trade_id | `not reached` |
| payout_event_id | `not reached` |

- proof snippets:
  - duplicate bootstrap response:
    - `409 {"error":"issued first-liquidity liq_1775293474898_9da626825742 but failed to queue the first sell order: operator bootstrap order already accepted","first_liquidity_id":"liq_1775293474898_9da626825742","operator_wallet_address":"0xc421d5ff322e4213a913ec257d6b4458af4255c6"}`
  - duplicate bootstrap side effects:
    - maker YES position for `market_id=1775293473345`: `quantity=1` after first bootstrap, `quantity=2` after duplicate rejection
    - maker USDT balance: `available=1106` before first bootstrap, `1105` after first bootstrap, `1104` after duplicate rejection
  - chain deposit succeeded on BSC Testnet:
    - tx `0x4129a4db5f66760ca8374a1dbe3df94652552df9768500ff0d49ec9654733a6c` receipt `status=success`, `blockNumber=99674293`
    - receipt included `FunnyVault` log from `0x7665d943c62268d27ffcbed29c6a8281f7364534` at `logIndex=18`
  - off-chain deposit/balance reads stayed empty for the same wallet/user:
    - `GET https://funnyoption.xyz/api/v1/deposits?user_id=1074720&limit=20` => `{"items":[]}`
    - `GET https://funnyoption.xyz/api/v1/deposits?wallet_address=0x7dd78d95c6a3aced695b6b3b349e391f1516a2ea&limit=20` => `{"items":[]}`
    - `GET https://funnyoption.xyz/api/v1/balances?user_id=1074720&limit=20` => `{"items":[]}`
  - Playwright UI proof points:
    - `https://funnyoption.xyz/` snapshot showed `Staging Concurrency` market and `/markets/1775293473345`
    - `https://funnyoption.xyz/markets/1775293473345` snapshot showed `交易中`, `挂单数量 1`, `当前买是 50¢`, and `连接钱包`
    - `https://admin.funnyoption.xyz/` snapshot showed the new market selected in first-liquidity and settlement panels
    - `https://funnyoption.xyz/portfolio` snapshot still showed wallet `0xc421d5ff322e4213a913ec257d6b4458af4255c6`, balance `0`, and historical positions/orders unrelated to the generated taker
- blockers:
  - staging BSC Testnet deposit events are currently not reflected in `/api/v1/deposits` or `/api/v1/balances` for a fresh generated wallet/session, even though the Vault `deposit` tx succeeded on-chain; this blocks the concurrent order/matching phase and settlement verification in this task
  - headless Playwright still cannot perform a real MetaMask signature flow; browser auth-sensitive checks were validated through UI smoke plus API/chain proof, not wallet-extension clicks
- follow-up:
  - `internal/chain` + staging deploy owner: investigate why Vault `Deposited` event tx `0x4129a4db5f66760ca8374a1dbe3df94652552df9768500ff0d49ec9654733a6c` at block `99674293` never appeared in `chain_deposits` / balance reads for user `1074720`, with priority on chain listener liveness, configured `FUNNYOPTION_CHAIN_START_BLOCK`, RPC source, and wallet->user attribution
  - `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts` owner: make duplicate bootstrap handling atomic/idempotent so a second same-terms call cannot issue extra inventory before the duplicate-order rejection
  - `internal/api` first-liquidity owner: fix collateral debit units so one YES/NO pair debits `100 * quantity` accounting units rather than raw `quantity`
  - `web/app/portfolio` owner: stop `/portfolio` from rendering default operator/user-1001 data when a generated taker session/wallet is expected, and make balance/positions/orders/payouts follow the connected session user
- next:
  - hand this matrix, script command, key IDs, deposit-listener blocker, and the four follow-up owners back to commander

### 2026-04-04 18:41 Asia/Shanghai

- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-STAGING-001.md`
  - `docs/harness/handshakes/HANDSHAKE-STAGING-001.md`
  - `docs/harness/worklogs/WORKLOG-STAGING-001.md`
  - `docs/operations/core-business-test-flow.md`
  - `docs/deploy/staging-bsc-testnet.md`
  - `docs/harness/worklogs/WORKLOG-API-004.md`
  - `docs/harness/worklogs/WORKLOG-API-005.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-004.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-011.md`
  - `scripts/staging-concurrency-orders.mjs`
  - `web/lib/session-client.ts`
  - `web/components/trading-session-provider.tsx`
  - `web/components/portfolio-shell.tsx`
  - `web/components/shell-top-bar.tsx`
  - `web/app/portfolio/page.tsx`
  - `web/lib/api.ts`
  - `internal/account/service/sql_store.go`
  - `internal/api/handler/bootstrap_replay.go`
  - `/Users/zhangza/.codex/skills/playwright/SKILL.md`
  - `/Users/zhangza/.codex/skills/playwright/references/cli.md`
- changed:
  - updated `scripts/staging-concurrency-orders.mjs` so `status=CONSUMED, remaining_amount=0` is treated as a healthy terminal freeze state instead of a `stale-freeze` anomaly
  - appended this staging retest entry only
- validated:
  - script syntax:
    - `node --check scripts/staging-concurrency-orders.mjs`
  - bounded concurrency script command and parameters:
    - `node scripts/staging-concurrency-orders.mjs --users 4 --seller-users 2 --orders-per-user 2 --concurrency 4 --bootstrap-price 58 --match-price 58 --poll-timeout-ms 240000 --poll-interval-ms 3000`
    - generated taker users: `4`
    - preseed seller users: `2`
    - orders per user: `2`
    - max parallel user order pipelines: `4`
    - bootstrap/match price: `58¢ / 58¢`
    - bootstrap quantity: `4`
    - per-user deposit / token funding / gas funding: `5.00 USDT / 10.00 USDT / 0.03 tBNB`
    - operator key was read from `/Users/zhangza/code/funnyoption/.secrets` in-process only, and no private key plaintext was printed or written
  - browser smoke via Playwright:
    - `https://funnyoption.xyz/`
    - `https://funnyoption.xyz/markets/1775298754455`
    - `https://admin.funnyoption.xyz/`
    - `https://funnyoption.xyz/portfolio`
  - direct API readback probes for `/portfolio` diagnosis:
    - `GET https://funnyoption.xyz/api/v1/profile?user_id=1355871`
    - `GET https://funnyoption.xyz/api/v1/balances?user_id=1355871&limit=20`
    - `GET https://funnyoption.xyz/api/v1/positions?user_id=1355871&limit=20`
    - `GET https://funnyoption.xyz/api/v1/orders?user_id=1355871&limit=20`
    - `GET https://funnyoption.xyz/api/v1/payouts?user_id=1355871&limit=20`

#### pass/fail matrix

| Check | Result | Evidence |
| --- | --- | --- |
| staging API healthz | PASS | script matrix `healthz=PASS` |
| user web homepage smoke | PASS | Playwright snapshot on `https://funnyoption.xyz/` showed market cards, `Connect 点击完成钱包授权`, and new market card `/markets/1775298754455` with `4.64 USDT` and `6 笔成交` |
| user market detail smoke | PASS | Playwright snapshot on `https://funnyoption.xyz/markets/1775298754455` showed `已结算`, YES `100¢`, NO `0¢`, `成交笔数 6`, `挂单数量 0`, `赔付进度 3/3`, and `结算结果 是` |
| admin web homepage/readback smoke | PASS | Playwright snapshot on `https://admin.funnyoption.xyz/` showed operator gate, market creation, first-liquidity, settlement, account snapshots, trade/market read panels, latest settled market `#1775298754455`, and trade rows `#1775298754455 · 是` |
| admin operator auth + create market | PASS | script-created `market_id=1775298754455` via signed admin route |
| admin first-liquidity bootstrap | PASS | `first_liquidity_id=liq_1775298755096_3ccfeabbce21`, `bootstrap_order_id=ord_bootstrap_9d701d098ad842b6617c6ad39b53986d`, maker order readback `status=NEW`, `remaining_quantity=4` |
| duplicate same-terms bootstrap rejected | PASS | second same-terms bootstrap call returned HTTP `409`, body included `first_liquidity_id=liq_1775298755980_dd81453c86c6` |
| duplicate bootstrap no side effects | FAIL | despite the `409`, maker YES position changed `4 -> 8` and maker USDT available changed `1724 -> 1720`; anomaly codes `duplicate-bootstrap-side-effect-position` and `duplicate-bootstrap-side-effect-balance` |
| first-liquidity collateral scales as `100 * quantity` | FAIL | maker USDT available changed `1728 -> 1724` for `quantity=4`, but expected debit is `400` accounting units; anomaly code `first-liquidity-collateral-unit-mismatch` |
| fresh session + deposit credited | PASS | all 4 generated users received sessions and deposits; example `user_id=1355871`, `session_id=sess_25979fbabb8f477dc52daac5bc0acc9b`, `deposit_tx=0xec59d0c504744d1f4522d651cf085ede8bcd56b8abc0e1dedd19f791f6992d63`, `deposit_id=dep_0d16e469f410b347bd031094191da734`, `balance_after_deposit.available=500` |
| seller preseed inventory | PASS | seller users `1355869` and `1355870` got preseed orders `ord_1775298850241_7c74c86d8e09` and `ord_1775298851029_b75e83828f75` |
| concurrent order submit | PASS | script metrics `submitted_orders=8`, `success_orders=8`, `failed_orders=0`, latency p95 `179.75ms` |
| concurrent matching | PASS | script metrics `matched_trade_count=4`, `matched_quantity=4`, `remaining_open_orders_after_match=0` |
| duplicate-fill / overfill / negative-balance / stale-freeze under concurrency | PASS | script matrix `duplicate_fill=PASS`, `overfill=PASS`, `negative_balance=PASS`, `stale_freeze=PASS`; after script fix, terminal `CONSUMED` freezes with `remaining_amount=0` are no longer misclassified as stale |
| admin resolve + final settlement/payout reads | PASS | script matrix `admin_resolve_market=PASS`, `final_market_resolved=PASS`, `final_payouts_read=PASS`; final market readback `status=RESOLVED`, `resolved_outcome=YES`, `payout_count=3`, `completed_payout_count=3` |
| user `/portfolio` no-session state | FAIL | after clearing `localStorage['funnyoption:session:v1']` and reloading `https://funnyoption.xyz/portfolio`, Playwright snapshot still rendered wallet `0xc421d5ff322e4213a913ec257d6b4458af4255c6`, `持仓 10`, `开的订单 5`, and old operator/user-1001-like positions instead of an empty/disconnected state |
| user `/portfolio` current-session user collections | FAIL | after manually writing a local session for `user_id=1355871`, top bar wallet switched to `0xB03137e211884B3Cf61fD5300da7E5e3820b6aCD`, but balances/positions/orders/payouts list still showed the old user-1001 dataset; Playwright network log showed `GET /api/v1/profile?user_id=1355871` but no `balances/positions/orders/payouts?user_id=1355871` requests |

#### script aggregate result

| Metric | Value |
| --- | --- |
| status | `FAIL` |
| submitted_orders | `8` |
| success_orders | `8` |
| failed_orders | `0` |
| matched_trade_count | `4` |
| matched_quantity | `4` |
| remaining_open_orders_after_match | `0` |
| remaining_open_orders_after_resolve | `0` |
| latency_summary_ms | `{count:8,min_ms:170.83,p50_ms:175.69,p95_ms:179.75,p99_ms:179.75,max_ms:675.4,avg_ms:237.67}` |
| anomalies | `duplicate-bootstrap-side-effect-position`, `duplicate-bootstrap-side-effect-balance`, `first-liquidity-collateral-unit-mismatch` |

#### core ids and tx hashes

| Kind | Value |
| --- | --- |
| market_id | `1775298754455` |
| operator_wallet | `0xC421d5Ff322e4213A913ec257d6b4458af4255c6` |
| first_liquidity_id | `liq_1775298755096_3ccfeabbce21` |
| duplicate_first_liquidity_id | `liq_1775298755980_dd81453c86c6` |
| bootstrap_order_id | `ord_bootstrap_9d701d098ad842b6617c6ad39b53986d` |
| seller_user_1 | `1355869 / 0x83E3434C6B1d0D10880F6537641F062f5cbA05e2 / sess_4b9889dde2698a957d468bec5fa7a863 / dep_c25d1727fdda27ff62db5499a40eecb2 / 0xd5d262caaa124a30a1f4aef13d743f4a7e9f94f510f6f8fc4ef3db7ec7d1a28a / ord_1775298850241_7c74c86d8e09` |
| seller_user_2 | `1355870 / 0x39a20D3CA87315E03706c074A26c4942e2361b67 / sess_8f89b0091c012d0e81243136fc5e0850 / dep_f57b16b4491bdf44bfb62592e12ec35d / 0xfc160ca7d32109304494135d70872817940bab5000f86ce5ca392ea20ac264d1 / ord_1775298851029_b75e83828f75` |
| buyer_user_1 | `1355871 / 0xB03137e211884B3Cf61fD5300da7E5e3820b6aCD / sess_25979fbabb8f477dc52daac5bc0acc9b / dep_0d16e469f410b347bd031094191da734 / 0xec59d0c504744d1f4522d651cf085ede8bcd56b8abc0e1dedd19f791f6992d63` |
| buyer_user_2 | `1355872 / 0xCeC52703FC40fE00a2d2ae78B9178f37E2753398 / sess_c4f21c9e74fe7793e0a10c7228cf418d / dep_fd7eb95f1b1fe1042080ada87fc82c22 / 0xef27ea7ffdcf14759c5a90e8f2d92d91c8ad82087e29d6c94657567566d7bcd8` |
| preseed_order_ids | `ord_1775298850241_7c74c86d8e09`, `ord_1775298851029_b75e83828f75` |
| concurrent_trade_ids | `trd_14`, `trd_13`, `trd_12`, `trd_11` |
| all_trade_ids_for_market | `trd_14`, `trd_13`, `trd_12`, `trd_11`, `trd_10`, `trd_9` |
| payout_event_ids | `evt_settlement_1775298754455_1002_YES`, `evt_settlement_1775298754455_1355871_YES`, `evt_settlement_1775298754455_1355872_YES` |

- proof snippets:
  - duplicate bootstrap response:
    - `409 {"error":"issued first-liquidity liq_1775298755980_dd81453c86c6 but failed to queue the first sell order: operator bootstrap order already accepted","first_liquidity_id":"liq_1775298755980_dd81453c86c6","operator_wallet_address":"0xc421d5ff322e4213a913ec257d6b4458af4255c6"}`
  - duplicate bootstrap side effects:
    - maker YES position for `market_id=1775298754455`: `quantity=4` after first bootstrap, `quantity=8` after duplicate rejection
    - maker USDT balance: `available=1728` before first bootstrap, `1724` after first bootstrap, `1720` after duplicate rejection
  - fresh deposit and API portfolio reads for `user_id=1355871`:
    - `GET /api/v1/balances?user_id=1355871&limit=20` => USDT `available=584`, `frozen=0`
    - `GET /api/v1/positions?user_id=1355871&limit=20` => `market_id=1775298754455`, YES `quantity=2`, `settled_quantity=2`
    - `GET /api/v1/orders?user_id=1355871&limit=20` => `2` orders
    - `GET /api/v1/payouts?user_id=1355871&limit=20` => `evt_settlement_1775298754455_1355871_YES`, `status=COMPLETED`, `payout_amount=200`
    - `GET /api/v1/profile?user_id=1355871` => `wallet_address=0xb03137e211884b3cf61fd5300da7e5e3820b6acd`
  - `/portfolio` browser/network evidence:
    - no-session reload after clearing localStorage still rendered wallet `0xc421d5ff322e4213a913ec257d6b4458af4255c6` and the old 10-position / 5-order dataset
    - injected session for `user_id=1355871` changed the top-bar wallet to `0xB031...6aCD`, but the holdings/orders/payouts cards stayed on the old dataset
    - Playwright network log only showed `GET https://funnyoption.xyz/api/v1/profile?user_id=1355871`, with no `balances/positions/orders/payouts?user_id=1355871` collection requests
- blockers:
  - no hard chain/API blocker for fresh deposits anymore; CHAIN-004 staging fix appears effective for new deposits in this run
  - full PASS is still blocked by duplicate bootstrap inventory/balance side effects, first-liquidity collateral under-debiting, and stale `/portfolio` collection reads on staging
  - browser-wallet connect/sign was not exercised through a real MetaMask extension in Playwright; operator/user signatures were validated through API + localStorage injection + page/network readback instead
- follow-up:
  - `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts` + staging deploy owner: make same-terms duplicate bootstrap 409 fully side-effect-free; current staging still issues a second `first_liquidity_id` and mutates maker inventory/balance before rejecting the duplicate order
  - `internal/api/handler/order_handler.go` + staging deploy owner: fix first-liquidity collateral debit to `100 * quantity` accounting units; current staging still debits raw `quantity`
  - `web/components/portfolio-shell.tsx` + staging web deploy owner: make `balances/positions/orders/payouts` refresh on `session.userId` in the deployed bundle and reset to empty/no-session state when local session is absent; current staging only refreshes profile and leaves the old user-1001 collections visible
- next:
  - hand this updated matrix, bounded script command, aggregate metrics, core IDs, and the three remaining bug owners back to commander

### 2026-04-04 18:58 Asia/Shanghai

- read:
  - `docs/harness/handshakes/HANDSHAKE-STAGING-001.md`
  - `docs/harness/worklogs/WORKLOG-STAGING-001.md`
  - `docs/harness/worklogs/WORKLOG-API-005.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-011.md`
  - local `git status --short`
  - GitHub Actions run `23977457019`
- changed:
  - no product code in this follow-up; deployed the reviewed `TASK-API-005` + `TASK-OFFCHAIN-011` fixset already present in the workspace
  - updated the staging/API/OFFCHAIN handshakes and worklogs with deploy closeout details
- validated:
  - commit and push:
    - commit `125f9cd4af344680e78529c5a98358b39427e703`
    - subject `Deploy reviewed API-005 and OFFCHAIN-011 fixset`
    - `git push origin main` => `ea71dc8..125f9cd  main -> main`
  - GitHub Actions:
    - workflow: `staging-deploy`
    - run id: `23977457019`
    - run URL: `https://github.com/alan1-666/funnyoption/actions/runs/23977457019`
    - result: `success`
    - `validate` job:
      - `Run Go tests` `success`
      - `Build web app` `success`
      - `Build admin app` `success`
      - `Check shell script syntax` `success`
    - `deploy-staging` job:
      - `Deploy over SSH` `success`
  - staging server proof:
    - `ssh root@76.13.220.236 'cd /opt/funnyoption-staging && git rev-parse --short HEAD && git status --short && curl -sS https://funnyoption.xyz/healthz'`
    - checkout `HEAD=125f9cd`
    - checkout status: clean
    - `GET https://funnyoption.xyz/healthz` => `{"env":"staging","service":"api","status":"ok"}`
- blockers:
  - the old undeployed-fixset blocker is closed
  - runtime truth for the three previously failing staging assertions still needs a fresh `TASK-STAGING-001` rerun against deployed `125f9cd`
- next:
  - rerun `TASK-STAGING-001` on staging now that the reviewed API/web fixset is actually deployed

### 2026-04-04 19:18 Asia/Shanghai

- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-STAGING-001.md`
  - `docs/harness/handshakes/HANDSHAKE-STAGING-001.md`
  - `docs/harness/worklogs/WORKLOG-STAGING-001.md`
  - `docs/operations/core-business-test-flow.md`
  - `docs/deploy/staging-bsc-testnet.md`
  - `docs/harness/worklogs/WORKLOG-API-004.md`
  - `docs/harness/worklogs/WORKLOG-API-005.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-011.md`
  - `scripts/staging-concurrency-orders.mjs`
  - `web/lib/session-client.ts`
  - `/Users/zhangza/.codex/skills/playwright/SKILL.md`
  - `/Users/zhangza/.codex/skills/playwright/references/cli.md`
- changed:
  - appended this deployment-verification-only rerun entry to `docs/harness/worklogs/WORKLOG-STAGING-001.md`
- validated:
  - minimal staging verification script:
    - command: `node scripts/staging-concurrency-orders.mjs --users 2 --seller-users 1 --orders-per-user 1 --concurrency 1 --poll-timeout-ms 180000 --poll-interval-ms 3000`
    - full output saved to `/tmp/staging-mini-verify-pass.log`
    - summary: `SUMMARY status=PASS market_id=1775300707239 success_orders=2 failed_orders=0 matched_trade_count=1 remaining_open_orders_after_match=0 p95_latency_ms=192.71 anomalies=0`
  - Playwright browser verification:
    - no-session browser session: `fo-portfolio-nosession`
    - injected-session browser session: `fo-portfolio-injected`
    - screenshots:
      - `/Users/zhangza/code/funnyoption/.playwright-cli/page-2026-04-04T11-14-53-915Z.png`
      - `/Users/zhangza/code/funnyoption/.playwright-cli/page-2026-04-04T11-18-14-610Z.png`

#### deployment verification matrix

| Check | Result | Evidence |
| --- | --- | --- |
| duplicate bootstrap `409` has no inventory / balance side effect | PASS | `market_id=1775300707239`, first bootstrap `first_liquidity_id=liq_1775300708008_8009f863cb99`, `order_id=ord_bootstrap_9862e540bc8c118fbf924445d1d21f58`; duplicate call returned `409 {"error":"operator bootstrap order already accepted","order_id":"ord_bootstrap_9862e540bc8c118fbf924445d1d21f58",...}` with `duplicate_first_liquidity_id=""`; maker YES position stayed `1 -> 1`, maker USDT available stayed `2210 -> 2210` after the duplicate check |
| first-liquidity collateral debits `100 * quantity` | PASS | maker `user_id=1002` USDT available changed `2310 -> 2210` for bootstrap `quantity=1`, so debit was exactly `100`; script matrix recorded `first_liquidity_collateral=PASS` |
| `/portfolio` no-session no longer shows old user-1001 set | PASS | Playwright no-session snapshot showed `我的余额 — 未连接`, `持仓 —`, `开的订单 —`, `历史结算 —`, and `请先连接钱包并授权交易会话后查看持仓。`; browser network log showed only `GET https://funnyoption.xyz/?_rsc=17350 => 200 OK`, with no `profile/balances/positions/orders/payouts` private reads |
| `/portfolio` injected `session.userId` shows the connected user collection instead of old user-1001 set | PASS | injected local session for `user_id=1308872`, wallet `0xdf8b417a9040ddb89cab34ee798473a4fc14daf8`, `session_id=sess_2cb501a83d39e772dc5ac3ef2ce4a923`; snapshot showed `5.42 USDT`, `当前展示 user #1308872 的账户数据。`, `持仓 1`, `开的订单 0`, `历史结算 1`, and position row `Staging Concurrency / 1 份 / 已结算 1`; network log showed only `GET /api/v1/profile?user_id=1308872`, `GET /api/v1/balances?user_id=1308872&limit=10`, `GET /api/v1/positions?user_id=1308872&limit=20`, `GET /api/v1/orders?user_id=1308872&limit=20`, and `GET /api/v1/payouts?user_id=1308872&limit=20` |

#### key ids

| Kind | Value |
| --- | --- |
| market_id | `1775300707239` |
| maker_user_id | `1002` |
| first_liquidity_id | `liq_1775300708008_8009f863cb99` |
| bootstrap_order_id | `ord_bootstrap_9862e540bc8c118fbf924445d1d21f58` |
| seller_user | `1308871 / 0x546929F503eb529F43564d2C09deb9dbdC6dda12 / sess_98c8bc1ab721a7e36d86b9d7d9a49746 / ord_1775300758287_3ddb5b8ae35c` |
| buyer_user | `1308872 / 0xDF8B417A9040ddb89cAB34Ee798473A4Fc14daF8 / sess_2cb501a83d39e772dc5ac3ef2ce4a923` |
| concurrent_trade_ids | `trd_17`, `trd_18` |
| payout_event_id | `evt_settlement_1775300707239_1308872_YES` |

- proof snippets:
  - duplicate bootstrap response:
    - `POST https://admin.funnyoption.xyz/api/operator/markets/1775300707239/first-liquidity` duplicate attempt => `409 {"error":"operator bootstrap order already accepted","order_id":"ord_bootstrap_9862e540bc8c118fbf924445d1d21f58","operator_wallet_address":"0xc421d5ff322e4213a913ec257d6b4458af4255c6"}`
  - duplicate bootstrap atomicity:
    - maker YES position after first bootstrap => `quantity=1`
    - maker YES position after duplicate bootstrap => `quantity=1`
    - maker USDT available after first bootstrap => `2210`
    - maker USDT available after duplicate bootstrap => `2210`
  - first-liquidity collateral:
    - maker USDT available before first bootstrap => `2310`
    - maker USDT available after first bootstrap => `2210`
    - debit => `100`
  - `/portfolio` no-session browser/network evidence:
    - snapshot text included `未连接钱包时不会读取任何用户账户集合，请先连接钱包。`
    - Playwright network output contained no `user_id=1001` or private collection reads
  - `/portfolio` injected-session browser/network evidence:
    - snapshot text included `当前展示 user #1308872 的账户数据。`
    - Playwright network output showed only `user_id=1308872` for `profile`, `balances`, `positions`, `orders`, and `payouts`
- blockers:
  - none in this deployment-verification-only rerun
- next:
  - hand this PASS matrix, key ids, screenshot paths, and network evidence back to commander as the staging revalidation result for deployed `125f9cd`
