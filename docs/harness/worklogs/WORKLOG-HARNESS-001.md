# WORKLOG-HARNESS-001

### 2026-04-01 18:00 Asia/Shanghai

- read:
  - Harness Engineering article
  - existing FunnyOption docs and repo map
- changed:
  - added slim `AGENTS.md`
  - added `PLAN.md`
  - added harness roles, protocol, templates, tasks, handshakes, and prompts
- validated:
  - file structure is navigable and cross-linked
- blockers:
  - none
- next:
  - use commander prompt to open planning-only threads

### 2026-04-03 20:20 Asia/Shanghai

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `TASK-API-004.md`
  - `HANDSHAKE-API-004.md`
  - `WORKLOG-API-004.md`
  - `TASK-CHAIN-002.md`
  - `HANDSHAKE-CHAIN-002.md`
  - `WORKLOG-CHAIN-002.md`
- changed:
  - marked `TASK-API-004` complete in the active plan and refreshed the top-level next-task pointers in `PLAN.md`
  - created `TASK-OFFCHAIN-010`, `HANDSHAKE-OFFCHAIN-010`, and `WORKLOG-OFFCHAIN-010` for a validation-first post-hardening local regression pass
  - created `TASK-CHAIN-003`, `HANDSHAKE-CHAIN-003`, and `WORKLOG-CHAIN-003` for legacy local `chain_deposits` schema-drift cleanup
- validated:
  - reviewed the API-004 implementation path in `internal/api/dto/operator_auth.go`, `internal/api/handler/order_handler.go`, and the focused handler/router tests
  - verified `go test ./internal/api/...`
  - verified `cd /Users/zhangza/code/funnyoption/admin && npm run build`
  - active plan and handshakes now agree that `TASK-API-004` is complete and the next parallel lanes are `TASK-OFFCHAIN-010` plus `TASK-CHAIN-003`
- blockers:
  - none at the planning layer
- next:
  - launch one validation worker for `TASK-OFFCHAIN-010`
  - launch one chain/docs worker for `TASK-CHAIN-003`

### 2026-04-03 20:35 Asia/Shanghai

- read:
  - `docs/deploy/staging-bsc-testnet.md`
  - `configs/staging/funnyoption.env.example`
  - current git ignore status for `.secrets`
- changed:
  - recorded the current deployed domains `https://funnyoption.xyz/` and `https://admin.funnyoption.xyz/` in the staging deployment runbook
  - created `TASK-STAGING-001`, `HANDSHAKE-STAGING-001`, and `WORKLOG-STAGING-001` for a full deployed-environment E2E pass
  - created `TASK-CICD-001`, `HANDSHAKE-CICD-001`, and `WORKLOG-CICD-001` for GitHub push-to-deploy automation
  - paused `TASK-OFFCHAIN-010` and `TASK-CHAIN-003` in the active plan while staging validation and CI/CD setup take priority
  - updated `PLAN.md` to point commander threads at the staging E2E and CI/CD tasks
- validated:
  - `.secrets` is not tracked by git
  - `.secrets` is ignored by `.gitignore`
  - `.github/` does not exist yet, so `TASK-CICD-001` has a clean workflow ownership boundary
- blockers:
  - staging E2E may still need a funded non-operator user wallet in addition to the funded operator key already available locally
  - CI/CD worker may still need server SSH host/user/deploy path and the exact remote restart command if those are not inferable from repo files
- next:
  - launch one validation worker for `TASK-STAGING-001`
  - launch one platform worker for `TASK-CICD-001`

### 2026-04-03 20:59 Asia/Shanghai

- read:
  - `HANDSHAKE-CICD-001.md`
  - `WORKLOG-CICD-001.md`
  - `.github/workflows/staging-deploy.yml`
  - `scripts/deploy-staging.sh`
  - `docs/deploy/staging-bsc-testnet.md`
- changed:
  - marked `TASK-CICD-001` as blocked in the active plan because repo-side implementation is done but the first live deploy still needs GitHub secret provisioning and server-side env bootstrap
  - updated `HANDSHAKE-CICD-001.md` with explicit handoff notes for the remaining external setup
  - updated `PLAN.md` next-focus wording to include GitHub Secrets + server `.env.staging` provisioning
- validated:
  - commander review found the workflow/script/runbook implementation present and aligned with the worker handoff
  - no plaintext private key was written into repo docs or workflow files during this review
- blockers:
  - external setup remains:
    - GitHub Secrets `STAGING_SSH_HOST`, `STAGING_SSH_USER`, `STAGING_SSH_PRIVATE_KEY`, `STAGING_DEPLOY_PATH`
    - optional `STAGING_SSH_PORT`, `STAGING_SSH_KNOWN_HOSTS`
    - server-local `deploy/staging/.env.staging`
- next:
  - continue `TASK-STAGING-001`
  - after secrets/env are provisioned, rerun commander review or open a tiny deploy-verification worker to trigger `staging-deploy` and capture first-run evidence

### 2026-04-04 15:12 Asia/Shanghai

- read:
  - `HANDSHAKE-CICD-001.md`
- changed:
  - recorded the verified staging server SSH/user/deploy-path values in `HANDSHAKE-CICD-001.md`
  - recorded a first-run CI/CD bootstrap blocker: `/opt/funnyoption-staging` is still on `main@fa07e19d48dd7a12c5a3533fdb03ccdb27b75dba` and does not yet have `scripts/deploy-staging.sh`, so the server clone needs one manual `git pull` after the workflow commit is pushed
- validated:
  - `funnyoption.xyz` and `admin.funnyoption.xyz` both resolve to `76.13.220.236`
  - `ssh root@76.13.220.236` can reach the server
  - `/opt/funnyoption-staging` is the git checkout path
  - `deploy/staging/.env.staging` exists on the server
  - funnyoption staging containers are running from the `funnyoption-staging-*` compose stack
- blockers:
  - first GitHub Actions deploy will fail until the server checkout contains `scripts/deploy-staging.sh`; perform one manual `git pull` on `/opt/funnyoption-staging` after pushing the CI/CD commit, or manually sync the script once
- next:
  - fill GitHub Secrets with host `76.13.220.236`, user `root`, private key, and deploy path `/opt/funnyoption-staging`
  - push the CI/CD commit and manually pull the server checkout once before triggering the first workflow run

### 2026-04-04 15:37 Asia/Shanghai

- read:
  - `.github/workflows/staging-deploy.yml`
  - `scripts/deploy-staging.sh`
  - `deploy/staging/docker-compose.staging.yml`
  - `deploy/docker/api.Dockerfile`
  - `deploy/docker/web.Dockerfile`
  - `deploy/docker/admin.Dockerfile`
  - `WORKLOG-CICD-001.md`
- changed:
  - marked `TASK-CICD-001` completed after the first staging workflow run succeeded
  - created `TASK-CICD-002`, `HANDSHAKE-CICD-002`, and `WORKLOG-CICD-002` for path-based selective CI/CD
  - updated `PLAN.md` and the active master plan to make `TASK-CICD-002` the next CI/CD lane
- validated:
  - current workflow and remote script still deploy the full stack on every push
  - current Go Dockerfiles use `COPY . .`, so docs/script-only changes can invalidate backend image cache unless unchanged services are skipped or Dockerfile contexts are narrowed carefully
- blockers:
  - selective deploy must preserve a conservative fallback for shared paths such as `go.mod`, `go.sum`, `internal/shared/**`, `proto/**`, `deploy/docker/**`, `deploy/staging/**`, and `scripts/deploy-staging.sh`
- next:
  - launch one platform worker for `TASK-CICD-002`

### 2026-04-04 16:06 Asia/Shanghai

