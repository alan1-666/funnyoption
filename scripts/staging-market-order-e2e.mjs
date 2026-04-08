#!/usr/bin/env node

/**
 * Market Order E2E — Full Lifecycle on Staging
 *
 * Creates two wallets (maker + taker), deposits funds, then:
 *   1. Maker places a LIMIT SELL on YES at price=50
 *   2. Taker places a MARKET BUY on YES → matched as IOC@100 against the sell
 *   3. Verify: taker order FILLED, trade created, positions and balances updated
 *
 * Uses the operator wallet for first-liquidity so the maker has YES position to sell.
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

const DEPOSIT_USDT = "20";
const SELL_PRICE = 50;
const SELL_QTY = 3;
const MKT_BUY_QTY = 1;
const HTTP_TIMEOUT = 30_000;
const TX_TIMEOUT = 120_000;

const textEncoder = new TextEncoder();
const tokenAbi = parseAbi([
  "function mint(address to, uint256 amount) external",
  "function approve(address spender, uint256 amount) external returns (bool)",
  "function balanceOf(address owner) view returns (uint256)",
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

function readSecret() {
  if (process.env.FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY?.trim()) return process.env.FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY.trim();
  const raw = readFileSync(SECRET_FILE, "utf-8");
  for (const line of raw.split("\n")) { const [k, ...v] = line.split(":"); if (k?.trim() === "bsc-testnet-operator.key" && v.join(":").trim()) return v.join(":").trim(); }
  throw new Error("operator key not found");
}

async function fetchJson(url, { method = "GET", body, headers: extra } = {}) {
  const h = { ...extra }; if (body !== undefined) h["Content-Type"] = "application/json";
  for (let a = 1; a <= 3; a++) {
    try {
      const r = await fetch(url, { method, headers: Object.keys(h).length ? h : undefined, body: body === undefined ? undefined : JSON.stringify(body), signal: AbortSignal.timeout(HTTP_TIMEOUT) });
      const text = await r.text(); let data; try { data = JSON.parse(text); } catch { data = { raw: text }; }
      return { ok: r.ok, status: r.status, data, text };
    } catch (e) { if (a === 3) throw new Error(`fetch ${method} ${url}: ${e.message}`); await sleep(500 * a); }
  }
}
async function postOk(url, body, label) { const r = await fetchJson(url, { method: "POST", body }); if (!r.ok) die(`${label}: HTTP ${r.status} ${r.text}`); return r.data ?? {}; }

async function signOp(account, message, requestedAt) {
  return { walletAddress: account.address, requestedAt, signature: await account.signMessage({ message }) };
}

function buildCreateMarketMsg({ walletAddress, market, requestedAt }) {
  return `FunnyOption Operator Authorization\n\naction: CREATE_MARKET\nwallet: ${norm(walletAddress)}\ntitle: ${clean(market.title)}\ndescription: ${clean(market.description)}\ncategory: ${clean(market.categoryKey).toUpperCase() || "CRYPTO"}\nsource_kind: ${clean(market.sourceKind).toLowerCase() || "manual"}\nsource_url: ${String(market.sourceUrl ?? "").trim()}\nsource_slug: ${clean(market.sourceSlug)}\nsource_name: ${clean(market.sourceName) || "FunnyOption"}\ncover_image: ${String(market.coverImage ?? "").trim()}\nstatus: ${clean(market.status).toUpperCase() || "OPEN"}\ncollateral_asset: ${clean(market.collateralAsset).toUpperCase() || "USDT"}\nopen_at: ${Math.max(0, Math.floor(market.openAt || 0))}\nclose_at: ${Math.max(0, Math.floor(market.closeAt || 0))}\nresolve_at: ${Math.max(0, Math.floor(market.resolveAt || 0))}\nrequested_at: ${Math.floor(requestedAt)}\noptions: ${optionFrag(market.options)}\n`;
}

function buildBootstrapMsg({ walletAddress, bootstrap, requestedAt }) {
  return `FunnyOption Operator Authorization\n\naction: ISSUE_FIRST_LIQUIDITY\nwallet: ${norm(walletAddress)}\nmarket_id: ${bootstrap.marketId}\nuser_id: ${bootstrap.userId}\nquantity: ${bootstrap.quantity}\noutcome: ${clean(bootstrap.outcome).toUpperCase()}\nprice: ${bootstrap.price}\nrequested_at: ${Math.floor(requestedAt)}\n`;
}

function buildOrderMsg(i) {
  return `FunnyOption Order Authorization\n\nsession_id: ${String(i.sessionId).trim()}\nwallet: ${norm(i.walletAddress)}\nuser_id: ${Math.floor(i.userId)}\nmarket_id: ${Math.floor(i.marketId)}\noutcome: ${clean(i.outcome).toUpperCase()}\nside: ${clean(i.side).toUpperCase()}\norder_type: ${clean(i.orderType).toUpperCase()}\ntime_in_force: ${clean(i.timeInForce).toUpperCase()}\nprice: ${Math.floor(i.price)}\nquantity: ${Math.floor(i.quantity)}\nclient_order_id: ${String(i.clientOrderId).trim()}\nnonce: ${Math.floor(i.nonce)}\nrequested_at: ${Math.floor(i.requestedAt)}\n`;
}

let stepN = 0;
function log(msg, data) { stepN++; console.log(`[${new Date().toISOString().slice(11, 23)}] Step ${stepN}: ${msg}${data !== undefined ? " " + JSON.stringify(data) : ""}`); }
function die(msg) { console.error(`\n❌ FAIL: ${msg}\n`); process.exit(1); }
async function waitTx(pc, hash, label) { const r = await pc.waitForTransactionReceipt({ hash, confirmations: 1, timeout: TX_TIMEOUT }); if (r.status !== "success") die(`${label} reverted: ${hash}`); return r; }

async function createSession(pc, account, walletClient) {
  const sessionPrivateKey = ed.utils.randomPrivateKey();
  const sessionPublicKey = toHex(await ed.getPublicKeyAsync(sessionPrivateKey));

  let challenge;
  for (let i = 0; i < 40; i++) {
    const r = await fetchJson(`${API_BASE}/api/v1/trading-keys/challenge`, { method: "POST", body: { wallet_address: account.address, chain_id: bscTestnet.id, vault_address: VAULT_ADDRESS } });
    if (r.ok) { challenge = r.data; break; }
    await sleep(3000);
  }
  if (!challenge) die(`challenge never succeeded for ${account.address}`);

  const typedData = {
    domain: { name: "FunnyOption Trading Authorization", version: "2", chainId: bscTestnet.id, verifyingContract: norm(VAULT_ADDRESS) },
    types: { AuthorizeTradingKey: [
      { name: "action", type: "string" }, { name: "wallet", type: "address" },
      { name: "tradingPublicKey", type: "bytes32" }, { name: "tradingKeyScheme", type: "string" },
      { name: "scope", type: "string" }, { name: "challenge", type: "bytes32" },
      { name: "challengeExpiresAt", type: "uint64" }, { name: "keyExpiresAt", type: "uint64" },
    ]},
    primaryType: "AuthorizeTradingKey",
    message: {
      action: "AUTHORIZE_TRADING_KEY", wallet: norm(account.address),
      tradingPublicKey: norm(sessionPublicKey), tradingKeyScheme: "ED25519", scope: "TRADE",
      challenge: norm(challenge.challenge),
      challengeExpiresAt: BigInt(Math.floor(Number(challenge.challenge_expires_at || 0))),
      keyExpiresAt: BigInt(0),
    },
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

  return {
    userId: Number(res.user_id),
    sessionId: String(res.session_id),
    lastNonce: Number(res.last_order_nonce || 0),
    sessionPrivateKey,
    walletAddress: account.address,
  };
}

async function submitOrder(session, marketId, params) {
  session.lastNonce++;
  const requestedAt = Date.now();
  const msg = buildOrderMsg({
    sessionId: session.sessionId, walletAddress: session.walletAddress,
    userId: session.userId, marketId,
    outcome: params.outcome, side: params.side,
    orderType: params.orderType, timeInForce: params.timeInForce,
    price: params.price, quantity: params.quantity,
    clientOrderId: params.clientOrderId, nonce: session.lastNonce, requestedAt,
  });
  const sig = toHex(await ed.signAsync(textEncoder.encode(msg), session.sessionPrivateKey));
  return postOk(`${API_BASE}/api/v1/orders`, {
    user_id: session.userId, market_id: marketId,
    outcome: params.outcome.toLowerCase(), side: params.side.toLowerCase(),
    type: params.orderType.toLowerCase(), time_in_force: params.timeInForce.toLowerCase(),
    price: params.price, quantity: params.quantity,
    client_order_id: params.clientOrderId,
    session_id: session.sessionId, session_signature: sig,
    order_nonce: session.lastNonce, requested_at: requestedAt,
  }, `submit ${params.orderType} ${params.side}`);
}

function authHeaders(sessionId) {
  return sessionId ? { Authorization: `Bearer ${sessionId}` } : {};
}

async function waitBalance(userId, sessionId) {
  for (let i = 0; i < 30; i++) {
    const r = await fetchJson(`${API_BASE}/api/v1/balances?user_id=${userId}&asset=USDT&limit=5`, { headers: authHeaders(sessionId) });
    if (r.ok) {
      const items = r.data?.items || r.data?.balances || [];
      const usdt = items.find(b => String(b.asset).toUpperCase() === "USDT");
      if (usdt && Number(usdt.available) > 0) return Number(usdt.available);
    }
    await sleep(2000);
  }
  return 0;
}

async function main() {
  console.log("╔═══════════════════════════════════════════════╗");
  console.log("║  Market Order E2E — Full Lifecycle on Staging  ║");
  console.log("╚═══════════════════════════════════════════════╝\n");

  const operatorAccount = privateKeyToAccount(`0x${readSecret().replace(/^0x/, "")}`);
  const pc = createPublicClient({ chain: bscTestnet, transport: http(RPC_URL) });
  const opWallet = createWalletClient({ account: operatorAccount, chain: bscTestnet, transport: http(RPC_URL) });
  log("Operator loaded", { address: operatorAccount.address });

  // ── 1. Create market ──
  const nowSec = Math.floor(Date.now() / 1000);
  const market = {
    title: `MktOrder E2E ${Date.now()}`, description: "Market order lifecycle test",
    categoryKey: "CRYPTO", coverImage: "", sourceUrl: "", sourceSlug: `mkt-e2e-${Date.now()}`,
    sourceName: "E2E", sourceKind: "manual", status: "OPEN", collateralAsset: "USDT",
    openAt: nowSec - 60, closeAt: nowSec + 3600, resolveAt: nowSec + 7200, options: binaryOptions,
  };
  const cra = Date.now();
  const cOp = await signOp(operatorAccount, buildCreateMarketMsg({ walletAddress: operatorAccount.address, market, requestedAt: cra }), cra);
  const cRes = await postOk(`${ADMIN_BASE}/api/operator/markets`, { market, operator: cOp }, "create market");
  const marketId = Number(cRes.market_id);
  if (!marketId) die("no market_id in response");
  log("Market created", { market_id: marketId });

  // ── 2. Create maker + taker wallets ──
  const makerKey = generatePrivateKey();
  const makerAccount = privateKeyToAccount(makerKey);
  const makerWalletClient = createWalletClient({ account: makerAccount, chain: bscTestnet, transport: http(RPC_URL) });

  const takerKey = generatePrivateKey();
  const takerAccount = privateKeyToAccount(takerKey);
  const takerWalletClient = createWalletClient({ account: takerAccount, chain: bscTestnet, transport: http(RPC_URL) });

  log("Wallets generated", { maker: makerAccount.address, taker: takerAccount.address });

  // Fund gas first (sequential because operator nonce cannot overlap)
  log("Funding gas for maker...");
  await waitTx(pc, await opWallet.sendTransaction({ to: makerAccount.address, value: parseUnits("0.005", 18) }), "maker gas");
  log("Funding gas for taker...");
  await waitTx(pc, await opWallet.sendTransaction({ to: takerAccount.address, value: parseUnits("0.005", 18) }), "taker gas");

  // ── 3. Create sessions BEFORE deposit (wallet must be bound first) ──
  log("Creating trading sessions (registers wallets in system)...");
  // The challenge endpoint auto-creates the wallet binding, even without balance.
  // We need to call it to establish the user before depositing.
  const [makerSession, takerSession] = await Promise.all([
    createSession(pc, makerAccount, makerWalletClient),
    createSession(pc, takerAccount, takerWalletClient),
  ]);
  log("Maker session", { user_id: makerSession.userId, session_id: makerSession.sessionId });
  log("Taker session", { user_id: takerSession.userId, session_id: takerSession.sessionId });

  // Now do on-chain deposits (chain listener will recognize the wallets)
  const amount = parseUnits(DEPOSIT_USDT, 6);
  log("Minting and depositing for maker...");
  await waitTx(pc, await opWallet.writeContract({ address: TOKEN_ADDRESS, abi: tokenAbi, functionName: "mint", args: [makerAccount.address, amount] }), "maker mint");
  await waitTx(pc, await makerWalletClient.writeContract({ address: TOKEN_ADDRESS, abi: tokenAbi, functionName: "approve", args: [VAULT_ADDRESS, amount] }), "maker approve");
  await waitTx(pc, await makerWalletClient.writeContract({ address: VAULT_ADDRESS, abi: vaultAbi, functionName: "deposit", args: [amount] }), "maker deposit");

  log("Minting and depositing for taker...");
  await waitTx(pc, await opWallet.writeContract({ address: TOKEN_ADDRESS, abi: tokenAbi, functionName: "mint", args: [takerAccount.address, amount] }), "taker mint");
  await waitTx(pc, await takerWalletClient.writeContract({ address: TOKEN_ADDRESS, abi: tokenAbi, functionName: "approve", args: [VAULT_ADDRESS, amount] }), "taker approve");
  await waitTx(pc, await takerWalletClient.writeContract({ address: VAULT_ADDRESS, abi: vaultAbi, functionName: "deposit", args: [amount] }), "taker deposit");
  log("On-chain deposits complete, waiting for chain listener...");

  // Wait for balances
  const [makerBal, takerBal] = await Promise.all([
    waitBalance(makerSession.userId, makerSession.sessionId),
    waitBalance(takerSession.userId, takerSession.sessionId),
  ]);
  if (makerBal <= 0) die("maker balance never became positive");
  if (takerBal <= 0) die("taker balance never became positive");
  log("Balances ready", { maker: makerBal, taker: takerBal });

  // ── 4. Seed first liquidity (maker gets YES+NO positions) ──
  const bootstrap = { marketId, userId: makerSession.userId, quantity: SELL_QTY, outcome: "YES", price: SELL_PRICE };
  const bra = Date.now();
  const bOp = await signOp(operatorAccount, buildBootstrapMsg({ walletAddress: operatorAccount.address, bootstrap, requestedAt: bra }), bra);
  const bRes = await postOk(`${ADMIN_BASE}/api/operator/markets/${marketId}/first-liquidity`, { bootstrap, operator: bOp }, "first liquidity");
  log("First liquidity seeded", { first_liquidity_id: bRes.first_liquidity_id, order_id: bRes.order_id });

  await sleep(3000);

  // ── 5. Taker places MARKET BUY on YES ──
  const clientOrderId = `mkt_e2e_${Date.now()}`;
  const orderRes = await submitOrder(takerSession, marketId, {
    outcome: "YES", side: "BUY", orderType: "MARKET", timeInForce: "IOC",
    price: 100, quantity: MKT_BUY_QTY, clientOrderId,
  });
  const orderId = orderRes.order_id || orderRes.id;
  log("MARKET BUY submitted", { order_id: orderId, qty: MKT_BUY_QTY });

  // ── 6. Verify ──
  console.log("\n── Verification ──\n");

  let filled = false;
  let marketOrder = null;
  for (let i = 0; i < 10; i++) {
    await sleep(3000);
    const r = await fetchJson(`${API_BASE}/api/v1/orders?user_id=${takerSession.userId}&market_id=${marketId}`, { headers: authHeaders(takerSession.sessionId) });
    if (r.ok) {
      const orders = r.data?.items || r.data?.orders || [];
      marketOrder = orders.find(o => String(o.client_order_id) === clientOrderId);
      if (marketOrder) {
        const st = String(marketOrder.status).toUpperCase();
        if (st === "FILLED" || st === "CANCELLED" || st === "EXPIRED") break;
      }
    }
  }

  let passed = true;
  if (marketOrder) {
    const st = String(marketOrder.status).toUpperCase();
    filled = st === "FILLED";
    log("Order status", { order_id: marketOrder.order_id, status: st, filled_quantity: marketOrder.filled_quantity, remaining_quantity: marketOrder.remaining_quantity });
    if (filled) {
      console.log(`  ✅ MARKET order FILLED`);
    } else {
      console.log(`  ⚠️  Unexpected status: ${st}`);
      passed = false;
    }
  } else {
    console.log("  ⚠️  Order not found (still processing?)");
    passed = false;
  }

  // Trades
  const tradesR = await fetchJson(`${API_BASE}/api/v1/trades?market_id=${marketId}`);
  const trades = tradesR.data?.items || tradesR.data?.trades || [];
  log("Trades", { count: trades.length });
  for (const t of trades.slice(0, 5)) {
    console.log(`    trade_id=${t.trade_id}  price=${t.price}  qty=${t.quantity}  taker=${t.taker_side}  outcome=${t.outcome}`);
  }
  if (trades.length > 0) console.log("  ✅ Trade execution confirmed");
  else { console.log("  ⚠️  No trades"); passed = false; }

  // Position
  const posR = await fetchJson(`${API_BASE}/api/v1/positions?user_id=${takerSession.userId}&market_id=${marketId}`, { headers: authHeaders(takerSession.sessionId) });
  const positions = posR.data?.items || posR.data?.positions || [];
  log("Taker positions", { count: positions.length });
  for (const p of positions) console.log(`    outcome=${p.outcome}  qty=${p.quantity}  avg_price=${p.avg_price ?? p.average_price}`);
  if (positions.length > 0) console.log("  ✅ Position created");

  // Final balance
  const finalBal = await waitBalance(takerSession.userId, takerSession.sessionId);
  log("Taker final balance", { initial: takerBal, final: finalBal, spent: takerBal - finalBal });
  if (finalBal < takerBal) console.log(`  ✅ Balance decreased: ${takerBal} → ${finalBal}`);

  console.log("\n══════════════════════════════════════════════════");
  if (passed) {
    console.log("  ✅ Market Order E2E: ALL CHECKS PASSED");
  } else {
    console.log("  ❌ Market Order E2E: SOME CHECKS FAILED");
  }
  console.log("══════════════════════════════════════════════════\n");
  process.exit(passed ? 0 : 1);
}

main().catch(e => { console.error(`\n❌ ${e.message}\n`); if (e.stack) console.error(e.stack); process.exit(1); });
