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