- read:
  - `TASK-CICD-002.md`
  - `HANDSHAKE-CICD-002.md`
  - `WORKLOG-CICD-002.md`
  - `.github/workflows/staging-deploy.yml`
  - `scripts/deploy-staging.sh`
  - `docs/deploy/staging-bsc-testnet.md`
- changed:
  - marked `TASK-CICD-002` as blocked in the active plan because commander review found a remote-script bootstrap ordering bug
  - created `TASK-CICD-003`, `HANDSHAKE-CICD-003`, and `WORKLOG-CICD-003` for a narrow fix
  - updated `PLAN.md` to point the CI/CD lane at `TASK-CICD-003`
- validated:
  - the selective path-to-service map and docs-only no-op behavior are documented and have worker dry-run evidence
  - one rollout blocker remains: the server's old `deploy-staging.sh` parses CLI flags before it syncs the new ref
- blockers:
  - a push that introduces new deploy-script flags can fail if the remote checkout still has an older script
- next:
  - launch one platform worker for `TASK-CICD-003`

### 2026-04-04 16:22 Asia/Shanghai

- read:
  - `TASK-CICD-003.md`
  - `HANDSHAKE-CICD-003.md`
  - `WORKLOG-CICD-003.md`
  - `.github/workflows/staging-deploy.yml`
  - `scripts/deploy-staging.sh`
  - `docs/deploy/staging-bsc-testnet.md`
- changed:
  - marked `TASK-CICD-002` and `TASK-CICD-003` completed in the active plan
  - updated `HANDSHAKE-CICD-002.md` and `HANDSHAKE-CICD-003.md` status fields to `completed`
  - refreshed `PLAN.md` next-focus wording so the CI/CD lane is no longer shown as blocked
- validated:
  - commander review found the remote self-bootstrap issue fixed: the workflow now fetches and checks out the target ref before invoking the deploy script from that ref
  - `bash -n scripts/deploy-staging.sh`
  - `ruby -e "require 'yaml'; YAML.load_file('.github/workflows/staging-deploy.yml')"`
  - `bash scripts/deploy-staging.sh --skip-git-sync --print-plan --diff-base HEAD~1`
  - `git diff --check`
- blockers:
  - no CICD-002/003 repo-code blocker remains at the declared scope
  - residual limitation remains documented: selected Go service images still use broad `COPY . .` contexts, so chosen backend image rebuilds can still lose cache on unrelated repo-file changes even though untouched services are no longer restarted
- next:
  - continue `TASK-STAGING-001`

### 2026-04-04 16:34 Asia/Shanghai

- read:
  - `TASK-STAGING-001.md`
  - `HANDSHAKE-STAGING-001.md`
  - `WORKLOG-STAGING-001.md`
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
- changed:
  - expanded `TASK-STAGING-001` to require a checked-in bounded concurrency script for parallel order placement and matching on staging
  - added `scripts/staging-concurrency-orders.mjs` to the worker ownership set so the script can be implemented without touching product code by default
  - marked `HANDSHAKE-STAGING-001` and the active master plan lane as `active`
- validated:
  - the existing staging worklog already contains one single-flow E2E pass plus three product follow-up findings
  - the new concurrency requirement is scoped to staging validation and explicitly asks for aggregate counters, latency summary, and duplicate-fill / overfill / negative-balance / stale-freeze anomaly evidence
- blockers:
  - no new commander-level blocker; the worker should stop and report if bounded concurrency starts hitting staging rate limits or transient overload thresholds
- next:
  - launch or resume the `TASK-STAGING-001` worker with the new concurrency-script requirement

### 2026-04-04 17:24 Asia/Shanghai

- read:
  - `HANDSHAKE-STAGING-001.md`
  - `WORKLOG-STAGING-001.md`
  - `scripts/staging-concurrency-orders.mjs`
  - `internal/chain/service/listener.go`
  - `internal/api/handler/order_handler.go`
  - `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts`
  - `web/app/portfolio/page.tsx`
  - `web/lib/api.ts`
- changed:
  - marked `TASK-STAGING-001` blocked because the second staging pass still has one environment blocker plus three product correctness regressions
  - created `TASK-CHAIN-004`, `TASK-API-005`, and `TASK-OFFCHAIN-011` with handshakes/worklogs for the three parallel fix lanes
  - updated `PLAN.md` and the active master plan to route next workers to those fix lanes before rerunning `TASK-STAGING-001`
- validated:
  - `node --check scripts/staging-concurrency-orders.mjs`
  - `git diff --check -- scripts/staging-concurrency-orders.mjs`
  - `ssh root@76.13.220.236 'cd /opt/funnyoption-staging && docker compose -f deploy/staging/docker-compose.staging.yml logs --since 4h chain | tail -n 300'`
  - `ssh root@76.13.220.236 "grep -E '^(FUNNYOPTION_CHAIN_START_BLOCK|FUNNYOPTION_CHAIN_RPC_URL)=' /opt/funnyoption-staging/deploy/staging/.env.staging"`
- blockers:
  - staging chain listener is replaying from `99452107` against a pruned public RPC while the blocked deposit tx was mined at `99674293`, so fresh deposits are not credited after chain-service restart
  - duplicate bootstrap is still non-atomic
  - first-liquidity collateral units are still under-debited
  - `/portfolio` still renders default-user collections instead of the connected session user
- next:
  - launch `TASK-CHAIN-004`, `TASK-API-005`, and `TASK-OFFCHAIN-011` in parallel

### 2026-04-04 17:42 Asia/Shanghai

- read:
  - `TASK-CHAIN-004.md`
  - `HANDSHAKE-CHAIN-004.md`
  - `WORKLOG-CHAIN-004.md`
  - `internal/chain/service/listener.go`
  - `internal/chain/service/sql_store.go`
  - `internal/chain/service/listener_test.go`
  - `internal/chain/service/processor_test.go`
  - `migrations/009_chain_listener_cursors.sql`
  - `docs/sql/schema.md`
  - `docs/deploy/staging-bsc-testnet.md`
  - `deploy/staging/docker-compose.staging.yml`
- changed:
  - marked `TASK-CHAIN-004` blocked in the active plan and handshake because commander review found one runbook command issue and the required live staging post-restart deposit proof is still missing
- validated:
  - `go test ./internal/chain/service/...`
  - `git diff --check -- internal/chain/service docs/deploy/staging-bsc-testnet.md docs/sql/schema.md migrations/009_chain_listener_cursors.sql`
  - implementation review confirms the listener now loads/saves `chain_listener_cursors` and can fast-forward past a pruned-history RPC range with an explicit warning log
- blockers:
  - `docs/deploy/staging-bsc-testnet.md` currently shows `docker compose exec postgres psql "$FUNNYOPTION_POSTGRES_DSN"`, but the `postgres` service does not load `.env.staging`, so the DSN is not guaranteed to exist in that shell
  - this thread still needs one deployed staging proof where a fresh post-restart Vault deposit appears in `/api/v1/deposits` and `/api/v1/balances`
- next:
  - ask the `TASK-CHAIN-004` worker to patch the runbook SQL snippets and provide the post-deploy fresh-deposit smoke evidence

### 2026-04-04 17:58 Asia/Shanghai

- read:
  - `TASK-API-005.md`
  - `HANDSHAKE-API-005.md`
  - `WORKLOG-API-005.md`
  - `internal/api/handler/order_handler.go`
  - `internal/api/handler/order_handler_test.go`
  - `internal/api/dto/operator_auth.go`
  - `internal/api/dto/order.go`
  - `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts`
  - `admin/lib/operator-server.ts`
  - `internal/shared/assets/assets.go`
  - `internal/account/service/balance_book.go`
  - `internal/account/service/sql_store.go`
