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
