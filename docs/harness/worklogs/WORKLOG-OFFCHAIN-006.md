# WORKLOG-OFFCHAIN-006

### 2026-04-01 21:55 Asia/Shanghai

- read:
  - `WORKLOG-OFFCHAIN-003.md`
  - `HANDSHAKE-OFFCHAIN-003.md`
  - `web/lib/api.ts`
- changed:
  - created a narrow follow-up task for honest SSR degraded-state handling
- validated:
  - the follow-up is tightly scoped to the read-surface truthfulness gap found in commander review
- blockers:
  - none yet
- next:
  - launch a worker against `TASK-OFFCHAIN-006`

### 2026-04-01 22:13 Asia/Shanghai

- read:
  - `web/lib/api.ts`
  - `web/lib/types.ts`
  - `web/app/page.tsx`
  - `web/app/control/page.tsx`
  - `web/app/markets/[marketId]/page.tsx`
  - `web/components/live-market-panel.tsx`
  - `web/components/chain-task-board.tsx`
  - `scripts/dev-up.sh`
  - `scripts/dev-status.sh`
- changed:
  - `web/lib/types.ts`
  - `web/lib/api.ts`
  - `web/app/page.tsx`
  - `web/app/control/page.tsx`
  - `web/app/markets/[marketId]/page.tsx`
  - `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-006.md`
- validated:
  - `cd /Users/zhangza/code/funnyoption/web && npm run build`
  - direct API contract checks:
    - `GET /api/v1/markets?limit=2`: returns a real `items` array
    - `GET /api/v1/markets/220140402`: returns the resolved market payload
    - `GET /api/v1/markets/999999999`: returns `404`
    - `GET /api/v1/trades?limit=3`: returns a real `items` array
    - `GET /api/v1/trades?market_id=220140402&limit=20`: returns `{"items":null}`
    - `GET /api/v1/chain-transactions?limit=5`: returns `{"items":null}`
  - SSR smoke against the real local API via `npm run start -- --hostname 127.0.0.1 --port 3001`:
    - homepage `/`: PASS; rendered real markets/trades and did not show outage copy
    - detail `/markets/220140402`: PASS; rendered the market payload and honestly marked the market-scoped trade tape unavailable because the API returned `{"items":null}`
    - detail `/markets/999999999`: PASS; rendered `Market not found`, not `API unavailable`
    - control `/control`: PASS; rendered real market runtime and honestly marked the claim queue unavailable because the API returned `{"items":null}`
  - degraded-path SSR smoke via `NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:65535 npm run dev -- --hostname 127.0.0.1 --port 3003`:
    - homepage `/`: PASS; rendered degraded copy plus explicit market-board and trade-tape unavailable states
    - detail `/markets/220140402`: PASS; rendered `API unavailable` instead of falling through to `not found`
    - control `/control`: PASS; rendered market runtime unavailable and queue snapshot unavailable instead of empty-state metrics
  - note on tooling:
    - `next start` reuses the already built env, so the intentionally invalid base-url smoke had to run under `next dev` to prove the degraded path
- blockers:
  - no frontend blocker remains for truthful SSR reads
  - backend collection endpoints still return `{"items":null}` for at least `chain-transactions` and market-scoped `trades`; the frontend now surfaces those as broken-response unavailable states instead of hiding them as empty collections
- next:
  - commander can treat `TASK-OFFCHAIN-006` as frontend-complete
  - if desired before chain hardening, route a backend/API contract cleanup task so empty collections return `[]` instead of `null`