- changed:
  - marked `TASK-API-005` completed in `HANDSHAKE-API-005.md` and the active master plan
  - updated `PLAN.md` next-focus wording so first-liquidity correctness is no longer shown as an open implementation lane
- validated:
  - implementation review found the duplicate bootstrap precheck now happens under the semantic replay lock before collateral/inventory mutation, and the admin route no longer issues a second `/api/v1/orders` call
  - `go test ./internal/api/...`
  - `cd /Users/zhangza/code/funnyoption/admin && npm run build`
  - `git diff --check -- internal/api admin docs/harness/handshakes/HANDSHAKE-API-005.md docs/harness/worklogs/WORKLOG-API-005.md`
- blockers:
  - no API-005 code blocker found at commander review
  - runtime replay on a live stack is still pending because the worker could not run the local lifecycle with the dev stack down; that check should move into the next `TASK-STAGING-001` rerun after `TASK-CHAIN-004` and `TASK-OFFCHAIN-011` are done
- next:
  - continue `TASK-CHAIN-004` and `TASK-OFFCHAIN-011`, then rerun `TASK-STAGING-001`

### 2026-04-04 18:11 Asia/Shanghai

- read:
  - `TASK-OFFCHAIN-011.md`
  - `HANDSHAKE-OFFCHAIN-011.md`
  - `WORKLOG-OFFCHAIN-011.md`
  - `web/app/portfolio/page.tsx`
  - `web/components/portfolio-shell.tsx`
  - `web/lib/api.ts`
- changed:
  - marked `TASK-OFFCHAIN-011` completed in `HANDSHAKE-OFFCHAIN-011.md` and the active master plan
  - updated `PLAN.md` next-focus wording so `/portfolio` connected-user reads are no longer shown as an open implementation lane
- validated:
  - implementation review found `/portfolio` SSR now passes only public markets plus an explicit no-session state, while client-side portfolio reads are keyed by `session.userId` after hydration
  - no external callsites still rely on no-arg `getBalances` / `getOrders` / `getPositions` / `getPayouts` / `getProfile` fallbacks
  - `cd /Users/zhangza/code/funnyoption/web && npm run build`
  - `git diff --check -- web/app/portfolio/page.tsx web/components/portfolio-shell.tsx web/lib/api.ts`
- blockers:
  - no OFFCHAIN-011 code blocker found at commander review
  - real staging browser proof is still pending; `TASK-STAGING-001` should rerun `/portfolio` with a fresh generated session wallet once `TASK-CHAIN-004` restores deposit ingestion
- next:
  - finish `TASK-CHAIN-004`, then rerun `TASK-STAGING-001`

### 2026-04-04 18:27 Asia/Shanghai

- read:
  - `HANDSHAKE-CHAIN-004.md`
  - `WORKLOG-CHAIN-004.md`
  - `docs/deploy/staging-bsc-testnet.md`
  - `PLAN.md`
  - `docs/harness/plans/active/PLAN-2026-04-01-master.md`
  - `.github/workflows/staging-deploy.yml`
- changed:
  - moved `HANDSHAKE-CHAIN-004.md` back to `blocked`
  - updated `PLAN.md` and the active master plan with the remaining release-hygiene blockers
- validated:
  - `go test ./internal/chain/service/...`
  - `ssh root@76.13.220.236 'cd /opt/funnyoption-staging && printf "HEAD=" && git rev-parse --short HEAD && printf "\nSTATUS\n" && git status --short && printf "\nSCRIPT\n" && ls -l scripts/deploy-staging.sh && printf "\nCHAIN_COMPOSE\n" && docker compose --env-file deploy/staging/.env.staging -f deploy/staging/docker-compose.staging.yml ps chain'`
  - server chain container is up and the fresh-deposit smoke from the worker is credible, but the checkout is still `HEAD=fbdcc5f` with tracked edits in `internal/chain/service/listener.go` and `internal/chain/service/sql_store.go` plus untracked `migrations/009_chain_listener_cursors.sql`
  - current workflow behavior in `.github/workflows/staging-deploy.yml` intentionally exits if tracked server-checkout edits exist before `git fetch` and `git checkout`, so the manual patch drift would block the next CI deploy
  - rechecked the recovery snippet shape in `docs/deploy/staging-bsc-testnet.md` and confirmed `psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"` still expands in the host shell, not inside the `postgres` container shell, unless the host shell exports those vars first
- blockers:
  - patch the runbook snippets so container DB vars expand in the container shell, or replace them with the validated literal `psql -U funnyoption -d funnyoption` form
  - push the chain cursor patch into the repo and normalize `/opt/funnyoption-staging` to a clean checkout of that commit before relying on the next Actions deployment
- next:
  - continue the same `TASK-CHAIN-004` worker for this release-hygiene cleanup, then rerun `TASK-STAGING-001`

### 2026-04-04 18:45 Asia/Shanghai

- read:
  - `HANDSHAKE-CHAIN-004.md`
  - `WORKLOG-CHAIN-004.md`
  - `docs/deploy/staging-bsc-testnet.md`
  - `PLAN.md`
  - `docs/harness/plans/active/PLAN-2026-04-01-master.md`
- changed:
  - marked `TASK-CHAIN-004` completed in the active master plan
  - updated `PLAN.md` so `TASK-STAGING-001` is the next worker focus again
- validated:
  - `go test ./internal/chain/service/...`
  - `git diff --check`
  - `ssh root@76.13.220.236 'cd /opt/funnyoption-staging && printf "HEAD=" && git rev-parse --short HEAD && printf "\nSTATUS\n" && git status --short && if ! git diff --quiet --ignore-submodules -- || ! git diff --cached --quiet --ignore-submodules --; then echo "DIRTY_GUARD=fail" >&2; exit 1; fi && printf "DIRTY_GUARD=clean\n" && docker compose --env-file deploy/staging/.env.staging -f deploy/staging/docker-compose.staging.yml ps chain'`
  - `ssh root@76.13.220.236 'cd /opt/funnyoption-staging && docker compose --env-file deploy/staging/.env.staging -f deploy/staging/docker-compose.staging.yml exec -T postgres sh -lc '\''psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT chain_name, network_name, vault_address, next_block, updated_at FROM chain_listener_cursors ORDER BY updated_at DESC LIMIT 5;"'\'''`
  - `ssh root@76.13.220.236 'curl -sS "https://funnyoption.xyz/api/v1/deposits?user_id=1430496&limit=20" && printf "\n" && curl -sS "https://funnyoption.xyz/api/v1/balances?user_id=1430496&limit=20"'`
  - server result: `HEAD=ea71dc8`, empty `git status --short`, `DIRTY_GUARD=clean`, `funnyoption-staging-chain-1` is `Up`, cursor `next_block=99683609`, and the fresh deposit `dep_09caa085db59579b1543683f64ae8238` plus `USDT available=100 frozen=0` are still visible for `user_id=1430496`
- blockers:
  - no release blocker remains for `TASK-CHAIN-004`
- follow-up:
  - one non-blocking runbook issue remains in the DSN fallback path: `source deploy/staging/.env.staging` emits `Testnet: command not found` on the current server env file because at least one value contains an unquoted space, even though `psql "$FUNNYOPTION_POSTGRES_DSN" -c "SELECT 1;"` still succeeds afterward
- next:
  - rerun `TASK-STAGING-001` against staging, including duplicate bootstrap, connected-user `/portfolio`, and bounded concurrent order/matching verification

### 2026-04-04 18:59 Asia/Shanghai

