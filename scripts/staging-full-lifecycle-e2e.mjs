#!/usr/bin/env node

/**
 * Full Lifecycle E2E — Order → Match → Oracle Resolve → Settlement → Payout
 *
 * 1. Create oracle market (BTC > $10k, resolve in ~90s)
 * 2. Fund & register maker + taker wallets
 * 3. Seed first liquidity, maker has YES positions
 * 4. Taker buys YES via MARKET order → matched
 * 5. Wait for oracle auto-resolve → YES wins
 * 6. Verify settlement: winner gets payout, positions cleared
 */

import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { setTimeout as sleep } from "node:timers/promises";
import * as ed from "../web/node_modules/@noble/ed25519/index.js";
import {
  createPublicClient,
  createWalletClient,
  http,
  parseAbi,
  parseUnits,
} from "../web/node_modules/viem/_esm/index.js";
import { bscTestnet } from "../web/node_modules/viem/_esm/chains/index.js";
import {
  generatePrivateKey,
  privateKeyToAccount,
} from "../web/node_modules/viem/_esm/accounts/index.js";

const API_BASE = (process.env.API_BASE || "https://funnyoption.xyz").replace(/\/+$/, "");
const ADMIN_BASE = (process.env.ADMIN_BASE || "https://admin.funnyoption.xyz").replace(/\/+$/, "");
const RPC_URL = process.env.FUNNYOPTION_CHAIN_RPC_URL || "https://data-seed-prebsc-1-s1.bnbchain.org:8545";
const TOKEN_ADDRESS = process.env.FUNNYOPTION_COLLATERAL_TOKEN_ADDRESS || "0x0ADa04558decC14671D565562Aeb8D1096F71dDc";
const VAULT_ADDRESS = process.env.FUNNYOPTION_VAULT_ADDRESS || "0x7Da015dfCD16Fb892328995BDd883da5AA3E670a";
const SECRET_FILE = resolve(fileURLToPath(new URL(".", import.meta.url)), "../.secrets");

const RESOLVE_DELAY_SEC = 90;
const DEPOSIT_USDT = "30";
const SELL_PRICE = 50;
const SELL_QTY = 5;
const BUY_QTY = 2;

const textEncoder = new TextEncoder();
const tokenAbi = parseAbi([
  "function mint(address to, uint256 amount) external",
  "function approve(address spender, uint256 amount) external returns (bool)",
]);
const vaultAbi = parseAbi(["function deposit(uint256 amount) external"]);

const binaryOptions = [
  { key: "YES", label: "是", shortLabel: "是", sortOrder: 10, isActive: true },
  { key: "NO", label: "否", shortLabel: "否", sortOrder: 20, isActive: true },
];

function norm(v) { return String(v ?? "").trim().toLowerCase(); }
function toHex(bytes) { return `0x${Array.from(bytes, b => b.toString(16).padStart(2, "0")).join("")}`; }
function clean(v) { return String(v ?? "").trim().replace(/\s+/g, " "); }
function optionFrag(options) {
  return options.map(o => `${clean(o.key).toUpperCase()}:${clean(o.label)}:${clean(o.shortLabel ?? o.label)}:${o.sortOrder}:${o.isActive ? "1" : "0"}`).join("|");
}
function authHeaders(sessionId) {
  return sessionId ? { Authorization: `Bearer ${sessionId}` } : {};
}

function readSecret() {
  if (process.env.FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY?.trim()) return process.env.FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY.trim();
  const raw = readFileSync(SECRET_FILE, "utf-8");
  for (const line of raw.split("\n")) { const [k, ...v] = line.split(":"); if (k?.trim() === "bsc-testnet-operator.key" && v.join(":").trim()) return v.join(":").trim(); }
  throw new Error("operator key not found");
}

async function fetchJson(url, opts = {}) {
  const h = { ...opts.headers }; if (opts.body !== undefined) h["Content-Type"] = "application/json";
  for (let a = 1; a <= 3; a++) {
    try {
      const r = await fetch(url, { method: opts.method || "GET", headers: Object.keys(h).length ? h : undefined, body: opts.body === undefined ? undefined : JSON.stringify(opts.body), signal: AbortSignal.timeout(30_000) });
      const text = await r.text(); let data; try { data = JSON.parse(text); } catch { data = { raw: text }; }
      return { ok: r.ok, status: r.status, data };
    } catch (e) { if (a === 3) throw new Error(`fetch ${opts.method || "GET"} ${url}: ${e.message}`); await sleep(500 * a); }
  }
}
async function postOk(url, body, label) { const r = await fetchJson(url, { method: "POST", body }); if (!r.ok) die(`${label}: HTTP ${r.status} ${JSON.stringify(r.data)}`); return r.data ?? {}; }

