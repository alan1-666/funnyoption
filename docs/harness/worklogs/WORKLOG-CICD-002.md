# WORKLOG-CICD-002

### 2026-04-04 15:37 Asia/Shanghai

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `.github/workflows/staging-deploy.yml`
  - `scripts/deploy-staging.sh`
  - `deploy/staging/docker-compose.staging.yml`
  - `WORKLOG-CICD-001.md`
- changed:
  - created a CI/CD optimization task and handshake for selective validation/build/deploy by changed path
  - marked `TASK-CICD-001` complete and promoted `TASK-CICD-002` as the next platform lane
- validated:
  - current workflow and deploy script are full-stack on every push
  - one concrete optimization boundary is available: derive a service subset from `git diff`, pass that subset to the remote script, and no-op docs-only pushes
- blockers:
  - path-to-service mapping must stay conservative for shared Go/frontend paths so we do not accidentally skip a required rebuild
- next:
  - launch a worker against `TASK-CICD-002`

### 2026-04-04 16:01 Asia/Shanghai

- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CICD-001.md`
  - `docs/harness/tasks/TASK-CICD-002.md`
  - `docs/harness/handshakes/HANDSHAKE-CICD-002.md`
  - `docs/harness/worklogs/WORKLOG-CICD-001.md`
  - `docs/harness/worklogs/WORKLOG-CICD-002.md`
  - `.github/workflows/staging-deploy.yml`
  - `scripts/deploy-staging.sh`
  - `deploy/staging/docker-compose.staging.yml`
  - `deploy/docker/**`
  - `docs/deploy/staging-bsc-testnet.md`
  - Go package imports from `cmd/*`
- changed:
  - `.github/workflows/staging-deploy.yml`
  - `scripts/deploy-staging.sh`
  - `docs/deploy/staging-bsc-testnet.md`
  - `docs/harness/handshakes/HANDSHAKE-CICD-002.md`
  - `docs/harness/worklogs/WORKLOG-CICD-002.md`
- validated:
  - `bash -n scripts/deploy-staging.sh`
  - `ruby -e "require 'yaml'; YAML.load_file('.github/workflows/staging-deploy.yml')"`
  - `git diff --check`
  - dry-run docs-only probe:
    - changed `docs/harness/worklogs/WORKLOG-CICD-002.md`
    - got `deploy_services=`, `skip_deploy=1`, `run_migrations=0`
  - dry-run single-service probe:
    - changed `cmd/api/main.go`
    - got `deploy_services=api`, `skip_deploy=0`, `run_migrations=0`, `validate_go=1`
  - dry-run shared-backend probe:
    - changed `internal/shared/logger/logger.go`
    - got `deploy_services=account,matching,ledger,settlement,chain,api,ws`, `validate_go=1`
  - dry-run migration probe:
    - changed `migrations/008_user_profiles.sql`
    - got `deploy_services=account,matching,ledger,settlement,chain,api,ws`, `run_migrations=1`, `validate_go=0`
- blockers:
  - no code blockers
  - residual limitation: Go service Dockerfiles still use `COPY . .`, so a
    selected backend image can still lose build cache on unrelated repo-file
    changes even though untouched services are no longer rebuilt or restarted
- next:
  - hand back the final map, fallback policy, validation commands, dry-run
    evidence, and residual limitations to commander
