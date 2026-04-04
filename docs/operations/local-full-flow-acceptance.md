# Local Full-Flow Acceptance

## Purpose

Run one repeatable local acceptance flow that stitches together:

1. trading-key challenge plus wallet authorization registration
2. truthful restore readback
3. deposit credit
4. oracle crypto market creation
5. order placement and matching
6. oracle auto settlement
7. payout and terminal readback verification

## Required environment

This flow requires the persistent local-chain path, not the older in-process
proof vault path.

Prerequisites:

- [`.env.local`](/Users/zhangza/code/funnyoption/.env.local) contains `FUNNYOPTION_LOCAL_CHAIN_MODE=anvil`
- local dev has already been started through:

```bash
/Users/zhangza/code/funnyoption/scripts/dev-up.sh
```

- the generated files exist:
  - [`.run/dev/local-chain.env`](/Users/zhangza/code/funnyoption/.run/dev/local-chain.env)
  - [`.run/dev/local-chain-wallets.env`](/Users/zhangza/code/funnyoption/.run/dev/local-chain-wallets.env)

Why this requirement exists:

- canonical `POST /api/v1/trading-keys/challenge` and `POST /api/v1/trading-keys`
  require one configured `chain_id + vault_address`
- the persistent local chain gives the API, chain-service, frontend, and this
  harness the same real vault target
- deposit credit can then follow the real `approve -> deposit -> listener ->
  account credit` path instead of the older local proof shortcut

## Run

```bash
/Users/zhangza/code/funnyoption/scripts/local-full-flow.sh
```

The command prints one JSON summary with:

- `pass_fail_matrix`
- key ids
- balances before and after the critical steps
- oracle resolution readback
- concrete curl / psql readback commands
- residual blind spots

## Signature boundaries

The acceptance path is intentionally explicit about what is real cryptography
versus what is still a harness substitute:

| Step | Signature / actor truth |
| --- | --- |
| Trading-key registration | Real `EIP-712` payload shape, but signed by deterministic local test EOAs instead of a browser wallet popup |
| Truthful restore | No signature; harness verifies local metadata against `GET /api/v1/sessions` |
| Deposit | Real EVM `approve` and `deposit` transactions signed by the buyer test EOA on the persistent local chain |
| Market create | Operator envelope signed by the deterministic local operator EOA |
| First-liquidity | Operator envelope signed by the deterministic local operator EOA |
| User order | Real `ED25519` trading-key signature, generated inside the harness process rather than stored in browser IndexedDB |
| Oracle settlement | No wallet signature; a local fake Binance HTTP fixture drives the real oracle worker |
| Payout/readback | No signature; verification is API plus SQL readback |

## What the runner does

1. requests trading-key challenges for buyer and maker
2. signs canonical `AuthorizeTradingKey` typed data with local test-wallet keys
3. registers both trading keys through `/api/v1/trading-keys`
4. verifies truthful restore via `/api/v1/sessions`
5. submits a real buyer `approve + deposit` on the persistent local chain
6. waits for `chain-service` to credit the deposit through normal readbacks
7. starts a local fake Binance fixture plus the real oracle worker
8. creates one `CRYPTO` oracle market with deterministic threshold metadata
9. issues first liquidity, waits for the bootstrap `SELL`, then submits the
   crossing `BUY`
10. waits for trade match, oracle auto settlement, payout creation, and final
    readback
11. prints a JSON summary with pass/fail evidence and ids

## Residual blind spots

This flow is intentionally stronger than the legacy local lifecycle smoke, but
it still does not prove:

- real browser wallet UX for `eth_signTypedData_v4`
- hardware wallet or smart-contract-wallet signature quirks
- browser `localStorage` / IndexedDB loss, restore, and hydration behavior
- live external oracle provider behavior, throttling, or outage handling
- end-user payout claim transaction submission and browser claim UX