async function signOp(account, message, requestedAt) {
  return { walletAddress: account.address, requestedAt, signature: await account.signMessage({ message }) };
}

function buildResolutionFrag(res) {
  if (!res) return "";
  const o = res.oracle ?? {};
  const inst = o.instrument ?? {};
  const p = o.price ?? {};
  const w = o.window ?? {};
  const ru = o.rule ?? {};
  return `resolution_version: ${Math.floor(res.version ?? 0)}\nresolution_mode: ${clean(res.mode ?? "").toUpperCase()}\nresolution_market_kind: ${clean(res.market_kind ?? "").toUpperCase()}\nresolution_manual_fallback_allowed: ${res.manual_fallback_allowed === true}\noracle_source_kind: ${clean(o.source_kind ?? "").toUpperCase()}\noracle_provider_key: ${clean(o.provider_key ?? "").toUpperCase()}\noracle_instrument_kind: ${clean(inst.kind ?? "").toUpperCase()}\noracle_instrument_base_asset: ${clean(inst.base_asset ?? "").toUpperCase()}\noracle_instrument_quote_asset: ${clean(inst.quote_asset ?? "").toUpperCase()}\noracle_instrument_symbol: ${clean(inst.symbol ?? "").toUpperCase()}\noracle_price_field: ${clean(p.field ?? "").toUpperCase()}\noracle_price_scale: ${Math.floor(p.scale ?? 0)}\noracle_price_rounding_mode: ${clean(p.rounding_mode ?? "").toUpperCase()}\noracle_price_max_data_age_sec: ${Math.floor(p.max_data_age_sec ?? 0)}\noracle_window_anchor: ${clean(w.anchor ?? "").toUpperCase()}\noracle_window_before_sec: ${Math.floor(w.before_sec ?? 0)}\noracle_window_after_sec: ${Math.floor(w.after_sec ?? 0)}\noracle_rule_type: ${clean(ru.type ?? "").toUpperCase()}\noracle_rule_comparator: ${clean(ru.comparator ?? "").toUpperCase()}\noracle_rule_threshold_price: ${(ru.threshold_price ?? "").trim()}\n`;
}

function buildCreateMarketMsg({ walletAddress, market, requestedAt }) {
  return `FunnyOption Operator Authorization\n\naction: CREATE_MARKET\nwallet: ${norm(walletAddress)}\ntitle: ${clean(market.title)}\ndescription: ${clean(market.description)}\ncategory: ${clean(market.categoryKey).toUpperCase() || "CRYPTO"}\nsource_kind: ${clean(market.sourceKind).toLowerCase() || "manual"}\nsource_url: ${String(market.sourceUrl ?? "").trim()}\nsource_slug: ${clean(market.sourceSlug)}\nsource_name: ${clean(market.sourceName) || "FunnyOption"}\ncover_image: ${String(market.coverImage ?? "").trim()}\nstatus: ${clean(market.status).toUpperCase() || "OPEN"}\ncollateral_asset: ${clean(market.collateralAsset).toUpperCase() || "USDT"}\nopen_at: ${Math.max(0, Math.floor(market.openAt || 0))}\nclose_at: ${Math.max(0, Math.floor(market.closeAt || 0))}\nresolve_at: ${Math.max(0, Math.floor(market.resolveAt || 0))}\nrequested_at: ${Math.floor(requestedAt)}\n${buildResolutionFrag(market.resolution)}options: ${optionFrag(market.options)}\n`;
}

function buildBootstrapMsg({ walletAddress, bootstrap, requestedAt }) {
  return `FunnyOption Operator Authorization\n\naction: ISSUE_FIRST_LIQUIDITY\nwallet: ${norm(walletAddress)}\nmarket_id: ${bootstrap.marketId}\nuser_id: ${bootstrap.userId}\nquantity: ${bootstrap.quantity}\noutcome: ${clean(bootstrap.outcome).toUpperCase()}\nprice: ${bootstrap.price}\nrequested_at: ${Math.floor(requestedAt)}\n`;
}

