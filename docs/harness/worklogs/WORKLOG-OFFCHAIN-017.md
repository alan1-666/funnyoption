# WORKLOG-OFFCHAIN-017

### 2026-04-05 02:42 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `WORKLOG-HARNESS-001.md`
  - `HANDSHAKE-OFFCHAIN-013.md`
  - `WORKLOG-OFFCHAIN-013.md`
  - `web/components/shell-top-bar.tsx`
  - `web/lib/api.ts`
  - `web/components/portfolio-shell.tsx`
  - `web/components/portfolio-shell.module.css`
- changed:
  - created a narrow follow-up task for deployed wallet/portfolio UX polish:
    - quiet refresh reconnect behavior
    - truthful collateral balance visibility for position-heavy users
    - visible wallet-copy success affordance
    - improved QR dialog placement
- validated:
  - commander staging checks already showed:
    - no repeated signature on refresh
    - no new trading-key row created on refresh
    - `USDT` exists for `user_id=1001` but is hidden from `balances?limit=10`
- blockers:
  - none yet
- next:
  - assign one narrow web worker to land the UX/read-path polish without
    reopening auth-contract scope

### 2026-04-05 02:51 CST

- read:
  - `AGENTS.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-OFFCHAIN-017.md`
  - `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-017.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-017.md`
  - `docs/architecture/direct-deposit-session-key.md`
  - `docs/harness/worklogs/WORKLOG-HARNESS-001.md`
  - `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-013.md`
  - `docs/harness/worklogs/WORKLOG-OFFCHAIN-013.md`
  - `web/components/trading-session-provider.tsx`
  - `web/lib/session-client.ts`
  - `web/components/shell-top-bar.tsx`
  - `web/lib/api.ts`
  - `web/components/portfolio-shell.tsx`
  - `web/components/portfolio-shell.module.css`
- changed:
  - updated `web/lib/session-client.ts` so one unambiguous local trading key
    can restore silently on refresh without probing `window.ethereum`, while
    multiple local keys still require an explicit wallet reconnect
  - updated `web/components/trading-session-provider.tsx` to distinguish a
    browser-restored key from a provider-backed wallet connection and to delay
    wallet verification until a user-triggered trading action actually needs it
  - updated `web/lib/api.ts`, `web/components/shell-top-bar.tsx`, and
    `web/components/portfolio-shell.tsx` so balance reads retry with a larger
    `balances` page when `USDT` is missing from the first page
  - updated `web/components/portfolio-shell.tsx` and
    `web/components/portfolio-shell.module.css` to add a visible copy-success
    badge/check state and to move the QR dialog higher in the viewport
  - updated `HANDSHAKE-OFFCHAIN-017.md` to mark the task completed and record
    residual limits
- validated:
  - `cd web && npm run build`
  - local headless-Chrome browser proof against a mock API on
    `http://127.0.0.1:8080`:
    - refresh with a stored valid trading key made zero wallet RPC calls
      (`walletCalls=[]`), so no automatic provider chooser / reconnect probe
      was triggered on load
    - browser restore called
      `GET /api/v1/sessions?wallet_address=0xc421d5ff322e4213a913ec257d6b4458af4255c6&vault_address=0xe7f1725e7734ce288f8367e1bb143e90bb3f0512&limit=200`
    - browser balance reads called
      `GET /api/v1/balances?user_id=1001&limit=10` and then fallback
      `GET /api/v1/balances?user_id=1001&limit=200`, and the UI showed
      `9,590 USDT`
    - copy feedback became visible and the QR dialog rendered with
      `top=65`, `height=658`, `centerOffset=-56` in a `1440x900` viewport
    - screenshot artifact:
      `/Users/zhangza/code/funnyoption/output/playwright/task-offchain-017-portfolio.png`
- blockers:
  - none at this slice
- next:
  - commander can rerun the same four checks on deployed staging to confirm the
    UX now matches the local/browser proof