- changed:
  - switched `TASK-STAGING-001` back to `active` in `HANDSHAKE-STAGING-001.md` and the active master plan
  - replaced stale `TASK-STAGING-001` blockers with explicit revalidation targets for `TASK-CHAIN-004`, `TASK-API-005`, `TASK-OFFCHAIN-011`, and the bounded concurrency script
- next:
  - launch one staging validation worker on `TASK-STAGING-001`

### 2026-04-04 19:16 Asia/Shanghai

- read:
  - `HANDSHAKE-STAGING-001.md`
  - `WORKLOG-STAGING-001.md`
  - `scripts/staging-concurrency-orders.mjs`
  - `internal/api/handler/order_handler.go`
  - `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts`
  - `web/app/portfolio/page.tsx`
  - `web/components/portfolio-shell.tsx`
  - `web/lib/api.ts`
- changed:
  - moved `TASK-STAGING-001` back to `blocked` in the handshake and active master plan
  - updated `PLAN.md` to make API/web deploy of the reviewed fixes the next focus instead of another immediate staging rerun
- validated:
  - latest staging worker evidence shows fresh deposits now credit again and bounded concurrency/matching is healthy: `submitted_orders=8`, `success_orders=8`, `matched_trade_count=4`, `duplicate_fill=PASS`, `overfill=PASS`, `negative_balance=PASS`, `stale_freeze=PASS`
  - server staging checkout is still `HEAD=ea71dc8`
  - local workspace still carries reviewed-but-undeployed diffs in:
    - `internal/api/handler/order_handler.go`
    - `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts`
    - `web/app/portfolio/page.tsx`
    - `web/components/portfolio-shell.tsx`
    - `web/lib/api.ts`
  - those local diffs line up exactly with the three remaining staging failures from `WORKLOG-STAGING-001.md`: duplicate bootstrap side effects, raw-quantity first-liquidity collateral debit, and stale `/portfolio` collection reads
- blockers:
  - `TASK-STAGING-001` cannot reach a truthful full PASS until the reviewed `TASK-API-005` and `TASK-OFFCHAIN-011` changes are committed, pushed, and deployed to staging
- next:
  - deploy the reviewed API/web fixset, then rerun `TASK-STAGING-001` as a narrow deployment-verification pass

### 2026-04-04 19:28 Asia/Shanghai

- read:
  - `HANDSHAKE-STAGING-001.md`
  - `WORKLOG-STAGING-001.md`
  - `PLAN.md`
  - `docs/harness/plans/active/PLAN-2026-04-01-master.md`
- changed:
  - marked `TASK-STAGING-001` completed in the handshake and active master plan
  - updated `PLAN.md` next-focus wording to resume the paused local follow-up lanes
- validated:
  - latest staging worklog now contains a deployment-verification-only pass on deployed `125f9cd`
  - staging server check confirms `/opt/funnyoption-staging` reports `HEAD=125f9cd`
  - combined staging evidence now covers:
    - fresh deposit credit restored
    - bounded concurrent submit/match/resolve flow with no duplicate-fill / overfill / negative-balance / stale-freeze anomaly
    - duplicate bootstrap `409` with no inventory/balance side effect
    - first-liquidity collateral debit at `100 * quantity`
    - truthful `/portfolio` no-session and session-user reads
- blockers:
  - none open for `TASK-STAGING-001`
- next:
  - resume `TASK-OFFCHAIN-010` and `TASK-CHAIN-003` in parallel

### 2026-04-04 19:40 Asia/Shanghai

- read:
  - `HANDSHAKE-OFFCHAIN-010.md`
  - `WORKLOG-OFFCHAIN-010.md`
  - `HANDSHAKE-CHAIN-003.md`
  - `WORKLOG-CHAIN-003.md`
  - `cmd/local-lifecycle/main.go`
  - `docs/operations/local-lifecycle-runbook.md`
  - `migrations/010_chain_deposits_tx_hash_width_repair.sql`
  - `docs/operations/local-chain-deposits-schema-repair.md`
- changed:
  - marked `TASK-CHAIN-003` completed in the handshake and active master plan
  - marked `TASK-OFFCHAIN-010` completed in the handshake and active master plan
  - created `TASK-OFFCHAIN-012` plus handshake/worklog as a narrow local lifecycle runner/docs follow-up
  - updated `PLAN.md` next focus to `TASK-OFFCHAIN-012`
- validated:
  - `TASK-CHAIN-003` output is coherent and low-risk:
    - repair migration only widens `chain_deposits.tx_hash` to `VARCHAR(128)`
    - runbook scope is local-only and rollback-safe
    - `go test ./internal/chain/...`
  - `TASK-OFFCHAIN-010` validation proves runtime parity with staging, but `cmd/local-lifecycle` still performs a stale second maker `SELL` after first-liquidity already queued the bootstrap order
- blockers:
  - no blocker remains for `TASK-CHAIN-003`
  - no shared-runtime blocker remains for `TASK-OFFCHAIN-010`, but `TASK-OFFCHAIN-012` is now the follow-up needed to make the one-command local wrapper proof green again
- next:
  - launch one worker on `TASK-OFFCHAIN-012`

### 2026-04-04 20:06 Asia/Shanghai

- read:
  - `HANDSHAKE-OFFCHAIN-012.md`
  - `WORKLOG-OFFCHAIN-012.md`
  - `cmd/local-lifecycle/main.go`
  - `docs/operations/local-lifecycle-runbook.md`
  - `docs/operations/local-offchain-lifecycle.md`
- changed:
  - marked `TASK-OFFCHAIN-012` completed in the handshake and active master plan
  - updated `PLAN.md` next-focus wording because no blocking execution lane remains
- validated:
  - worker removed the stale second maker `SELL` from `cmd/local-lifecycle`
  - `./scripts/local-lifecycle.sh` completed successfully
  - local lifecycle docs now describe the one-shot first-liquidity contract truthfully
  - residual caveat is narrow and non-blocking: persistent `anvil` plus reused local postgres can reuse deterministic deposit evidence across runs unless the local DB is reset
- blockers:
  - no blocker remains for `TASK-OFFCHAIN-012`
- next:
  - no mandatory worker launch; optional future cleanup is the shell-safe DSN docs follow-up

### 2026-04-04 20:18 Asia/Shanghai

- read:
  - `PLAN.md`
  - `docs/harness/plans/active/PLAN-2026-04-01-master.md`
  - `TASK-CICD-003.md`
  - `HANDSHAKE-CICD-003.md`
  - `WORKLOG-CICD-001.md`
  - `WORKLOG-CICD-003.md`
  - `.github/workflows/staging-deploy.yml`
  - `scripts/deploy-staging.sh`
- changed:
  - created `TASK-CICD-004` plus handshake/worklog as an optional platform simplification lane
  - updated `PLAN.md` next-focus wording and the active master plan task table
- validated:
  - current staging deploy already has the right safety boundaries:
    - exact-SHA checkout
    - clean server checkout guard
    - selective deploy planning
  - the change request is about simplifying control flow so GitHub Actions becomes a thinner SSH trigger into one fixed server-side deploy entrypoint
- blockers:
  - none open; the worker should preserve the current safety properties while reducing operator friction
- next:
  - launch one worker on `TASK-CICD-004`

### 2026-04-04 20:24 Asia/Shanghai

- read:
  - `TASK-CICD-004.md`
  - `HANDSHAKE-CICD-004.md`
  - `.github/workflows/staging-deploy.yml`
  - `scripts/deploy-staging.sh`
- changed:
  - fixed the server-side install path, lock path, and default deploy control flow for `TASK-CICD-004`