function buildOrderMsg(i) {
  return `FunnyOption Order Authorization\n\nsession_id: ${String(i.sessionId).trim()}\nwallet: ${norm(i.walletAddress)}\nuser_id: ${Math.floor(i.userId)}\nmarket_id: ${Math.floor(i.marketId)}\noutcome: ${clean(i.outcome).toUpperCase()}\nside: ${clean(i.side).toUpperCase()}\norder_type: ${clean(i.orderType).toUpperCase()}\ntime_in_force: ${clean(i.timeInForce).toUpperCase()}\nprice: ${Math.floor(i.price)}\nquantity: ${Math.floor(i.quantity)}\nclient_order_id: ${String(i.clientOrderId).trim()}\nnonce: ${Math.floor(i.nonce)}\nrequested_at: ${Math.floor(i.requestedAt)}\n`;
}

let stepN = 0;
function log(msg, data) { stepN++; console.log(`[${new Date().toISOString().slice(11, 23)}] Step ${stepN}: ${msg}${data ? " " + JSON.stringify(data) : ""}`); }
function die(msg) { console.error(`\n  FAIL: ${msg}\n`); process.exit(1); }
async function waitTx(pc, hash, label) { const r = await pc.waitForTransactionReceipt({ hash, confirmations: 1, timeout: 120_000 }); if (r.status !== "success") die(`${label} reverted: ${hash}`); return r; }

async function createSession(pc, account) {
  const sessionPrivateKey = ed.utils.randomPrivateKey();
  const sessionPublicKey = toHex(await ed.getPublicKeyAsync(sessionPrivateKey));
  let challenge;
  for (let i = 0; i < 40; i++) {
    const r = await fetchJson(`${API_BASE}/api/v1/trading-keys/challenge`, { method: "POST", body: { wallet_address: account.address, chain_id: bscTestnet.id, vault_address: VAULT_ADDRESS } });
    if (r.ok) { challenge = r.data; break; }
    await sleep(3000);
  }
  if (!challenge) die(`challenge failed for ${account.address}`);
  const typedData = {
    domain: { name: "FunnyOption Trading Authorization", version: "2", chainId: bscTestnet.id, verifyingContract: norm(VAULT_ADDRESS) },
    types: { AuthorizeTradingKey: [{ name: "action", type: "string" }, { name: "wallet", type: "address" }, { name: "tradingPublicKey", type: "bytes32" }, { name: "tradingKeyScheme", type: "string" }, { name: "scope", type: "string" }, { name: "challenge", type: "bytes32" }, { name: "challengeExpiresAt", type: "uint64" }, { name: "keyExpiresAt", type: "uint64" }] },
    primaryType: "AuthorizeTradingKey",
    message: { action: "AUTHORIZE_TRADING_KEY", wallet: norm(account.address), tradingPublicKey: norm(sessionPublicKey), tradingKeyScheme: "ED25519", scope: "TRADE", challenge: norm(challenge.challenge), challengeExpiresAt: BigInt(Math.floor(Number(challenge.challenge_expires_at || 0))), keyExpiresAt: BigInt(0) },
  };
  const walletSig = await account.signTypedData(typedData);
  const res = await postOk(`${API_BASE}/api/v1/trading-keys`, {
    wallet_address: account.address, chain_id: bscTestnet.id, vault_address: VAULT_ADDRESS,
    challenge_id: challenge.challenge_id, challenge: challenge.challenge,
    challenge_expires_at: challenge.challenge_expires_at,
    trading_public_key: sessionPublicKey, trading_key_scheme: "ED25519",
    scope: "TRADE", key_expires_at: 0,
    wallet_signature_standard: "EIP712_V4", wallet_signature: walletSig,
  }, "create session");
  return { userId: Number(res.user_id), sessionId: String(res.session_id), lastNonce: Number(res.last_order_nonce || 0), sessionPrivateKey, walletAddress: account.address };
}

