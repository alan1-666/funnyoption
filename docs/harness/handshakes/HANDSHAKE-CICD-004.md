# HANDSHAKE-CICD-004

## Task

- [TASK-CICD-004.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-004.md)

## Thread owner

- implementation worker for platform/deployment simplification

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `TASK-CICD-001.md`
- `TASK-CICD-002.md`
- `TASK-CICD-003.md`
- this handshake
- `WORKLOG-CICD-001.md`
- `WORKLOG-CICD-002.md`
- `WORKLOG-CICD-003.md`
- `WORKLOG-CICD-004.md`
- `.github/workflows/staging-deploy.yml`
- `scripts/deploy-staging.sh`
- `docs/deploy/staging-bsc-testnet.md`

## Files in scope

- `.github/workflows/staging-deploy.yml`
- `scripts/deploy-staging.sh`
- `deploy/staging/server-deploy-entrypoint.sh`
- `docs/deploy/staging-bsc-testnet.md`
- `docs/harness/handshakes/HANDSHAKE-CICD-004.md`
- `docs/harness/worklogs/WORKLOG-CICD-004.md`

## Inputs from other threads

- `TASK-CICD-001` created the current GitHub push-to-deploy baseline
- `TASK-CICD-002` added selective service planning and docs-only no-op deploys
- `TASK-CICD-003` fixed the self-bootstrap problem by fetching and checking out
  the target ref before invoking the checked-out repo deploy script
- operator feedback is that the current setup still feels heavier than needed
  because too much orchestration lives in GitHub Actions instead of one stable
  server-side command path
- commander has already fixed the preferred host-side contract unless the
  worker discovers a real server constraint:
  - stable entrypoint path: `/usr/local/bin/funnyoption-staging-deploy`
  - source file in repo: `deploy/staging/server-deploy-entrypoint.sh`
  - lock path: `/var/lock/funnyoption-staging-deploy.lock`
  - repo clone path on the current server: `/opt/funnyoption-staging`
  - default push flow:
    - read current deployed `HEAD` as `diff_base`
    - fetch and check out explicit target SHA/ref
    - invoke checked-out `scripts/deploy-staging.sh --skip-git-sync --diff-base <diff_base>`
    - preserve `--all-services` as the manual full-deploy override

## Outputs back to commander

- changed files
- final control flow from GitHub push to server deploy
- exact install path and invocation contract for the fixed server-side
  entrypoint
- validation notes for:
  - exact-SHA deploy behavior
  - selective one-service or docs-only behavior
  - deploy serialization / locking behavior

## Blockers

- do not regress from exact-SHA deploys to implicit `git pull origin main`
- do not drop the dirty-checkout guard
- do not weaken SSH/secrets handling
- do not silently turn selective deploy back into unconditional full rebuilds
- do not move the fixed host entrypoint back inside the mutable repo checkout
  unless a blocking server constraint forces that tradeoff and it is documented

## Handoff notes

- changed files:
  - `.github/workflows/staging-deploy.yml`
  - `deploy/staging/server-deploy-entrypoint.sh`
  - `docs/deploy/staging-bsc-testnet.md`
  - `docs/harness/handshakes/HANDSHAKE-CICD-004.md`
  - `docs/harness/worklogs/WORKLOG-CICD-004.md`
- final control flow:
  - GitHub Actions resolves the exact target SHA/ref and SSHes to the staging host
  - Actions invokes `/usr/local/bin/funnyoption-staging-deploy --repo /opt/funnyoption-staging --ref <target>` and adds `--all-services` only for manual full deploys
  - the fixed host entrypoint acquires `/var/lock/funnyoption-staging-deploy.lock`, rejects dirty tracked/staged checkout drift, captures the current deployed `HEAD` as `diff_base`, fetches `origin`, checks out the exact target commit, then delegates to the checked-out `scripts/deploy-staging.sh --skip-git-sync --diff-base <previous_head>`
  - `scripts/deploy-staging.sh` remains the source of truth for selective service choice, docs-only no-op deploys, migrations, and health checks
  - symbolic branch refs such as `main` or `refs/heads/main` now prefer the freshly fetched remote-tracking ref before any same-named local branch fallback, while raw commit SHAs still resolve exactly as supplied
- install/update command:
  - `sudo install -m 0755 /opt/funnyoption-staging/deploy/staging/server-deploy-entrypoint.sh /usr/local/bin/funnyoption-staging-deploy`
  - `sudo install -o <deploy-user> -g <deploy-group> -m 0664 /dev/null /var/lock/funnyoption-staging-deploy.lock`
- validation notes:
  - `bash -n scripts/deploy-staging.sh`
  - `bash -n deploy/staging/server-deploy-entrypoint.sh`
  - `ruby -e "require 'yaml'; YAML.load_file('.github/workflows/staging-deploy.yml')"`
  - symbolic-ref proof: temp `origin`/server clone kept its local `main` branch pinned at `b864908f22abd90708d1bcffdf92b1f7669cab72`, then `origin/main` advanced to `144d075219877cd7a55cfe4bdb9a92bc08cb9fa2`; running `--ref main` checked out `144d075...` rather than the stale local branch, and the mocked compose log showed `up -d --build --remove-orphans api`
  - exact-SHA proof: the same temp repo then deployed `--ref 485e6871b4f44d76c059fc310b2cf89dc518d5f2`, and the server checkout landed on that exact SHA with `diff_base=144d075219877cd7a55cfe4bdb9a92bc08cb9fa2`
  - docs-only proof: that exact-SHA deploy changed only docs, printed `no staging service changes detected; skipping compose deploy`, and produced no compose calls
  - lock proof: two concurrent entrypoint invocations against the same repo serialized with start/end order `commit_b -> commit_c`, proving the host lock prevents overlap
- residual tradeoff:
  - changes to `deploy/staging/server-deploy-entrypoint.sh` do not self-install; the copied `/usr/local/bin/funnyoption-staging-deploy` must be refreshed once with the documented `install` command
  - GitHub Actions is now a thin trigger and no longer runs the previous selective build/test matrix before SSHing; deploy safety now relies on the server entrypoint plus the repo deploy script

## Status

- completed
