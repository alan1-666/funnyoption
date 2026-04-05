# WORKLOG-CHAIN-025

## Goal

Make post-close market lifecycle truthfully distinguish oracle-closed markets
from manual-resolution markets by introducing one runtime-effective
`WAITING_RESOLUTION` state and tightening operator resolve gating accordingly.

## Notes

- task opened from commander thread after `TASK-CHAIN-024`
- chosen direction before coding:
  - keep runtime-effective derivation from stored `status + close_at +
    resolve_at + resolution mode`
  - do not add a background persistence job
  - unresolved markets stay runtime `CLOSED` between `close_at` and `resolve_at`
  - use `WAITING_RESOLUTION` only for non-oracle markets that have reached
    `resolve_at` but are not yet resolved
  - ordinary operator resolve must only accept `WAITING_RESOLUTION`
- implementation landed:
  - `internal/api/handler/market_lifecycle.go` now derives runtime-effective
    `WAITING_RESOLUTION` for non-oracle markets only at/after `resolve_at`,
    while oracle markets remain runtime `CLOSED` until oracle resolution lands
  - `internal/api/handler/order_handler.go` now rejects ordinary operator
    resolve unless a market is runtime `WAITING_RESOLUTION`, and rejects oracle
    markets on the ordinary manual lane
  - `internal/api/handler/sql_store.go` now supports truthful runtime
    filtering for `OPEN`, `CLOSED`, and `WAITING_RESOLUTION`
  - admin/web status types and labels were updated to surface `等待裁决`
- validation:
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/api/handler ./internal/matching/service ./internal/oracle/service ./internal/settlement/service`
  - `cd web && npm run build`
  - `cd admin && npm run build`
  - `git diff --check`

## Status

- completed
