# TASK-CICD-001

## Summary

Build GitHub push-to-deploy automation for the current staging server deployment without committing plaintext secrets.

## Scope

- inspect the current deployment runbook, Dockerfiles, and any server-side deployment conventions that are already documented in the repo
- design and implement a GitHub Actions pipeline that deploys the current staging environment automatically after a push to the chosen branch
- keep all server SSH material, private keys, and chain/operator secrets in GitHub Secrets or server-only env files
- do not commit `.secrets`, private keys, or plaintext deployment credentials
- document the required GitHub Secrets, deployment assumptions, rollback/retry behavior, and operational commands
- if the current server deploy flow requires manual details not present in repo files (SSH host/user/path, compose/systemd command, registry strategy), record the exact blocker and the minimal user input needed

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md](/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md)
- [/Users/zhangza/code/funnyoption/configs/staging/funnyoption.env.example](/Users/zhangza/code/funnyoption/configs/staging/funnyoption.env.example)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CICD-001.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CICD-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-001.md)

## Owned files

- `.github/workflows/**`
- `scripts/**` for narrowly scoped deploy helper scripts
- `docs/deploy/**`
- `docs/harness/worklogs/WORKLOG-CICD-001.md`
- `docs/harness/handshakes/HANDSHAKE-CICD-001.md` if blocker/status updates are needed

## Acceptance criteria

- repo contains a GitHub Actions workflow for push-triggered staging deploy
- required GitHub Secrets are documented by name and purpose, with no plaintext secret values committed
- workflow has a validation step before deployment and a clear deployment command path
- deployment assumptions and rollback/manual recovery notes are documented
- if a fully working workflow cannot be proven because server SSH details are missing, the worker still returns a concrete partial implementation plus a precise blocker list

## Validation

- static review of `.github/workflows/**` and any deploy helper scripts
- run local syntax/lint checks for scripts if applicable
- if `gh` or server SSH validation is available, perform a dry-run check without exposing secrets

## Dependencies

- staging deployment already exists and is reachable at `https://funnyoption.xyz/` and `https://admin.funnyoption.xyz/`
- this task can run in parallel with `TASK-STAGING-001` if file ownership remains disjoint

## Handoff

- return changed files, workflow behavior, required GitHub Secrets names, and deployment assumptions
- include any one-time manual setup steps in GitHub repo settings or server env
- redact all private keys and secret values from worklog and chat
