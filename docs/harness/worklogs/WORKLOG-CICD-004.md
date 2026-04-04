# WORKLOG-CICD-004

### 2026-04-04 20:18 Asia/Shanghai

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `TASK-CICD-003.md`
  - `HANDSHAKE-CICD-003.md`
  - `WORKLOG-CICD-001.md`
  - `WORKLOG-CICD-003.md`
  - `.github/workflows/staging-deploy.yml`
  - `scripts/deploy-staging.sh`
- changed:
  - created a new CI/CD simplification task, handshake, and worklog
  - updated the top-level plan and active master plan so this decision is
    recorded in repo memory before opening a worker thread
- validated:
  - current deployment behavior already has the right safety semantics:
    exact-SHA checkout, clean-checkout guard, and selective deploy planning
  - the remaining issue is operator ergonomics and orchestration placement,
    not a known release blocker
- blockers:
  - none yet; worker should preserve the existing safety properties while
    simplifying the trigger path
- next:
  - launch one worker on `TASK-CICD-004`

### 2026-04-04 20:24 Asia/Shanghai

- read:
  - `TASK-CICD-004.md`
  - `HANDSHAKE-CICD-004.md`
  - `.github/workflows/staging-deploy.yml`
  - `scripts/deploy-staging.sh`
- changed:
  - fixed the commander-side deployment contract so the worker does not need to
    reopen basic server-path choices
- decided:
  - stable host entrypoint path: `/usr/local/bin/funnyoption-staging-deploy`
  - host lock path: `/var/lock/funnyoption-staging-deploy.lock`
  - current server repo path: `/opt/funnyoption-staging`
  - source file to install from repo: `deploy/staging/server-deploy-entrypoint.sh`
  - default push control flow:
    - GitHub Actions resolves target SHA and SSHes into the server
    - the fixed host entrypoint captures current deployed `HEAD` as `diff_base`
    - the host entrypoint fetches and checks out the explicit target SHA/ref
    - the checked-out repo `scripts/deploy-staging.sh` runs with
      `--skip-git-sync --diff-base <previous_head>`
    - manual full deploy remains `--all-services`
- validated:
  - these choices preserve the current safety boundaries while simplifying the
    operator mental model down to one fixed server command path
- blockers:
  - none at planning level
- next:
  - hand the narrowed contract to the implementation worker

### 2026-04-04 20:50 Asia/Shanghai

- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CICD-001.md`
  - `docs/harness/tasks/TASK-CICD-002.md`
  - `docs/harness/tasks/TASK-CICD-003.md`
  - `docs/harness/tasks/TASK-CICD-004.md`
  - `docs/harness/handshakes/HANDSHAKE-CICD-004.md`
  - `docs/harness/worklogs/WORKLOG-CICD-001.md`
  - `docs/harness/worklogs/WORKLOG-CICD-002.md`
  - `docs/harness/worklogs/WORKLOG-CICD-003.md`
  - `docs/harness/worklogs/WORKLOG-CICD-004.md`
  - `.github/workflows/staging-deploy.yml`
  - `scripts/deploy-staging.sh`
  - `docs/deploy/staging-bsc-testnet.md`
- changed:
  - rewrote `.github/workflows/staging-deploy.yml` into a thin SSH trigger that only resolves the target ref, configures SSH, and invokes `/usr/local/bin/funnyoption-staging-deploy --repo /opt/funnyoption-staging --ref <target>`
  - added `deploy/staging/server-deploy-entrypoint.sh` as the fixed host-side deploy command path, with exact-SHA checkout, dirty-checkout guard, diff-base capture, and `flock`-based serialization
  - made `deploy/staging/server-deploy-entrypoint.sh` executable in the repo so the install target copies a ready-to-run script
  - updated `docs/deploy/staging-bsc-testnet.md` with the new control flow, required secrets, install/update command, and manual deploy/redeploy/rollback commands
  - updated `HANDSHAKE-CICD-004.md` and `WORKLOG-CICD-004.md` with the final handoff, proofs, and residual tradeoffs
- validated:
  - `bash -n scripts/deploy-staging.sh`
  - `bash -n deploy/staging/server-deploy-entrypoint.sh`
  - `ruby -e "require 'yaml'; YAML.load_file('.github/workflows/staging-deploy.yml')"`
  - `git diff --check -- .github/workflows/staging-deploy.yml deploy/staging/server-deploy-entrypoint.sh docs/deploy/staging-bsc-testnet.md docs/harness/handshakes/HANDSHAKE-CICD-004.md docs/harness/worklogs/WORKLOG-CICD-004.md scripts/deploy-staging.sh`
  - exact-SHA + one-service selective local proof with a temp `origin` and server clone:
    - previous deployed `HEAD`: `afa63380ad6fdca4e988a67762a31a6e34d9af04`
    - target deploy SHA: `a955c4f7244b1abd8dc87f4f3cbbba17abaf336e`
    - entrypoint output recorded `target_commit=a955c4f7244b1abd8dc87f4f3cbbba17abaf336e` and `diff_base=afa63380ad6fdca4e988a67762a31a6e34d9af04`
    - mocked compose output showed `up -d --build --remove-orphans api`
  - docs-only local proof with the same temp repo:
    - previous deployed `HEAD`: `a955c4f7244b1abd8dc87f4f3cbbba17abaf336e`
    - target deploy SHA: `b88b843fef16e748191cf4a504238770d67720a9`
    - entrypoint output recorded `target_commit=b88b843fef16e748191cf4a504238770d67720a9`
    - checked-out deploy script printed `no staging service changes detected; skipping compose deploy`
    - mocked compose log stayed empty
  - host lock proof with two concurrent entrypoint invocations against the same temp repo:
    - first start/end pair for `41dc1c807ceec8dd30d883fbf8840d5f5fca9d45`
    - second start/end pair for `ebee5d872833503faf3a793a1bc57a00dbc1fab3`
    - log order was strictly `start first -> end first -> start second -> end second`
- blockers:
  - no repo blocker
  - local proof environment on macOS did not provide a system `flock`, so the proof used a temporary compatible `flock` shim; the production script still requires a real `flock` binary on the Linux staging host
- next:
  - hand back changed files, final control flow, install/update command, validation commands, and residual tradeoffs to commander

### 2026-04-04 21:05 Asia/Shanghai

- read:
  - `deploy/staging/server-deploy-entrypoint.sh`
  - `.github/workflows/staging-deploy.yml`
  - `docs/deploy/staging-bsc-testnet.md`
  - `HANDSHAKE-CICD-004.md`
- changed:
  - commander review reopened `TASK-CICD-004` status from completion handoff to blocked
- validated:
  - exact-SHA push deploy behavior is still correct because the workflow passes `${{ github.sha }}`
  - one correctness gap remains for symbolic manual refs:
    - `resolve_target_commit()` currently verifies `${target_ref}^{commit}` and plain `${branch_ref}^{commit}` before `origin/<ref>`
    - on a server clone that still has a stale local `main` branch, `--ref main` can therefore deploy the stale local branch instead of the freshly fetched remote `origin/main`
- blockers:
  - symbolic ref deploys like `main`, `refs/heads/main`, or similar branch-name inputs are not yet trustworthy enough to close the task
- next:
  - keep the task narrow and fix ref-resolution order so symbolic refs prefer remote-tracking refs after fetch, while raw commit-SHA deploys continue to work unchanged

### 2026-04-04 20:58 CST

- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CICD-004.md`
  - `docs/harness/handshakes/HANDSHAKE-CICD-004.md`
  - `docs/harness/worklogs/WORKLOG-CICD-004.md`
  - `.github/workflows/staging-deploy.yml`
  - `deploy/staging/server-deploy-entrypoint.sh`
  - `scripts/deploy-staging.sh`
  - `docs/deploy/staging-bsc-testnet.md`
