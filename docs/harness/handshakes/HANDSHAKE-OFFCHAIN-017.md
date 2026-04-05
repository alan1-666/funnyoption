# HANDSHAKE-OFFCHAIN-017

## Task

- [TASK-OFFCHAIN-017.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-017.md)

## Thread owner

- off-chain wallet UX worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `WORKLOG-HARNESS-001.md`
- `HANDSHAKE-OFFCHAIN-013.md`
- `WORKLOG-OFFCHAIN-013.md`
- `web/components/trading-session-provider.tsx`
- `web/lib/session-client.ts`
- `web/components/shell-top-bar.tsx`
- `web/lib/api.ts`
- `web/components/portfolio-shell.tsx`
- `web/components/portfolio-shell.module.css`
- this handshake
- `WORKLOG-OFFCHAIN-017.md`

## Files in scope

- `web/components/trading-session-provider.tsx` only if reconnect prompting
  needs a narrow fix
- `web/lib/session-client.ts` only if restore gating needs a narrow alignment
- `web/components/shell-top-bar.tsx`
- `web/lib/api.ts`
- `web/components/portfolio-shell.tsx`
- `web/components/portfolio-shell.module.css`
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-017.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-017.md`

## Inputs from other threads

- deployed staging on `https://funnyoption.xyz/` now proves canonical V2
  trading-key authorization and restore server truth for wallet
  `0xc421d5ff322e4213a913ec257d6b4458af4255c6`
- commander manually verified:
  - refresh reopens the wallet-provider chooser (`MetaMask` / `Phantom`)
    but does not request a new signature and does not create a new
    trading-key row
  - `GET /api/v1/balances?user_id=1001&limit=10` returns only `POSITION:*`
    rows while `GET /api/v1/balances?user_id=1001&limit=50` includes
    `USDT available=9590 frozen=500`
  - the personal-page wallet copy action currently lacks obvious success
    feedback, and the QR dialog feels too low in the viewport

## Outputs back to commander

- changed files
- validation commands
- one clear before/after summary for:
  - refresh reconnect prompting
  - collateral balance visibility
  - copy-wallet success affordance
  - QR dialog placement

## Blockers

- do not redesign the auth contract in this slice
- do not widen into admin/operator wallet flows
- do not hide real wallet/provider mismatch errors just to silence the chooser
- if a fix depends on a provider SDK limitation, record that explicitly instead
  of masking it with brittle local state

## Status

- completed

## Handoff notes

- treat “no repeated signature” as already-proven PASS
- focus the refresh UX fix on the remaining provider chooser prompt, not on
  reworking the whole trading-key lifecycle
- the balance issue is currently truthful backend data with an unhelpful read
  shape; the slice may fix this in frontend query shape, backend prioritization,
  or the narrowest combination that keeps user-facing collateral visible
- landed narrow frontend fixes:
  - refresh now restores one unambiguous local trading key without probing the
    wallet provider on mount, so the browser stays quiet until a user-triggered
    trading action actually needs wallet verification
  - order signing still forces a real provider reconnect if the restored key is
    only browser-local and not yet re-verified against the connected wallet
  - balance reads now retry with a larger page when the requested collateral
    asset is missing from the first `balances` page, so `USDT` remains visible
    for position-heavy users
  - the personal page now shows a visible copy-success badge/check state, and
    the QR dialog opens higher with a less low-slung visual center
- residual tradeoffs:
  - if the browser holds multiple local trading keys for the same
    `chain + vault`, the app still needs an explicit wallet reconnect to pick
    the right one instead of guessing
  - the collateral read fix now falls back to `limit=200`; if one user can
    eventually exceed that with `POSITION:*` rows, backend prioritization would
    still be the cleaner long-term answer
