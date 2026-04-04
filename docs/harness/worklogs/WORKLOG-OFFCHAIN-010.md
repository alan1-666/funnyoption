# WORKLOG-OFFCHAIN-010

### 2026-04-03 20:20 Asia/Shanghai

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `WORKLOG-API-004.md`
  - `docs/operations/core-business-test-flow.md`
- changed:
  - created a validation-first post-hardening regression task for the local core business flow
- validated:
  - task, handshake, ownership, and acceptance criteria are in repo files
  - this worker can run in parallel with `TASK-CHAIN-003` because it owns only its worklog and should not edit product code
- blockers:
  - none yet
- next:
  - launch a worker against `TASK-OFFCHAIN-010`

### 2026-04-03 20:35 Asia/Shanghai

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `HANDSHAKE-OFFCHAIN-010.md`
- changed:
  - paused this local regression task because the app is already deployed to staging and `TASK-STAGING-001` now has higher priority
- validated:
  - active plan and handshake status now match
- blockers:
  - none
- next:
  - resume only after the staging E2E lane no longer has a higher-priority blocker

### 2026-04-04 19:35 Asia/Shanghai

- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-OFFCHAIN-010.md`
  - `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-010.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-010.md`
  - `docs/operations/core-business-test-flow.md`
  - `docs/operations/local-lifecycle-runbook.md`
  - `docs/operations/local-offchain-lifecycle.md`
  - `docs/harness/worklogs/WORKLOG-API-004.md`
  - `docs/harness/worklogs/WORKLOG-API-005.md`
  - `docs/harness/worklogs/WORKLOG-STAGING-001.md`
  - `scripts/dev-up.sh`
  - `scripts/local-lifecycle.sh`
  - `scripts/local-chain-up.sh`
  - `cmd/local-lifecycle/main.go`
  - `internal/shared/auth/session.go`
  - `internal/api/dto/order.go`
  - `internal/api/dto/operator_auth.go`
- changed:
  - appended this local validation result only
- validated:
  - resolved local env bootstrap first:
    - initial `./scripts/dev-up.sh` failed because local postgres was not listening on `127.0.0.1:5432`
    - started Homebrew postgres with `brew services start postgresql@16`
    - created/verified the default local DSN target with:
      - `psql postgres -Atc "SELECT rolname FROM pg_roles WHERE rolname='funnyoption';" && createdb -O funnyoption funnyoption 2>/dev/null || true`
      - `PGPASSWORD=funnyoption psql 'postgres://funnyoption:funnyoption@127.0.0.1:5432/funnyoption?sslmode=disable' -Atc 'SELECT current_user, current_database();'`
  - core commands:
    - `./scripts/dev-up.sh`
    - `curl -sS http://127.0.0.1:8080/healthz`
    - `./scripts/local-lifecycle.sh`
    - `curl -sS 'http://127.0.0.1:8080/api/v1/orders?user_id=1002&market_id=1775302215404&limit=20'`
    - `curl -sS 'http://127.0.0.1:8080/api/v1/positions?user_id=1002&market_id=1775302215404&limit=20'`
    - `curl -sS 'http://127.0.0.1:8080/api/v1/balances?user_id=1002&limit=20'`
    - `set -a; source .run/dev/local-chain-wallets.env; set +a; MARKET_ID=1775302215404 node --input-type=module <<'NODE' ... NODE | tee /tmp/offchain-010-targeted-summary.json`
  - pass/fail matrix:

    | Check | Result | Evidence |
    | --- | --- | --- |
    | local API healthz | PASS | `GET http://127.0.0.1:8080/healthz` => `{"env":"local","service":"api","status":"ok"}` |
    | local stack on persistent anvil | PASS | `./scripts/dev-status.sh` showed `anvil`, `api`, `chain`, `matching`, `account`, `settlement`, `ledger`, `ws`, `web`, and `admin` all `up` |
    | listener-driven deposit credit | PASS | `./scripts/local-lifecycle.sh` created `market_id=1775302215404`, submitted deposit tx `0xd888d0c8e2d4323a2a81d15776438bd2060cd7354242b8c72f41750699b0a9cb`, and API readback returned `deposit_id=dep_fd6fe3438271a7cbd94ee762ba1a4b98`, `status=CREDITED`, `block_number=9`, buyer USDT `1000000 -> 1005000` |
    | first bootstrap sell is queued exactly once | PASS | `./scripts/local-lifecycle.sh` logged `first_liquidity_id=liq_1775302217259_519df16d2eb8`; `GET /api/v1/orders?user_id=1002&market_id=1775302215404` returned `order_id=ord_bootstrap_48c1970ee46193c456c29969718bfaf7`, `status=NEW`, `remaining_quantity=40` |
    | duplicate same-terms bootstrap semantic uniqueness | PASS | local targeted probe returned `409 {"error":"operator bootstrap order already accepted","order_id":"ord_bootstrap_48c1970ee46193c456c29969718bfaf7"}`; maker YES position stayed `40 -> 40` and maker USDT stayed `996000 -> 996000`; this matches the deployed staging verification in `WORKLOG-STAGING-001` on `125f9cd` |
    | normal session-backed user order after duplicate check | PASS | helper created `session_id=sess_c10dae9b3bc82bd93fbc7faa8c7dc5e6`; `POST /api/v1/orders` queued `buy_order_id=ord_1775302491293_30414a0bf998`; trade readback returned `trade_id=trd_1`, maker `ord_bootstrap_48c1970ee46193c456c29969718bfaf7`, `YES @ 58 x 1`; this matches the staging post-fix session order lane |
    | market resolution | PASS | `POST /api/v1/markets/1775302215404/resolve` returned `202`; `GET /api/v1/markets/1775302215404` returned `status=RESOLVED`, `resolved_outcome=YES`, `runtime.active_order_count=0`, `runtime.payout_count=2`; maker bootstrap order became `CANCELLED` with `cancel_reason=MARKET_RESOLVED` |
    | portfolio / orders / payouts read surfaces | PASS | buyer readback returned USDT `available=1005042`, YES position `quantity=1`, `settled_quantity=1`, order `ord_1775302491293_30414a0bf998` `FILLED`, payout `evt_settlement_1775302215404_1001_YES` `payout_amount=100 status=COMPLETED`; this matches the staging pass shape after the `125f9cd` deploy |
    | `scripts/local-lifecycle.sh` wrapper end-to-end | FAIL_BLOCKED | runner still assumes a second explicit sell-order call after `/api/v1/admin/markets/:market_id/first-liquidity`; after the bootstrap endpoint already queued `ord_bootstrap_48c1970ee46193c456c29969718bfaf7`, the runner exited on `create sell order: POST /api/v1/orders: rpc error: code = FailedPrecondition desc = insufficient available balance` |
  - local vs staging parity summary:
    - runtime business behavior is aligned with the latest verified staging behavior in `WORKLOG-STAGING-001`:
      - duplicate bootstrap returns `409` with the accepted `order_id` and no balance/position side effects
      - a normal session-signed taker order still matches the bootstrap sell
      - resolution cancels remaining maker liquidity and completes payouts
      - balances / positions / orders / payouts reads converge to the expected terminal state
    - the only mismatch found in local was the automation wrapper itself, not the shared API/runtime behavior
  - core ids and tx hashes:

    | Kind | Value |
    | --- | --- |
    | market_id | `1775302215404` |
    | first_liquidity_id | `liq_1775302217259_519df16d2eb8` |
    | bootstrap_order_id | `ord_bootstrap_48c1970ee46193c456c29969718bfaf7` |
    | duplicate_response_order_id | `ord_bootstrap_48c1970ee46193c456c29969718bfaf7` |
    | deposit_id | `dep_fd6fe3438271a7cbd94ee762ba1a4b98` |
    | deposit_tx_hash | `0xd888d0c8e2d4323a2a81d15776438bd2060cd7354242b8c72f41750699b0a9cb` |
    | buyer_session_id | `sess_c10dae9b3bc82bd93fbc7faa8c7dc5e6` |
    | buy_order_id | `ord_1775302491293_30414a0bf998` |
    | trade_id | `trd_1` |
    | payout_event_id | `evt_settlement_1775302215404_1001_YES` |
  - key response snippets:
    - duplicate bootstrap:
      - `409 {"error":"operator bootstrap order already accepted","order_id":"ord_bootstrap_48c1970ee46193c456c29969718bfaf7"}`
    - session order queue:
      - `{"command_id":"cmd_1775302491293_37828ebe3128","order_id":"ord_1775302491293_30414a0bf998","freeze_id":"frz_1775302491299_13f59165fed4","asset":"USDT","amount":58,"topic":"funnyoption.order.command","status":"QUEUED"}`
    - terminal market read:
      - `status=RESOLVED resolved_outcome=YES active_order_count=0 payout_count=2`
    - terminal buyer reads:
      - `balances`: USDT `available=1005042 frozen=0`
      - `positions`: YES `quantity=1 settled_quantity=1`
      - `orders`: `ord_1775302491293_30414a0bf998 FILLED`
      - `payouts`: `evt_settlement_1775302215404_1001_YES payout_amount=100 status=COMPLETED`
  - evidence paths:
    - targeted runtime proof JSON: `/tmp/offchain-010-targeted-summary.json`
    - local service logs: `/Users/zhangza/code/funnyoption/.logs/dev/api.log`, `/Users/zhangza/code/funnyoption/.logs/dev/chain.log`, `/Users/zhangza/code/funnyoption/.logs/dev/account.log`, `/Users/zhangza/code/funnyoption/.logs/dev/matching.log`
- blockers:
  - local `cmd/local-lifecycle` is now the narrow blocker for a green one-command local proof:
    - it still follows the pre-`TASK-API-005` two-step bootstrap flow and submits a second maker `SELL` after `/first-liquidity` has already queued the bootstrap order
    - smallest follow-up owner: `cmd/local-lifecycle/main.go` plus the local lifecycle docs owner (`docs/operations/local-lifecycle-runbook.md` / `docs/operations/local-offchain-lifecycle.md`)
    - no product-code regression evidence was found in `internal/api`, `internal/account`, `internal/matching`, or `internal/settlement` during this validation pass
- next:
  - hand back to commander that local runtime parity with staging is confirmed, but `TASK-OFFCHAIN-010` still needs one narrow follow-up if the repo wants `./scripts/local-lifecycle.sh` itself to pass again under the one-shot first-liquidity contract