- validated:
  - the chosen host-side contract keeps the stable trigger script outside the mutable repo checkout:
    - entrypoint `/usr/local/bin/funnyoption-staging-deploy`
    - lock file `/var/lock/funnyoption-staging-deploy.lock`
  - push-driven selective deploy can use `current HEAD -> target SHA` as the diff base without regressing to implicit `git pull main`
- blockers:
  - none at commander/planning level
- next:
  - launch one worker on `TASK-CICD-004` with the fixed path contract

### 2026-04-04 21:05 Asia/Shanghai

- read:
  - `TASK-CICD-004.md`
  - `HANDSHAKE-CICD-004.md`
  - `WORKLOG-CICD-004.md`
  - `deploy/staging/server-deploy-entrypoint.sh`
  - `.github/workflows/staging-deploy.yml`
- changed:
  - marked `TASK-CICD-004` blocked in the active master plan and handshake after commander review
  - updated `PLAN.md` next-focus wording to keep the follow-up on the same task instead of opening a new one
- validated:
  - push-driven exact-SHA deploys are still sound because the workflow passes `github.sha`
  - manual symbolic refs are not yet safe enough:
    - the host entrypoint resolves local `${target_ref}` / `${branch_ref}` before `origin/<ref>`
    - a stale local `main` branch in `/opt/funnyoption-staging` can therefore shadow the freshly fetched remote branch tip
- blockers:
  - `TASK-CICD-004` cannot close until symbolic branch refs prefer remote-tracking refs after fetch
- next:
  - continue `TASK-CICD-004` as a narrow ref-resolution fix, then rerun one exact-SHA proof and one symbolic-ref proof

### 2026-04-04 21:14 Asia/Shanghai

- read:
  - `deploy/staging/server-deploy-entrypoint.sh`
  - `HANDSHAKE-CICD-004.md`
  - `WORKLOG-CICD-004.md`
  - `PLAN.md`
  - `docs/harness/plans/active/PLAN-2026-04-01-master.md`
- changed:
  - marked `TASK-CICD-004` completed in the active master plan
  - updated `PLAN.md` next-focus wording because no blocking execution lane remains
- validated:
  - `deploy/staging/server-deploy-entrypoint.sh` now preserves exact-SHA deploys while preferring freshly fetched remote-tracking refs for symbolic branch names
  - worker evidence includes:
    - symbolic-ref proof with stale local `main` shadowing avoided
    - exact-SHA proof still landing on the requested commit
    - docs-only selective no-op behavior preserved
    - host lock serialization preserved
- blockers:
  - no blocker remains for `TASK-CICD-004`
- next:
  - no mandatory worker launch; remaining follow-ups are optional operational polish only

### 2026-04-04 21:10 CST

- read:
  - GitHub Actions failure output for missing `/usr/local/bin/funnyoption-staging-deploy`
  - `web/lib/session-client.ts`
  - `web/components/trading-session-provider.tsx`
  - `docs/architecture/direct-deposit-session-key.md`
  - `admin/components/market-studio.tsx`
  - `internal/settlement/service/processor.go`
- changed:
  - completed the one-time host install step for the fixed staging deploy entrypoint on `76.13.220.236`
  - created `TASK-OFFCHAIN-013` plus handshake/worklog for wallet-signed session login / restore UX optimization
  - created `TASK-CHAIN-005` plus handshake/worklog for design-first oracle-settled crypto markets
  - updated `PLAN.md` and the active master plan with the next two product lanes
- validated:
  - server rollout fix:
    - `/opt/funnyoption-staging` was manually advanced to `d7a79c177beec77e0a43f95ca69adc3242905ff4`
    - `/usr/local/bin/funnyoption-staging-deploy` is now installed
    - the host entrypoint completed successfully for the pushed commit
  - next product lanes are now explicit in repo memory:
    - wallet session UX can start immediately as an implementation task
    - oracle auto-settlement remains design-first before runtime implementation
- blockers:
  - no blocker remains for the staging deploy lane
  - no new blocker yet for the two new product lanes
- next:
  - launch one worker on `TASK-OFFCHAIN-013`
  - launch one design worker on `TASK-CHAIN-005`

### 2026-04-04 21:14 CST

- read:
  - user-stated Stark-style first-login / deposit flow
  - `docs/architecture/direct-deposit-session-key.md`
  - `TASK-OFFCHAIN-013.md`
  - `HANDSHAKE-OFFCHAIN-013.md`
- changed:
  - created `TASK-OFFCHAIN-014` plus handshake/worklog as a design-first auth lane
  - moved `TASK-OFFCHAIN-013` to blocked because it assumed the current
    session-key trust model would remain unchanged
  - updated `PLAN.md` and the active master plan so the new auth direction is
    explicit in repo memory before any worker starts coding
- validated:
  - the user-requested flow is materially different from the current repo
    baseline:
    - current repo: wallet-authorized browser-generated session key
    - requested direction: one MetaMask signature, then one browser-local
      Stark-style trading key for later order signing
  - direct-vault deposit flow still aligns with the current chain architecture,
    so only the auth/trading-key contract needs a new design lane first
- blockers:
  - do not start the old `TASK-OFFCHAIN-013` implementation lane until
    `TASK-OFFCHAIN-014` closes the auth contract
- next:
  - launch one design worker on `TASK-OFFCHAIN-014`
  - keep `TASK-CHAIN-005` running in parallel as the oracle-settlement design lane

### 2026-04-04 21:26 CST

- read:
  - `foundry.toml`
  - `contracts/src/FunnyVault.sol`
  - `contracts/src/MockUSDT.sol`
  - `TASK-CHAIN-005.md`
  - `HANDSHAKE-CHAIN-005.md`
- changed:
  - clarified in repo memory that any oracle-lane contract work should stay on
    the existing Foundry toolchain instead of introducing a second Solidity
    framework
  - updated `PLAN.md`, the active master plan, and the `TASK-CHAIN-005`
    handshake/task pair accordingly
- validated:
  - the repo already has a minimal but real Foundry baseline:
    - `foundry.toml` config exists
    - contract layout is `contracts/src`, `contracts/test`, `contracts/script`
    - current Solidity surface is still narrow
- blockers:
  - none
- next:
  - launch `TASK-CHAIN-005` with Foundry explicitly fixed as the contract
    toolchain boundary

### 2026-04-04 21:52 CST

- read:
  - `HANDSHAKE-OFFCHAIN-014.md`
  - `WORKLOG-OFFCHAIN-014.md`
  - `HANDSHAKE-CHAIN-005.md`
  - `WORKLOG-CHAIN-005.md`
  - `docs/architecture/direct-deposit-session-key.md`
  - `docs/architecture/oracle-settled-crypto-markets.md`
  - `internal/settlement/service/sql_store.go`
- changed:
  - accepted `TASK-OFFCHAIN-014` and `TASK-CHAIN-005` as completed design tasks
  - updated `PLAN.md` and the active master plan with their completed status
  - created follow-up implementation lanes:
    - `TASK-OFFCHAIN-015`
    - `TASK-CHAIN-006`
- validated:
  - auth design result:
    - reject deterministic signature-derived trading keys
    - accept wallet-authorized browser-local trading keys
    - first implementation slice is challenge issuance plus `EIP-712`
      registration
  - oracle design result:
    - use `markets.metadata.resolution`
    - use one dedicated oracle worker
    - reuse `market_resolutions` for first cut
  - commander review found one required runtime truthfulness guard for the
    oracle lane:
    - manual fallback must overwrite stale oracle ownership fields in
      `market_resolutions` when the operator wins from earlier error states
- blockers:
  - `TASK-OFFCHAIN-013` remains blocked because its old session-key UX wording
    no longer matches the new trading-key baseline
