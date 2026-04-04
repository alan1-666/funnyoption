# HANDSHAKE-CICD-001

## Task

- [TASK-CICD-001.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-001.md)

## Thread owner

- implementation worker for platform/deployment automation

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/deploy/staging-bsc-testnet.md`
- `configs/staging/funnyoption.env.example`
- this handshake
- `WORKLOG-CICD-001.md`

## Files in scope

- `.github/workflows/**`
- `scripts/**` for narrowly scoped deploy helpers
- `docs/deploy/**`
- `docs/harness/worklogs/WORKLOG-CICD-001.md`
- this handshake if status/blockers need to be updated

## Inputs from other threads

- current deployed domains are:
  - `https://funnyoption.xyz/`
  - `https://admin.funnyoption.xyz/`
- no `.github/` workflow directory exists yet, so this is likely a greenfield CI/CD implementation
- a funded BSC testnet operator key exists locally, but it must be injected through GitHub Secrets or a server-only env file; do not commit or echo the plaintext key
- `.secrets` is git-ignored and should stay that way

## Outputs back to commander

- changed files
- required GitHub Secrets names and setup instructions
- deployment command path and rollback/manual recovery notes
- exact blockers if server SSH host/user/deploy-path or runtime manager details are missing

## Blockers

- do not modify `.secrets`
- do not print private keys or secret-bearing env values
- do not touch files owned by `TASK-STAGING-001`
- GitHub push-to-deploy is implemented, but the workflow still requires these
  runtime inputs to be configured outside the repo before the first successful
  deploy:
  - `STAGING_SSH_HOST`
  - `STAGING_SSH_USER`
  - `STAGING_SSH_PRIVATE_KEY`
  - `STAGING_DEPLOY_PATH`
  - server-local `deploy/staging/.env.staging`

## Status

- blocked

## Handoff notes

- `.github/workflows/staging-deploy.yml`, `scripts/deploy-staging.sh`, and the staging deploy runbook are implemented.
- first live GitHub push-to-deploy is blocked only by external setup outside the repo:
  - configure `STAGING_SSH_HOST`, `STAGING_SSH_USER`, `STAGING_SSH_PRIVATE_KEY`, and `STAGING_DEPLOY_PATH`
  - optionally configure `STAGING_SSH_PORT` and `STAGING_SSH_KNOWN_HOSTS`
  - create the server-local `deploy/staging/.env.staging`
- commander verified the current staging server values:
  - `STAGING_SSH_HOST=76.13.220.236`
  - `STAGING_SSH_USER=root`
  - `STAGING_DEPLOY_PATH=/opt/funnyoption-staging`
- the server-side clone currently has `deploy/staging/.env.staging`, but it is checked out on `main` at `fa07e19d48dd7a12c5a3533fdb03ccdb27b75dba` and does not yet contain `scripts/deploy-staging.sh`; run one manual `git pull` in `/opt/funnyoption-staging` after pushing the CI/CD commit so the first workflow can execute the deploy script.
- do not place `FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY` or other runtime secrets in plaintext repo files; keep them in GitHub Secrets or server-only env files.
