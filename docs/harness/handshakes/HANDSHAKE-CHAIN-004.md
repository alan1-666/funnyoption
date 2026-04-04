# HANDSHAKE-CHAIN-004

## Task

- [TASK-CHAIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-004.md)

## Thread owner

- chain/platform worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `WORKLOG-STAGING-001.md`
- `docs/deploy/staging-bsc-testnet.md`
- `docs/architecture/direct-deposit-session-key.md`
- this handshake
- `WORKLOG-CHAIN-004.md`

## Files in scope

- `internal/chain/service/**`
- `docs/deploy/staging-bsc-testnet.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-004.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-004.md`
- smallest migration/doc updates for a persisted listener cursor if needed

## Inputs from other threads

- staging worker evidence:
  - fresh tx `0x4129a4db5f66760ca8374a1dbe3df94652552df9768500ff0d49ec9654733a6c` succeeded on BSC Testnet at block `99674293`
  - `/api/v1/deposits?user_id=1074720` and `/api/v1/balances?user_id=1074720` stayed empty
- commander server check:
  - `FUNNYOPTION_CHAIN_RPC_URL=https://bsc-testnet-rpc.publicnode.com`
  - `FUNNYOPTION_CHAIN_START_BLOCK=99452107`
  - chain container logs repeatedly say `deposit poll failed: History has been pruned for this block`

## Outputs back to commander

- root cause and final restart-safe listener behavior
- patch summary and changed files
- validation commands and proof snippets
- any one-time staging recovery step
- residual replay/skip tradeoffs if an old pruned range cannot be scanned
- implementation note:
  - `chain` now persists vault scan progress in `chain_listener_cursors`
  - restart resumes from `max(FUNNYOPTION_CHAIN_START_BLOCK, persisted_next_block)`
  - a pruned-history RPC error triggers an explicit fast-forward to `safeHead + 1`, logs the skipped `[from_block, to_block]` interval, and persists the new cursor

## Blockers

- do not print `.secrets` or private-key plaintext
- do not silently fast-forward past pruned history without recording the skipped range and replay consequence
- do not touch order/portfolio product files owned by `TASK-API-005` or `TASK-OFFCHAIN-011`
- none open for `TASK-CHAIN-004` closure

## Status

- completed

## Handoff notes

- code/docs changes are in place under the declared ownership set plus
  `migrations/009_chain_listener_cursors.sql` and `docs/sql/schema.md`
- targeted chain-service tests pass locally:
  - `go test ./internal/chain/service/...`
  - `go test -run TestDepositListenerPollOnceCreditsFreshDepositAfterPrunedFastForwardRestart -v ./internal/chain/service/...`
- live staging restart/deposit smoke passed after applying the runtime chain patch
  and `migrations/009_chain_listener_cursors.sql` to
  `/opt/funnyoption-staging`, running the migrate profile, and rebuilding
  `chain`
- runbook recovery SQL snippets now use container-native `POSTGRES_USER` /
  `POSTGRES_DB` plus `docker compose --env-file deploy/staging/.env.staging`
  and document the host-shell `source deploy/staging/.env.staging` path for
  DSN-based commands
- pruned-range fast-forward was explicitly logged as `[99452107,99679358]`; any
  historical deposit/withdrawal events only inside that interval still need an
  archival RPC replay or manual backfill
- fresh post-restart proof:
  - `user_id=1430496`
  - `deposit_tx=0xa598e8cf7022a67ee27f4ba7f075ed4b3d6d027a93f9f2e96047b1a5094759b0`
  - `deposit_id=dep_09caa085db59579b1543683f64ae8238`
  - `/api/v1/balances?user_id=1430496` returned `USDT available=100 frozen=0`
- closeout results:
  - `docs/deploy/staging-bsc-testnet.md` recovery SQL now expands
    `$POSTGRES_USER` / `$POSTGRES_DB` inside `sh -lc` in the `postgres`
    container, uses a literal `psql -U funnyoption -d funnyoption` command for
    manual cursor fast-forward, and documents the `set -a && source
    deploy/staging/.env.staging && set +a` flow before injecting
    `FUNNYOPTION_POSTGRES_DSN` from the host shell
  - chain cursor code/schema patch was committed and pushed as `ea71dc8`
    (`Persist chain listener scan cursor`)
  - `/opt/funnyoption-staging` was normalized to a clean detached checkout on
    `HEAD=ea71dc8`; `git status --short` is empty and the Actions dirty-check
    guard condition would pass
  - `docker compose --env-file deploy/staging/.env.staging -f
    deploy/staging/docker-compose.staging.yml ps chain` shows
    `funnyoption-staging-chain-1` still `Up`
  - `GET https://funnyoption.xyz/api/v1/deposits?user_id=1430496&limit=20`
    still returns `deposit_id=dep_09caa085db59579b1543683f64ae8238`,
    `tx_hash=a598e8cf7022a67ee27f4ba7f075ed4b3d6d027a93f9f2e96047b1a5094759b0`,
    `status=CREDITED`, `amount=100`
  - `GET https://funnyoption.xyz/api/v1/balances?user_id=1430496&limit=20`
    still returns `asset=USDT`, `available=100`, `frozen=0`
