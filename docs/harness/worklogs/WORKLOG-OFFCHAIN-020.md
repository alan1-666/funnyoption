# WORKLOG-OFFCHAIN-020

### 2026-04-06 03:08 CST

- read:
  - `PLAN.md`
  - `TASK-OFFCHAIN-019.md`
  - `HANDSHAKE-OFFCHAIN-019.md`
  - `WORKLOG-OFFCHAIN-019.md`
  - `web/app/markets/[marketId]/page.tsx`
  - `web/components/live-market-panel.tsx`
  - `web/components/order-ticket.tsx`
  - `web/components/market-order-activity.tsx`
- changed:
  - created the surface-copy cleanup task, handshake, and worklog
- validated:
  - commander review confirmed the remaining issue is wording, not layout:
    the page still shows internal design-rationale sentences to end users
- blockers:
  - none yet
- next:
  - add the frontend copy guideline and strip the market detail page down to
  concise user-facing text

### 2026-04-06 03:15 CST

- changed:
  - `docs/architecture/frontend-surface-copy.md`
  - `web/app/markets/[marketId]/page.tsx`
  - `web/components/live-market-panel.tsx`
  - `web/components/live-market-panel.module.css`
  - `web/components/order-ticket.tsx`
  - `web/components/market-order-activity.tsx`
- implemented:
  - removed the visible meta/design-rationale copy from the public market
    detail page
  - replaced long explanatory empty states with concise state labels
  - added one checked-in frontend surface-copy guideline that forbids
    self-referential/meta UI language
- validated:
  - `rg -n "这里会显示|这里会|把走势|把下单动作|不再像以前|像 Worm 一样|主舞台|收成" web -g '!node_modules'`
  - `cd web && npm run build`
  - `git diff --check`
- blockers:
  - none locally
- next:
  - push to staging and verify the cleaned-up detail page copy in the browser

### 2026-04-06 03:24 CST

- changed:
  - `web/components/order-ticket.tsx`
  - `web/components/order-ticket.module.css`
- implemented:
  - removed the remaining leverage helper sentence that still leaked
    implementation/roadmap-style wording into the trade rail
- validated:
  - `cd web && npm run build`
  - `git diff --check`
