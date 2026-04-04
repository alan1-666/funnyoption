# WORKLOG-HARNESS-002

### 2026-04-05 01:41 CST

- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/architecture/direct-deposit-session-key.md`
  - `docs/architecture/oracle-settled-crypto-markets.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-012.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-015.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-016.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-006.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-007.md`
  - `docs/operations/local-lifecycle-runbook.md`
  - `docs/operations/local-offchain-lifecycle.md`
  - `docs/operations/local-persistent-chain.md`
  - `cmd/local-lifecycle/**`
  - `cmd/oracle/main.go`
  - `internal/oracle/service/**`
  - `internal/api/routes_auth.go`
  - `internal/api/handler/order_handler.go`
  - `internal/api/handler/sql_store.go`
- changed:
  - added a new `trading-key-oracle` flow mode inside `cmd/local-lifecycle`
    so one local acceptance runner now stitches together:
    - canonical trading-key challenge + `EIP-712` registration
    - truthful restore readback
    - persistent-local-chain deposit credit
    - oracle crypto market creation
    - order placement + matching
    - oracle auto settlement
    - payout + SQL/API readback
  - added `cmd/local-lifecycle/trading_key_oracle_flow.go` with:
    - local fake Binance HTTP fixture
    - real oracle worker startup in-process for the harness
    - pass/fail matrix output
    - key ids and readback command output
    - explicit residual blind-spot output
  - added `cmd/local-lifecycle/trading_key_oracle_flow_test.go` to prove the
    harness-side trading-key `EIP-712` signer round-trips through the existing
    server verifier
  - updated `cmd/local-lifecycle/main.go` so buyer/maker test wallets prefer
    `.run/dev/local-chain-wallets.env` and so the new flow is reachable through
    `--flow trading-key-oracle`
  - updated `scripts/local-lifecycle.sh` to source
    `.run/dev/local-chain-wallets.env`
  - added `scripts/local-full-flow.sh` as the new repeatable local acceptance
    wrapper
  - added `docs/operations/local-full-flow-acceptance.md` and linked it from
    the older local lifecycle docs
- validated:
  - formatting / syntax:
    - `gofmt -w cmd/local-lifecycle/main.go cmd/local-lifecycle/trading_key_oracle_flow.go cmd/local-lifecycle/trading_key_oracle_flow_test.go`
    - `bash -n scripts/local-lifecycle.sh scripts/local-full-flow.sh`
  - package test:
    - `go test ./cmd/local-lifecycle`
  - local runtime:
    - `./scripts/dev-up.sh`
    - `./scripts/local-full-flow.sh`
  - independent readbacks after the green run:
    - `curl -sS 'http://127.0.0.1:8080/api/v1/sessions?wallet_address=0x1532d37232c783c531bf0ce9860cb15f5f68aeb3&vault_address=0xe7f1725e7734ce288f8367e1bb143e90bb3f0512&status=ACTIVE&limit=20'`
    - `curl -sS 'http://127.0.0.1:8080/api/v1/markets/1775324420674'`
    - `curl -sS 'http://127.0.0.1:8080/api/v1/payouts?user_id=1001&market_id=1775324420674&limit=20'`
    - `psql 'postgres://funnyoption:funnyoption@127.0.0.1:5432/funnyoption?sslmode=disable' -c "SELECT market_id, status, resolved_outcome, resolver_type, resolver_ref FROM market_resolutions WHERE market_id = 1775324420674;"`
  - green full-flow ids:
    - `buyer_challenge_id=tkc_1775324418953_1f09a7ae8674`
    - `buyer_trading_key_id=tk_bbd7f736cfe67de562bb5a2ea3f077de`
    - `maker_challenge_id=tkc_1775324418970_433e10cc0b17`
    - `maker_trading_key_id=tk_292c2175015b3e7f08bfd89ad5eb8256`
    - `deposit_id=dep_3a098dd07d37a222a17bac6be8061c3e`
    - `deposit_tx_hash=0x6f146e49186ca15c969b62bde1b472ca7d9862f02a10dfff8ce981d1af15207c`
    - `market_id=1775324420674`
    - `first_liquidity_id=liq_1775324420683_275bbcf20d87`
    - `bootstrap_order_id=ord_bootstrap_a6aebe64954f73d8f745d48ec1ad6ffe`
    - `buy_order_id=ord_1775324421233_280fabf59405`
    - `trade_id=trd_3`
    - `payout_event_id=evt_settlement_1775324420674_1001_YES`
    - `resolution_resolver_ref=oracle_price:BINANCE:BTCUSDT:1775324440`
  - notable local blocker found and resolved during validation:
    - Docker Desktop was initially `paused`, which made `dev-up.sh` falsely
      treat Kafka `127.0.0.1:9092` as healthy while publishes timed out
    - after `docker desktop restart`, a fresh `dev-down.sh` + `dev-up.sh`
      cycle made the new full-flow script pass cleanly
- blockers:
  - no repo-code blocker remains for the harness/script/docs slice
  - residual blind spots remain explicit by design:
    - browser wallet popup / hardware wallet auth path is not covered
    - browser `localStorage` / IndexedDB restore behavior is not covered
    - live external oracle provider behavior is replaced with a local fixture
    - payout claim transaction UX is not covered
- next:
  - hand back changed files, validation commands, pass/fail matrix, and the
    remaining real-wallet / browser blind spots
