# TASK-CHAIN-004

## Summary

Restore staging deposit ingestion after deploy restarts by fixing the chain listener's stale start-block replay against pruned public RPC history, and document a restart-safe scanner strategy.

## Scope

- investigate why staging chain deposits now stay absent from `/api/v1/deposits` and `/api/v1/balances` even though BSC Testnet Vault txs succeed on-chain
- use the current commander evidence as the starting point:
  - `FUNNYOPTION_CHAIN_RPC_URL=https://bsc-testnet-rpc.publicnode.com`
  - `FUNNYOPTION_CHAIN_START_BLOCK=99452107`
  - blocked tx `0x4129a4db5f66760ca8374a1dbe3df94652552df9768500ff0d49ec9654733a6c`
  - blocked tx block `99674293`
  - chain logs repeatedly report `History has been pruned for this block`
- fix the chain listener so a service restart does not permanently trap scanning on an already-pruned static start block
- prefer a restart-safe persisted scanner checkpoint or an explicit safe fast-forward policy with clear observability and replay tradeoffs
- update the staging deploy runbook with the final chain listener restart rule and any one-time recovery step
- do not print or write private-key plaintext

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-STAGING-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-STAGING-001.md)
- [/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md](/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-004.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-004.md)

## Owned files

- `internal/chain/service/**`
- `docs/deploy/staging-bsc-testnet.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-004.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-004.md`
- if a persisted cursor needs schema support, add the smallest migration/doc updates required for that cursor only

## Acceptance criteria

- root cause of the staging deposit miss is recorded with concrete block/RPC evidence
- after the fix, restarting the chain service no longer wedges the listener on a pruned block range
- a fresh BSC Testnet Vault deposit from a session-bound wallet appears in `/api/v1/deposits` and `/api/v1/balances`
- regression coverage exists for the listener restart / pruned-history edge case, or the handoff states the exact reason it cannot be automated locally
- staging runbook documents any required one-time recovery step and the steady-state restart behavior

## Validation

- targeted Go tests for `internal/chain/service/...`
- local or staging smoke that proves one fresh deposit is ingested after a chain-service restart
- server log snippet or script output showing the listener has moved past the stale start block without exposing secrets

## Dependencies

- `TASK-STAGING-001` supplies the failing tx, block number, and symptom matrix

## Handoff

- write the root cause, patch summary, validation commands, restart behavior, and any one-time staging recovery command to `WORKLOG-CHAIN-004.md`
- if a residual tradeoff remains (for example intentionally skipping an old pruned range), state the skipped range and why it is acceptable