- next:
  - launch `TASK-OFFCHAIN-015`
  - launch `TASK-CHAIN-006`

### 2026-04-05 00:10 CST

- read:
  - `HANDSHAKE-OFFCHAIN-015.md`
  - `WORKLOG-OFFCHAIN-015.md`
  - `HANDSHAKE-CHAIN-006.md`
  - `WORKLOG-CHAIN-006.md`
  - `internal/api/routes_auth.go`
  - `cmd/local-lifecycle/main.go`
  - `scripts/staging-concurrency-orders.mjs`
  - `internal/oracle/service/worker.go`
  - `internal/settlement/service/processor.go`
  - `internal/account/service/event_processor.go`
- changed:
  - marked `TASK-OFFCHAIN-015` and `TASK-CHAIN-006` back to blocked after
    commander review
  - updated `PLAN.md` and the active master plan so the blockers are explicit in
    repo memory
- validated:
  - OFFCHAIN-015 blocker:
    - the new V2 auth slice removed `POST /api/v1/sessions`, but existing repo
      lifecycle / staging proof clients still call that route
  - OFFCHAIN-015 residual risk:
    - `wallet_sessions` compatibility storage still collapses active-key scope
      to `wallet + chain` because `vault_address` is not durably stored there
  - CHAIN-006 blocker:
    - the oracle worker republishes the same resolved `market.event` while the
      row is still `OBSERVED`
    - downstream settlement/account side effects are not idempotent enough for
      duplicate emits
- blockers:
  - keep both implementation workers on their current task ids until those
    review blockers are closed
- next:
  - continue `TASK-OFFCHAIN-015`
  - continue `TASK-CHAIN-006`

### 2026-04-04 23:22 CST

- read:
  - `HANDSHAKE-OFFCHAIN-015.md`
  - `WORKLOG-OFFCHAIN-015.md`
  - `HANDSHAKE-CHAIN-006.md`
  - `WORKLOG-CHAIN-006.md`
  - `internal/api/routes_auth.go`
  - `internal/api/router_test.go`
  - `internal/oracle/service/worker.go`
  - `internal/oracle/service/worker_test.go`
  - `docs/architecture/direct-deposit-session-key.md`
  - `docs/sql/schema.md`
- changed:
  - accepted the follow-up fixes and marked `TASK-OFFCHAIN-015` and
    `TASK-CHAIN-006` completed in `PLAN.md` and the active master plan
  - refreshed commander memory so those lanes are no longer shown as
    review-blocked
- validated:
  - `go test ./internal/oracle/service ./internal/settlement/service ./internal/account/service`
  - `go test ./internal/api/... ./cmd/local-lifecycle`
  - `node --check scripts/staging-concurrency-orders.mjs`
  - `TASK-CHAIN-006` closure:
    - duplicate polling of an already-recorded oracle `OBSERVED` outcome now
      skips publish instead of replaying the same resolved `market.event`
    - manual resolve conflict guard and operator-owned overwrite semantics stay
      intact
  - `TASK-OFFCHAIN-015` closure:
    - `POST /api/v1/sessions` is restored as a deprecated compatibility route
      for repo proof tooling
    - canonical browser auth remains `POST /api/v1/trading-keys/challenge` plus
      `POST /api/v1/trading-keys`
    - truthful restore remains active, and the remaining durable-scope caveat
      is explicitly documented as a single-vault-per-environment assumption
- blockers:
  - no release blocker remains in either task slice
- next:
  - `TASK-OFFCHAIN-013` can resume against the landed V2 trading-key baseline
  - a later oracle follow-up may add an explicit dispatch marker / retry
    contract for the narrower case of publish failure after writing `OBSERVED`

### 2026-04-05 00:08 CST

- read:
  - `HANDSHAKE-OFFCHAIN-013.md`
  - `WORKLOG-OFFCHAIN-013.md`
  - `web/lib/session-client.ts`
  - `web/components/trading-session-provider.tsx`
  - `web/components/session-console.tsx`
  - `web/components/site-header.tsx`
- changed:
  - accepted `TASK-OFFCHAIN-013` as completed
  - updated `PLAN.md` and the active master plan so this lane is no longer
    shown as resumable / blocked
- validated:
  - `cd web && npm run build`
  - commander review confirmed the browser canonical flow still uses
    `POST /api/v1/trading-keys/challenge` +
    `POST /api/v1/trading-keys`
  - restore now reconciles before prompting a new wallet signature, and UI
    state honestly distinguishes restore-in-progress vs reauthorization-needed
- blockers:
  - no blocker in this task slice
- next:
  - if product wants to remove the remaining single-vault-per-environment
    assumption, a later auth/schema task should add durable `vault_address` to
    the server-side trading-key carrier

### 2026-04-05 00:31 CST

- read:
  - `HANDSHAKE-OFFCHAIN-013.md`
  - `WORKLOG-OFFCHAIN-013.md`
  - `HANDSHAKE-OFFCHAIN-015.md`
  - `WORKLOG-OFFCHAIN-015.md`
  - `HANDSHAKE-CHAIN-006.md`
  - `WORKLOG-CHAIN-006.md`
- changed:
  - created two narrow follow-up tasks so the remaining auth and oracle
    tradeoffs are recorded in repo memory instead of chat only:
    - `TASK-OFFCHAIN-016`
    - `TASK-CHAIN-007`
  - added matching handshake / worklog files
  - updated `PLAN.md` and the active master plan with their pending status and
    scope
- validated:
  - `TASK-OFFCHAIN-016` isolates the remaining durable `vault_address`
    truthfulness gap without reopening the wider V2 auth design
  - `TASK-CHAIN-007` isolates the remaining oracle dispatch retry gap without
    reopening duplicate-emit or multi-provider scope
- blockers:
  - none; these are queued follow-up lanes, not release blockers
- next:
  - launch one worker on `TASK-OFFCHAIN-016`
  - optionally launch a second worker on `TASK-CHAIN-007` in parallel

### 2026-04-05 01:02 CST

- read:
  - `HANDSHAKE-OFFCHAIN-016.md`
  - `WORKLOG-OFFCHAIN-016.md`
  - `HANDSHAKE-CHAIN-007.md`
  - `WORKLOG-CHAIN-007.md`
  - `internal/api/handler/sql_store.go`
  - `internal/api/handler/sql_store_scope_test.go`
  - `migrations/003_wallet_sessions_and_deposits.sql`
  - `migrations/012_wallet_sessions_vault_scope.sql`
  - `internal/oracle/service/worker.go`
  - `internal/oracle/service/worker_test.go`
  - `internal/settlement/service/processor.go`
- changed:
  - accepted `TASK-CHAIN-007` as completed
  - moved `TASK-OFFCHAIN-016` back to blocked in the handshake, `PLAN.md`, and
    active master plan after commander review
- validated:
  - `OFFCHAIN-016`:
    - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/api/handler -run TestListSessionsPassesVaultFilter`
    - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/api/...`
    - `zsh -lc 'source .env.local; GOCACHE=/tmp/funnyoption-gocache go test ./internal/api/handler -run TestSQLStoreRegisterTradingKeyScopesByVault'`
    - `cd web && npm run build`
    - `zsh -lc 'source .env.local; psql "$FUNNYOPTION_POSTGRES_DSN" -v ON_ERROR_STOP=1 -c "BEGIN" -f migrations/012_wallet_sessions_vault_scope.sql -c "ROLLBACK"'`
    - commander also confirmed the remaining gap with a temp SQL probe:
      - the legacy `UNIQUE (wallet_address, session_public_key)` rule still
        rejects reusing the same trading public key across two vault scopes
  - `CHAIN-007`:
    - `go test ./internal/oracle/service ./internal/settlement/service`
    - `go test ./cmd/oracle ./internal/account/service`
    - dispatch checkpoint behavior and repeated-poll safety match the worker
      summary