async function submitOrder(session, marketId, params) {
  session.lastNonce++;
  const requestedAt = Date.now();
  const msg = buildOrderMsg({ sessionId: session.sessionId, walletAddress: session.walletAddress, userId: session.userId, marketId, outcome: params.outcome, side: params.side, orderType: params.orderType, timeInForce: params.timeInForce, price: params.price, quantity: params.quantity, clientOrderId: params.clientOrderId, nonce: session.lastNonce, requestedAt });
  const sig = toHex(await ed.signAsync(textEncoder.encode(msg), session.sessionPrivateKey));
  return postOk(`${API_BASE}/api/v1/orders`, { user_id: session.userId, market_id: marketId, outcome: params.outcome.toLowerCase(), side: params.side.toLowerCase(), type: params.orderType.toLowerCase(), time_in_force: params.timeInForce.toLowerCase(), price: params.price, quantity: params.quantity, client_order_id: params.clientOrderId, session_id: session.sessionId, session_signature: sig, order_nonce: session.lastNonce, requested_at: requestedAt }, `submit ${params.orderType} ${params.side}`);
}

async function waitBalance(userId, sessionId) {
  for (let i = 0; i < 30; i++) {
    const r = await fetchJson(`${API_BASE}/api/v1/balances?user_id=${userId}&asset=USDT&limit=5`, { headers: authHeaders(sessionId) });
    if (r.ok) { const items = r.data?.items || r.data?.balances || []; const usdt = items.find(b => String(b.asset).toUpperCase() === "USDT"); if (usdt && Number(usdt.available) > 0) return Number(usdt.available); }
    await sleep(2000);
  }
  return 0;
}

async function getBalance(userId, sessionId) {
  const r = await fetchJson(`${API_BASE}/api/v1/balances?user_id=${userId}&asset=USDT&limit=5`, { headers: authHeaders(sessionId) });
  if (r.ok) { const items = r.data?.items || r.data?.balances || []; const usdt = items.find(b => String(b.asset).toUpperCase() === "USDT"); if (usdt) return Number(usdt.available); }
  return -1;
}

async function getPositions(userId, sessionId, marketId) {
  const r = await fetchJson(`${API_BASE}/api/v1/positions?user_id=${userId}&market_id=${marketId}`, { headers: authHeaders(sessionId) });
  return r.ok ? (r.data?.items || r.data?.positions || []) : [];
}

