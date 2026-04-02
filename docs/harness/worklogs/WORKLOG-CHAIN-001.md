# WORKLOG-CHAIN-001

### 2026-04-01 22:46 Asia/Shanghai

- read:
  - `WORKLOG-OFFCHAIN-008.md`
  - `HANDSHAKE-OFFCHAIN-008.md`
  - `docs/architecture/direct-deposit-session-key.md`
  - `docs/topics/kafka-topics.md`
  - `internal/api/dto/order.go`
  - `internal/api/handler/order_handler.go`
  - `internal/api/handler/sql_store.go`
  - `internal/chain/service/claim_processor.go`
- changed:
  - created the first chain hardening task as a narrow claim-lane validation pass
- validated:
  - commander review confirmed the gap on both sides of the same lane:
    - API claim requests require fields but do not validate address shape
    - chain submission currently passes queued addresses through `common.HexToAddress`, which silently maps malformed input to the zero address
- blockers:
  - none yet; worker should verify the exact bad-input behavior before patching
- next:
  - launch a worker against `TASK-CHAIN-001`

### 2026-04-02 00:31 Asia/Shanghai

- read:
  - `internal/api/dto/order.go`
  - `internal/api/handler/order_handler.go`
  - `internal/api/handler/order_handler_test.go`
  - `internal/api/handler/sql_store.go`
  - `internal/chain/service/claim_processor.go`
  - `internal/chain/service/claim_processor_test.go`
- changed:
  - `internal/api/dto/order.go`
    - added claim-address normalization and validation helpers
    - added queued-claim payload validation for address shape, zero-address rejection, and positive payout amount
  - `internal/api/handler/order_handler.go`
    - `POST /api/v1/payouts/:event_id/claim` now rejects malformed or zero claim addresses with `400` before queue creation
  - `internal/api/handler/order_handler_test.go`
    - existing accepted-path test now proves normalized addresses reach the store
    - added a malformed-wallet regression test that proves queue creation is skipped and the API returns `400`
  - `internal/chain/service/claim_processor.go`
    - validates `ref_id`, wallet address, recipient address, payout amount, and vault address before building/signing a claim tx
    - invalid queued claim tasks now fail before any tx submission path is entered
  - `internal/chain/service/claim_processor_test.go`
    - added a regression test proving an invalid queued claim task is marked failed and sends no tx
- validated:
  - `cd /Users/zhangza/code/funnyoption && go test ./internal/api/... ./internal/chain/...`
  - API-level proof now lives in `TestCreateClaimPayoutRejectsMalformedWalletAddress`
  - chain-level proof now lives in `TestClaimProcessorPollOnceFailsInvalidQueuedClaim`
- blockers:
  - no blocker remains in the claim lane itself after this validation pass
  - broader real-chain proof still depends on a live vault/operator config rather than the default local env, but malformed claim payloads no longer queue or degrade into zero-address submissions
- next:
  - commander can treat `TASK-CHAIN-001` as complete