- blockers:
  - `TASK-OFFCHAIN-016` still needs one narrow schema / uniqueness follow-up
    before the server contract is fully truthful to `wallet + chain + vault`
- next:
  - continue `TASK-OFFCHAIN-016`

### 2026-04-05 01:12 CST

- read:
  - `HANDSHAKE-OFFCHAIN-016.md`
  - `WORKLOG-OFFCHAIN-016.md`
  - `migrations/013_wallet_sessions_vault_key_uniqueness.sql`
  - `migrations/003_wallet_sessions_and_deposits.sql`
  - `internal/api/handler/sql_store.go`
  - `internal/api/handler/sql_store_scope_test.go`
  - `docs/sql/schema.md`
  - `docs/architecture/direct-deposit-session-key.md`
- changed:
  - accepted `TASK-OFFCHAIN-016` as completed
  - updated `PLAN.md`, the active master plan, and the handshake so this lane
    is no longer shown as review-blocked
  - refreshed plan memory to record the remaining truth as a compatibility
    tradeoff, not a correctness blocker
- validated:
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/api/handler -run TestListSessionsPassesVaultFilter -count=1`
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/api/...`
  - `zsh -lc 'set -a; source .env.local; set +a; GOCACHE=/tmp/funnyoption-gocache go test ./internal/api/handler -run TestSQLStoreRegisterTradingKeyScopesByVaultEvenWithSamePublicKey -count=1'`
  - `zsh -lc 'set -a; source .env.local; set +a; psql "$FUNNYOPTION_POSTGRES_DSN" -v ON_ERROR_STOP=1 -f migrations/013_wallet_sessions_vault_key_uniqueness.sql'`
  - commander confirmed on the real database that `wallet_sessions` now keeps
    only:
    - `wallet_sessions_wallet_chain_vault_public_key_key`
    - `UNIQUE (wallet_address, chain_id, vault_address, session_public_key)`
- blockers:
  - no release blocker remains in this task slice
- next:
  - any later auth cleanup should focus on retiring deprecated
    `/api/v1/sessions` only after repo proof tooling migrates off the
    blank-vault compatibility contract

### 2026-04-05 02:02 CST

- read:
  - `WORKLOG-HARNESS-002.md`
  - `cmd/local-lifecycle/main.go`
  - `cmd/local-lifecycle/trading_key_oracle_flow.go`
  - `cmd/local-lifecycle/trading_key_oracle_flow_test.go`
  - `scripts/local-full-flow.sh`
  - `docs/operations/local-full-flow-acceptance.md`
- changed:
  - accepted the new local `trading-key-oracle` full-flow harness as landed
    proof infrastructure
  - recorded one remaining harness-truthfulness caveat from commander rerun:
    deposit credit can still false-pass on reused local postgres because the
    flow only matches a credited row by deterministic tx hash instead of
    proving a fresh balance delta or fresh row boundary
- validated:
  - `bash -n scripts/local-lifecycle.sh scripts/local-full-flow.sh`
  - `go test ./cmd/local-lifecycle`
  - reran in one persistent PTY session:
    - `./scripts/dev-up.sh`
    - `./scripts/local-full-flow.sh`
  - independent readback after the rerun:
    - `curl -sS 'http://127.0.0.1:8080/api/v1/sessions?wallet_address=0x1532d37232c783c531bf0ce9860cb15f5f68aeb3&vault_address=0xe7f1725e7734ce288f8367e1bb143e90bb3f0512&status=ACTIVE&limit=20'`
    - `curl -sS 'http://127.0.0.1:8080/api/v1/markets/1775325169450'`
    - `curl -sS 'http://127.0.0.1:8080/api/v1/payouts?user_id=1001&market_id=1775325169450&limit=20'`
    - `psql 'postgres://funnyoption:funnyoption@127.0.0.1:5432/funnyoption?sslmode=disable' -c "SELECT market_id, status, resolved_outcome, resolver_type, resolver_ref FROM market_resolutions WHERE market_id = 1775325169450;"`
  - rerun result:
    - trading-key auth, truthful restore, oracle market creation, matching,
      oracle auto settlement, and payout/readback all reproduced as PASS
    - resolution row read back as `RESOLVED / YES / ORACLE_PRICE` with
      `resolver_ref=oracle_price:BINANCE:BTCUSDT:1775325189`
    - payout read back as
      `evt_settlement_1775325169450_1001_YES payout_amount=4000`
  - residual truthfulness caveat:
    - the rerun summary still showed `initial_usdt == post_deposit_usdt`
      while step 3 reported PASS
    - on persistent anvil plus reused local postgres, the deterministic
      deposit tx hash / deposit row can be reused across runs unless the
      harness proves freshness explicitly
- blockers:
  - no product-runtime blocker
  - one local-harness P2 follow-up remains if we want the deposit step to prove
    fresh credit rather than reuse prior evidence
- next:
  - if we want this harness to become the strongest local acceptance gate,
    add a narrow follow-up to prove deposit freshness explicitly

### 2026-04-05 02:18 CST

- read:
  - `scripts/staging-concurrency-orders.mjs`
  - `docs/operations/core-business-test-flow.md`
  - `docs/harness/worklogs/WORKLOG-STAGING-001.md`
- changed:
  - pushed integrated auth/oracle/full-flow branch to `origin/main` as
    `c9ad5e6 Land trading-key auth, oracle settlement, and local full-flow harness`
  - watched GitHub Actions run
    `https://github.com/alan1-666/funnyoption/actions/runs/23984469462`
    complete `success`
  - confirmed deployed staging API health:
    - `GET https://funnyoption.xyz/healthz => {"env":"staging","service":"api","status":"ok"}`
  - confirmed canonical V2 auth route is live on deployed staging:
    - `POST https://funnyoption.xyz/api/v1/trading-keys/challenge`
      with `chain_id=97` and vault
      `0x7665d943c62268d27ffcbed29c6a8281f7364534` returned `201`
  - recorded one staging-harness false-negative:
    - `node scripts/staging-concurrency-orders.mjs --users 2 --seller-users 1 --orders-per-user 1 --concurrency 1 --poll-timeout-ms 180000 --poll-interval-ms 3000`
      exited `FAIL`, but the failure is in the harness wait condition rather
      than the deployed product path
- validated:
  - pre-push broad checks:
    - `go test ./cmd/local-lifecycle ./cmd/oracle ./internal/shared/auth ./internal/api/... ./internal/oracle/service ./internal/settlement/service ./internal/account/service ./internal/chain/service`
    - `cd web && npm run build`
    - `cd admin && npm run build`
    - `git diff --check`
  - staging deploy:
    - pushed `main -> origin/main`
    - workflow run `23984469462` finished `success`
  - staging readback after the mini E2E script failure:
    - `GET /api/v1/markets/1775325910776` showed `status=OPEN`,
      `active_order_count=1`
    - `GET /api/v1/orders?user_id=1002&market_id=1775325910776&limit=20`
      showed bootstrap order
      `ord_bootstrap_9804827a5c26d5dafe3e3e8d31d923cd status=NEW`
    - `GET /api/v1/positions?user_id=1002&market_id=1775325910776&limit=20`
      showed maker YES/NO inventory both present at quantity `1`
    - `GET /api/v1/balances?user_id=1002` showed `USDT available=2168`
