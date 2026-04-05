# TASK-OFFCHAIN-017

## Summary

Polish the staging-proven wallet restore and portfolio UX so refresh is quieter,
collateral balance remains visible for position-heavy users, wallet-address copy
gives explicit success feedback, and the wallet QR dialog opens in a more
usable viewport position.

## Scope

- close the narrow frontend/read-path issues discovered during deployed staging
  verification for wallet `0xc421d5ff322e4213a913ec257d6b4458af4255c6`
- keep the slice focused on wallet reconnect + portfolio presentation:
  - refresh with a valid local trading key should not reopen the wallet
    provider chooser unless user action or a real restore blocker requires it
  - top-bar / portfolio collateral summary should not hide `USDT` just because
    `balances?limit=10` returns only `POSITION:*` rows first
  - wallet-address copy on the personal page should show an obvious success
    affordance, not only a tooltip/title change
  - the wallet QR dialog should render higher and more naturally in the
    viewport instead of feeling visually too low
- keep the slice narrow:
  - do not widen into auth-contract redesign
  - do not touch admin/operator wallet flows
  - do not change trading-key server semantics unless a narrow reconnect guard
    really requires it

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-HARNESS-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-HARNESS-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-013.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-013.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-013.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-013.md)
- [/Users/zhangza/code/funnyoption/web/components/trading-session-provider.tsx](/Users/zhangza/code/funnyoption/web/components/trading-session-provider.tsx)
- [/Users/zhangza/code/funnyoption/web/lib/session-client.ts](/Users/zhangza/code/funnyoption/web/lib/session-client.ts)
- [/Users/zhangza/code/funnyoption/web/components/shell-top-bar.tsx](/Users/zhangza/code/funnyoption/web/components/shell-top-bar.tsx)
- [/Users/zhangza/code/funnyoption/web/lib/api.ts](/Users/zhangza/code/funnyoption/web/lib/api.ts)
- [/Users/zhangza/code/funnyoption/web/components/portfolio-shell.tsx](/Users/zhangza/code/funnyoption/web/components/portfolio-shell.tsx)
- [/Users/zhangza/code/funnyoption/web/components/portfolio-shell.module.css](/Users/zhangza/code/funnyoption/web/components/portfolio-shell.module.css)

## Owned files

- `web/components/trading-session-provider.tsx` only if reconnect prompting
  needs a narrow behavior guard
- `web/lib/session-client.ts` only if reconnect / restore coordination needs a
  narrow helper change
- `web/components/shell-top-bar.tsx`
- `web/lib/api.ts`
- `web/components/portfolio-shell.tsx`
- `web/components/portfolio-shell.module.css`
- narrow shared style/helpers only if necessary for the UX polish
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-017.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-017.md`

## Acceptance criteria

- a refresh with an already-valid local trading key does not auto-reopen the
  provider chooser unless the user actually needs to reconnect a wallet
- the main collateral balance summary remains truthful for users whose first
  balance page is crowded with `POSITION:*` assets
- copying the wallet address on the personal page gives an obvious success
  cue/animation
- opening the wallet QR dialog feels visually centered and no longer appears
  awkwardly low in the viewport
- validation includes:
  - `cd web && npm run build`
  - one manual/browser proof for refresh behavior
  - one proof that the displayed collateral balance still shows `USDT` for a
    position-heavy user like staging `user_id=1001`

## Validation

- `cd web && npm run build`
- one browser/manual proof for:
  - refresh with existing active trading key
  - top-bar or portfolio collateral balance visibility
  - copy-wallet feedback
  - QR dialog placement

## Dependencies

- `TASK-OFFCHAIN-013` restore UX baseline is complete
- `TASK-OFFCHAIN-015` and `TASK-OFFCHAIN-016` canonical trading-key runtime
  are complete

## Handoff

- return changed files, validation commands, and before/after UX behavior
- call out any residual limitation, such as wallet-provider SDK behavior that
  cannot be fully silenced without a larger connection-management refactor