- changed:
  - updated `deploy/staging/server-deploy-entrypoint.sh` so raw commit SHAs still resolve directly, but symbolic branch refs such as `main` and `refs/heads/main` now prefer the freshly fetched `origin/<branch>` ref before any same-named local-branch fallback
  - updated `docs/deploy/staging-bsc-testnet.md` to document the new symbolic-ref resolution rule
  - updated `HANDSHAKE-CICD-004.md` and `WORKLOG-CICD-004.md` with blocker resolution and new validation evidence
- validated:
  - `bash -n deploy/staging/server-deploy-entrypoint.sh`
  - `bash -n scripts/deploy-staging.sh`
  - `ruby -e "require 'yaml'; YAML.load_file('.github/workflows/staging-deploy.yml')"`
  - `git diff --check -- .github/workflows/staging-deploy.yml deploy/staging/server-deploy-entrypoint.sh docs/deploy/staging-bsc-testnet.md docs/harness/handshakes/HANDSHAKE-CICD-004.md docs/harness/worklogs/WORKLOG-CICD-004.md`
  - symbolic-ref proof with a temp `origin` and server clone that intentionally kept a stale local `main` branch:
    - local `main` before and after deploy stayed at `b864908f22abd90708d1bcffdf92b1f7669cab72`
    - `origin/main` advanced to `144d075219877cd7a55cfe4bdb9a92bc08cb9fa2`
    - running `deploy/staging/server-deploy-entrypoint.sh --repo <temp-server> --ref main` checked out `144d075219877cd7a55cfe4bdb9a92bc08cb9fa2`
    - mocked compose output showed `up -d --build --remove-orphans api`, so selective deploy stayed intact
  - exact-SHA proof in the same temp repo:
    - target SHA `485e6871b4f44d76c059fc310b2cf89dc518d5f2`
    - server checkout landed on `485e6871b4f44d76c059fc310b2cf89dc518d5f2`
    - the change was docs-only, so the deploy script printed `no staging service changes detected; skipping compose deploy` and produced no compose calls
- blockers:
  - no repo blocker remains for `TASK-CICD-004`
  - local macOS proof still needed a temporary `flock` shim because the host environment here does not ship a system `flock`; the production Linux host still requires a real `flock` binary
- next:
  - hand back the symbolic-ref fix, updated resolution rule, and fresh proofs so commander can close `TASK-CICD-004`

### 2026-04-04 21:10 CST

- read:
  - GitHub Actions failure output for missing `/usr/local/bin/funnyoption-staging-deploy`
  - current server checkout state at `/opt/funnyoption-staging`
- changed:
  - no repo-code change; completed the one-time host install / rollout step that
    the docs already required
- validated:
  - staging host `76.13.220.236` was still on `HEAD=125f9cd`, so
    `deploy/staging/server-deploy-entrypoint.sh` was not yet present there
  - installed flow executed manually:
    - `git fetch --prune origin`
    - `git checkout --detach d7a79c177beec77e0a43f95ca69adc3242905ff4`
    - `install -m 0755 deploy/staging/server-deploy-entrypoint.sh /usr/local/bin/funnyoption-staging-deploy`
    - `install -o root -g root -m 0664 /dev/null /var/lock/funnyoption-staging-deploy.lock`
    - `/usr/local/bin/funnyoption-staging-deploy --repo /opt/funnyoption-staging --ref d7a79c177beec77e0a43f95ca69adc3242905ff4`
  - host deploy completed successfully and printed `no staging service changes detected; skipping compose deploy`
- blockers:
  - none remaining for future pushes; the missing piece was the first host-side
    install of the fixed entrypoint
- next:
  - future GitHub Actions runs can call the installed host entrypoint directly