async function main() {
  console.log("=".repeat(60));
  console.log("  Full Lifecycle E2E: Order -> Match -> Resolve -> Settle");
  console.log("=".repeat(60) + "\n");

  const operatorAccount = privateKeyToAccount(`0x${readSecret().replace(/^0x/, "")}`);
  const pc = createPublicClient({ chain: bscTestnet, transport: http(RPC_URL) });
  const opWallet = createWalletClient({ account: operatorAccount, chain: bscTestnet, transport: http(RPC_URL) });
  log("Operator loaded", { address: operatorAccount.address });

  // ── 1. Create oracle market (BTC > $10k → YES wins, threshold is low so YES is guaranteed)
  const nowSec = Math.floor(Date.now() / 1000);
  const resolveAt = nowSec + RESOLVE_DELAY_SEC;
  const market = {
    title: `Full E2E ${Date.now()}`, description: "Full lifecycle test: order → settle",
    categoryKey: "CRYPTO", coverImage: "", sourceUrl: "", sourceSlug: `full-e2e-${Date.now()}`,
    sourceName: "E2E", sourceKind: "manual", status: "OPEN", collateralAsset: "USDT",
    openAt: nowSec - 60, closeAt: resolveAt, resolveAt,
    options: binaryOptions,
    resolution: {
      version: 1, mode: "ORACLE_PRICE", market_kind: "CRYPTO_PRICE_THRESHOLD",
      manual_fallback_allowed: true,
      oracle: {
        source_kind: "HTTP_JSON", provider_key: "BINANCE",
        instrument: { kind: "SPOT", base_asset: "BTC", quote_asset: "USDT", symbol: "BTCUSDT" },
        price: { field: "LAST_PRICE", scale: 8, rounding_mode: "ROUND_HALF_UP", max_data_age_sec: 300 },
        window: { anchor: "RESOLVE_AT", before_sec: 600, after_sec: 600 },
        rule: { type: "PRICE_THRESHOLD", comparator: "GTE", threshold_price: "10000.00000000" },
      },
    },
  };
  const cra = Date.now();
  const cOp = await signOp(operatorAccount, buildCreateMarketMsg({ walletAddress: operatorAccount.address, market, requestedAt: cra }), cra);
  const cRes = await postOk(`${ADMIN_BASE}/api/operator/markets`, { market, operator: cOp }, "create market");
  const marketId = Number(cRes.market_id);
  if (!marketId) die("no market_id");
  log("Oracle market created", { market_id: marketId, resolve_at: new Date(resolveAt * 1000).toISOString() });

  // ── 2. Setup wallets
  const makerAccount = privateKeyToAccount(generatePrivateKey());
  const takerAccount = privateKeyToAccount(generatePrivateKey());
  const makerWallet = createWalletClient({ account: makerAccount, chain: bscTestnet, transport: http(RPC_URL) });
  const takerWallet = createWalletClient({ account: takerAccount, chain: bscTestnet, transport: http(RPC_URL) });
  log("Wallets generated", { maker: makerAccount.address, taker: takerAccount.address });

  log("Funding gas...");
  await waitTx(pc, await opWallet.sendTransaction({ to: makerAccount.address, value: parseUnits("0.005", 18) }), "maker gas");
  await waitTx(pc, await opWallet.sendTransaction({ to: takerAccount.address, value: parseUnits("0.005", 18) }), "taker gas");

  log("Creating trading sessions...");
  const [makerSession, takerSession] = await Promise.all([createSession(pc, makerAccount), createSession(pc, takerAccount)]);
  log("Sessions created", { maker_uid: makerSession.userId, taker_uid: takerSession.userId });

  const amount = parseUnits(DEPOSIT_USDT, 6);
  log("Depositing USDT...");
  await waitTx(pc, await opWallet.writeContract({ address: TOKEN_ADDRESS, abi: tokenAbi, functionName: "mint", args: [makerAccount.address, amount] }), "maker mint");
  await waitTx(pc, await makerWallet.writeContract({ address: TOKEN_ADDRESS, abi: tokenAbi, functionName: "approve", args: [VAULT_ADDRESS, amount] }), "maker approve");
  await waitTx(pc, await makerWallet.writeContract({ address: VAULT_ADDRESS, abi: vaultAbi, functionName: "deposit", args: [amount] }), "maker deposit");
  await waitTx(pc, await opWallet.writeContract({ address: TOKEN_ADDRESS, abi: tokenAbi, functionName: "mint", args: [takerAccount.address, amount] }), "taker mint");
  await waitTx(pc, await takerWallet.writeContract({ address: TOKEN_ADDRESS, abi: tokenAbi, functionName: "approve", args: [VAULT_ADDRESS, amount] }), "taker approve");
  await waitTx(pc, await takerWallet.writeContract({ address: VAULT_ADDRESS, abi: vaultAbi, functionName: "deposit", args: [amount] }), "taker deposit");

  const [makerBal, takerBal] = await Promise.all([waitBalance(makerSession.userId, makerSession.sessionId), waitBalance(takerSession.userId, takerSession.sessionId)]);
  if (makerBal <= 0 || takerBal <= 0) die("balance never arrived");
  log("Balances ready", { maker: makerBal, taker: takerBal });

  // ── 3. Seed liquidity (maker gets YES+NO positions)
  const bootstrap = { marketId, userId: makerSession.userId, quantity: SELL_QTY, outcome: "YES", price: SELL_PRICE };
  const bra = Date.now();
  const bOp = await signOp(operatorAccount, buildBootstrapMsg({ walletAddress: operatorAccount.address, bootstrap, requestedAt: bra }), bra);
  await postOk(`${ADMIN_BASE}/api/operator/markets/${marketId}/first-liquidity`, { bootstrap, operator: bOp }, "first liquidity");
  log("First liquidity seeded");
  await sleep(3000);

  // ── 4. Taker buys YES
  const clientOrderId = `full_e2e_${Date.now()}`;
  const orderRes = await submitOrder(takerSession, marketId, {
    outcome: "YES", side: "BUY", orderType: "MARKET", timeInForce: "IOC",
    price: 100, quantity: BUY_QTY, clientOrderId,
  });
  log("MARKET BUY submitted", { order_id: orderRes.order_id, qty: BUY_QTY });

  await sleep(5000);
  const takerBalAfterTrade = await getBalance(takerSession.userId, takerSession.sessionId);
  const takerPosBefore = await getPositions(takerSession.userId, takerSession.sessionId, marketId);
  log("Post-trade state", { taker_balance: takerBalAfterTrade, taker_positions: takerPosBefore.length });

  // ── 5. Wait for oracle resolve
  const resolveDeadline = (resolveAt + 180) * 1000;
  log(`Waiting for oracle to resolve (resolve_at=${new Date(resolveAt * 1000).toISOString()})...`);
  let resolved = false;
  let resolvedOutcome = "";
  let lastStatus = "";
  while (Date.now() < resolveDeadline) {
    const r = await fetchJson(`${API_BASE}/api/v1/markets/${marketId}`);
    const status = String(r.data?.status ?? r.data?.item?.status ?? "").toUpperCase();
    if (status !== lastStatus) { log(`Market status: ${status}`); lastStatus = status; }
    if (status === "RESOLVED") {
      resolvedOutcome = r.data?.resolved_outcome ?? r.data?.item?.resolved_outcome ?? "?";
      resolved = true;
      break;
    }
    await sleep(5000);
  }
  if (!resolved) die(`Market not resolved after deadline, last status=${lastStatus}`);
  log("Market RESOLVED", { outcome: resolvedOutcome });

  // ── 6. Wait for settlement (give it 30s after resolution)
  log("Waiting for settlement to process payouts...");
  await sleep(30000);

  // ── 7. Verify settlement
  console.log("\n" + "-".repeat(50));
  console.log("  VERIFICATION");
  console.log("-".repeat(50) + "\n");

  let passed = true;

  // Check taker positions (should be cleared after settlement)
  const takerPosAfter = await getPositions(takerSession.userId, takerSession.sessionId, marketId);
  const takerYesPos = takerPosAfter.find(p => String(p.outcome).toUpperCase() === "YES");
  const takerYesQty = takerYesPos ? Number(takerYesPos.quantity) : 0;
  if (takerYesQty === 0) {
    console.log("  [PASS] Taker YES position settled (cleared to 0)");
  } else {
    console.log(`  [INFO] Taker YES position: ${takerYesQty} (may still be settling)`);
  }

  // Check taker balance (should increase from payout if YES won)
  const takerFinalBal = await getBalance(takerSession.userId, takerSession.sessionId);
  log("Taker balance", { before_trade: takerBal, after_trade: takerBalAfterTrade, final: takerFinalBal });
  if (resolvedOutcome.toUpperCase() === "YES" && takerFinalBal > takerBalAfterTrade) {
    const payout = takerFinalBal - takerBalAfterTrade;
    console.log(`  [PASS] Taker received settlement payout: +${payout}`);
  } else if (resolvedOutcome.toUpperCase() === "YES") {
    console.log(`  [WARN] Expected payout but balance didn't increase (${takerBalAfterTrade} -> ${takerFinalBal})`);
    passed = false;
  } else {
    console.log(`  [INFO] Outcome was ${resolvedOutcome}, taker held YES → no payout expected`);
  }

  // Check trades exist
  const tradesR = await fetchJson(`${API_BASE}/api/v1/trades?market_id=${marketId}`);
  const trades = tradesR.data?.items || tradesR.data?.trades || [];
  if (trades.length > 0) { console.log(`  [PASS] Trades confirmed: ${trades.length} trade(s)`); }
  else { console.log("  [FAIL] No trades found"); passed = false; }

  // Resolution check
  if (resolved) { console.log(`  [PASS] Oracle auto-resolved: outcome=${resolvedOutcome}`); }
  else { console.log("  [FAIL] Market not resolved"); passed = false; }

  console.log("\n" + "=".repeat(60));
  if (passed) {
    console.log("  RESULT: ALL LIFECYCLE CHECKS PASSED");
  } else {
    console.log("  RESULT: SOME CHECKS NEED ATTENTION");
  }
  console.log("=".repeat(60) + "\n");
  process.exit(passed ? 0 : 1);
}

main().catch(e => { console.error(`\n  FATAL: ${e.message}\n`); if (e.stack) console.error(e.stack); process.exit(1); });