- findings:
  - the current staging concurrency harness calls
    `GET /api/v1/balances?user_id=<maker>&limit=20` during the
    `wait maker bootstrap order and inventory` step, then searches only that
    truncated page for asset `USDT`
  - on current staging, maker `user_id=1002` already has more than 20 balance
    rows, so `USDT` is paged out even though the bootstrap order and inventory
    are already live
  - this makes the harness report
    `wait maker bootstrap order and inventory timeout after 180000ms; last=null`
    while the deployed product state is healthy
- blockers:
  - no staging deployment blocker
  - one harness-only follow-up remains if we want this script to be a truthful
    staging gate under high-balance-history users
- next:
  - keep manual browser-wallet verification as the remaining human-in-the-loop
    step for staging
  - if we want automated staging E2E to go green again, narrow-fix the harness
    balance lookup so it does not false-fail on paginated users

### 2026-04-05 02:24 CST

- read:
  - staging readbacks only
- changed:
  - confirmed one real-browser manual wallet authorization on deployed staging
    for wallet `0xc421d5ff322e4213a913ec257d6b4458af4255c6`
- validated:
  - `GET https://funnyoption.xyz/api/v1/sessions?wallet_address=0xc421d5ff322e4213a913ec257d6b4458af4255c6&vault_address=0x7665d943c62268d27ffcbed29c6a8281f7364534&status=ACTIVE&limit=20`
  - `GET https://funnyoption.xyz/api/v1/sessions?wallet_address=0xc421d5ff322e4213a913ec257d6b4458af4255c6&limit=20`
  - `GET https://funnyoption.xyz/api/v1/profile?wallet_address=0xc421d5ff322e4213a913ec257d6b4458af4255c6`
  - canonical V2 auth row observed:
    - `session_id=tk_128797dea6ba55823159fc7ec1200865`
    - `user_id=1001`
    - `scope=TRADE`
    - `chain_id=97`
    - `vault_address=0x7665d943c62268d27ffcbed29c6a8281f7364534`
    - `status=ACTIVE`
    - `issued_at=1775326136201`
    - `last_order_nonce=1`
  - profile readback stayed truthful for the same wallet:
    - `user_id=1001`
    - `updated_at=1775326136`
- blockers:
  - none for this manual authorization proof
- next:
  - manual follow-up can now focus on refresh-restore and one real browser
    order placement, because the wallet authorization itself is confirmed

### 2026-04-05 02:29 CST

- read:
  - staging auth/balance readbacks after manual browser refresh
- changed:
  - recorded one real-browser refresh observation:
    - refresh reopened the wallet-provider chooser
      (`MetaMask` / `Phantom`)
    - no new signing prompt appeared
- validated:
  - `GET https://funnyoption.xyz/api/v1/sessions?wallet_address=0xc421d5ff322e4213a913ec257d6b4458af4255c6&vault_address=0x7665d943c62268d27ffcbed29c6a8281f7364534&status=ACTIVE&limit=20`
    still returned the same canonical row:
    - `session_id=tk_128797dea6ba55823159fc7ec1200865`
    - `issued_at=1775326136201`
    - `last_order_nonce=1`
  - `GET https://funnyoption.xyz/api/v1/balances?user_id=1001&limit=50`
    returned:
    - `USDT available=9590`
    - `USDT frozen=500`
- findings:
  - refresh did not create a new trading-key session and did not require a new
    authorization signature
  - the remaining UX gap is narrower:
    wallet-provider reconnect still reopens the provider chooser on refresh,
    even though the V2 trading-key restore itself is preserved
  - server-side balance is present; if the browser looked like “no balance”
    before reconnect completed, that is a frontend/provider restore experience
    issue rather than missing funds on the backend
- blockers:
  - no backend auth blocker
  - one frontend wallet reconnect UX follow-up remains if we want refresh to be
    fully quiet
- next:
  - treat “no repeated signature” as PASS
  - treat “provider chooser still opens on refresh” as a narrower UX polish
    follow-up rather than an auth-runtime regression

### 2026-04-05 02:35 CST

- read:
  - staging API readbacks for wallet `0xc421d5ff322e4213a913ec257d6b4458af4255c6`
- changed:
  - recorded one concrete staging read-model issue behind the browser “no
    balance” symptom
- validated:
  - `GET https://funnyoption.xyz/api/v1/profile?user_id=1001`
    returned the expected wallet/profile row
  - `GET https://funnyoption.xyz/api/v1/balances?user_id=1001&limit=10`
    returned only ten `POSITION:*` assets and no `USDT`
  - `GET https://funnyoption.xyz/api/v1/balances?user_id=1001&limit=50`
    returned the same ten position assets plus:
    - `asset=USDT`
    - `available=9590`
    - `frozen=500`
  - `GET https://funnyoption.xyz/api/v1/positions?user_id=1001&limit=20`
    returned ten position rows, matching the assets that fill the first
    `balances?limit=10` page
- findings:
  - the test-environment API does return the user’s USDT balance, but not on
    the first `balances?limit=10` page for this wallet
  - current symptom is pagination/read-shape, not missing funds:
    the first ten balance rows are all `POSITION:*`, and `USDT` appears only
    once the page size exceeds ten
- blockers:
  - no backend balance-loss blocker
  - one read-model/frontend query-shape follow-up remains if we want the main
    wallet summary to reliably show collateral balances for position-heavy users
- next:
  - treat this as a staging read-path issue: either request a larger balance
    page or prioritize collateral assets like `USDT` in the balance response

### 2026-04-05 03:01 CST

- read:
  - `TASK-OFFCHAIN-017.md`
  - `HANDSHAKE-OFFCHAIN-017.md`
  - `WORKLOG-OFFCHAIN-017.md`
  - `web/lib/session-client.ts`
  - `web/components/trading-session-provider.tsx`
  - `web/lib/api.ts`
  - `web/components/shell-top-bar.tsx`
  - `web/components/portfolio-shell.tsx`
  - `web/components/portfolio-shell.module.css`
- changed:
  - commander accepted `TASK-OFFCHAIN-017` as completed and synchronized plan
    state after re-review
- validated:
  - `cd web && npm run build`
  - reviewed the landed frontend diff against the task acceptance criteria
  - worker browser proof recorded:
    - silent refresh restore with `walletCalls=[]`
    - `balances?limit=10` then fallback `balances?limit=200`
    - visible `9,590 USDT`
    - visible copy-success state
    - QR dialog screenshot artifact:
      `/Users/zhangza/code/funnyoption/output/playwright/task-offchain-017-portfolio.png`
- findings:
  - no new P0/P1 found in the narrow wallet/portfolio slice
  - this closure is consistent with the staged browser observations:
    - no repeated signature prompt
    - no new server-side trading-key row created on refresh
    - balance truth restored for the position-heavy staging account
- blockers:
  - no release blocker remains in this slice
- next:
  - any later polish can stay narrow:
    - reduce provider chooser friction further if wallet SDK behavior allows
    - consider backend collateral-asset prioritization if a user can exceed the
      current frontend fallback window of `limit=200`

### 2026-04-05 03:14 CST

- read:
  - `web/components/portfolio-shell.tsx`
  - `web/components/home-market-board.module.css`
  - `web/components/home-market-card.module.css`
- changed:
  - applied one user-requested post-verification micro-polish set:
    - removed the personal-page CTA copy `交易已开启` by hiding the already-on
      primary action once trading is active
    - removed the overview sentence
      `当前展示 user #<id> 的账户数据。`
    - aligned the home market board width to the same `1380px` rail used by the
      portfolio page
    - reduced home market-card density slightly so the narrower rail still
      feels balanced
- validated:
  - `cd web && npm run build`
- blockers:
  - none
- next:
  - push the micro-polish set to staging if the user wants to inspect the new
    homepage density and quieter portfolio copy on the deployed environment
