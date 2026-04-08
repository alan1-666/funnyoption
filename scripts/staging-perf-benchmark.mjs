#!/usr/bin/env node

/**
 * Matching Engine V2 — Performance Benchmark
 *
 * Reuses the E2E auth flow (trading-key challenge → EIP-712 → ed25519 order intent).
 * Sends a burst of concurrent signed orders and measures throughput + latency.
 *
 * Usage:
 *   node scripts/staging-perf-benchmark.mjs [--orders N] [--concurrency C] [--users U]
 */

import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { setTimeout as sleep } from "node:timers/promises";
import * as ed from "../web/node_modules/@noble/ed25519/index.js";
import {
  createPublicClient,
  createWalletClient,
  http,
  parseAbi,
  parseEther,
  parseUnits,
} from "../web/node_modules/viem/_esm/index.js";
import { bscTestnet } from "../web/node_modules/viem/_esm/chains/index.js";
import { generatePrivateKey, privateKeyToAccount } from "../web/node_modules/viem/_esm/accounts/index.js";

const SECRET_FILE = fileURLToPath(new URL("../.secrets", import.meta.url));
const API_BASE   = process.env.API_BASE   || "https://funnyoption.xyz";
const ADMIN_BASE = process.env.ADMIN_BASE || "https://admin.funnyoption.xyz";
const RPC_URL    = "https://data-seed-prebsc-1-s1.bnbchain.org:8545";
const TOKEN_ADDR = "0x756D0b1AF00C0e2447cb5c891A838B508Df5ff43";
const VAULT_ADDR = "0xf47e6e19DC896ff8C9137C19782eb22411d0d1Bb";
const CHAIN_ID   = bscTestnet.id;

const textEncoder = new TextEncoder();
const toHex = (bytes) => `0x${Buffer.from(bytes).toString("hex")}`;
function normalizeAddress(a) {
  return String(a ?? "").trim().toLowerCase();
}
const cleanText = (s) => (s == null ? "" : String(s).trim());

const tokenAbi = parseAbi([
  "function owner() view returns (address)",
  "function balanceOf(address) view returns (uint256)",
  "function mint(address,uint256) returns (bool)",
  "function approve(address,uint256) returns (bool)"
]);
const vaultAbi = parseAbi(["function deposit(uint256 amount)"]);

// ─── CLI ────────────────────────────────────────────────────────────
function parseArgs() {
  const args = process.argv.slice(2);
  const o = { orders: 50, concurrency: 10, users: 4, price: 50 };
  for (let i = 0; i < args.length; i += 2) {
    const k = args[i], v = args[i + 1];
    if (k === "--orders")      o.orders = +v;
    if (k === "--concurrency") o.concurrency = +v;
    if (k === "--users")       o.users = +v;
    if (k === "--price")       o.price = +v;
  }
  return o;
}

// ─── Secrets ────────────────────────────────────────────────────────
function readSecret(key) {
  const raw = readFileSync(SECRET_FILE, "utf-8");
  for (const line of raw.split(/\r?\n/)) {
    const t = line.trim();
    if (t.startsWith(`${key}:`)) return t.slice(key.length + 1).trim();
    if (t.startsWith(`${key}=`)) return t.slice(key.length + 1).trim();
  }
  throw new Error(`secret ${key} not found in ${SECRET_FILE}`);
}

// ─── HTTP ───────────────────────────────────────────────────────────
async function fetchJson(url, init = {}) {
  const res = await fetch(url, { ...init, signal: AbortSignal.timeout(15_000) });
  const text = await res.text();
  if (!res.ok) throw new Error(`HTTP ${res.status} ${url}: ${text.slice(0, 200)}`);
  return text ? JSON.parse(text) : {};
}

async function postJson(url, body, headers = {}) {
  return fetchJson(url, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...headers },
    body: JSON.stringify(body)
  });
}

// ─── Auth: trading-key challenge + register ─────────────────────────
function buildTradingKeyTypedData(input) {
  return {
    domain: {
      name: "FunnyOption Trading Authorization",
      version: "2",
      chainId: CHAIN_ID,
      verifyingContract: normalizeAddress(VAULT_ADDR)
    },
    types: {
      AuthorizeTradingKey: [
        { name: "action", type: "string" },
        { name: "wallet", type: "address" },
        { name: "tradingPublicKey", type: "bytes32" },
        { name: "tradingKeyScheme", type: "string" },
        { name: "scope", type: "string" },
        { name: "challenge", type: "bytes32" },
        { name: "challengeExpiresAt", type: "uint64" },
        { name: "keyExpiresAt", type: "uint64" },
      ]
    },
    primaryType: "AuthorizeTradingKey",
    message: {
      action: "AUTHORIZE_TRADING_KEY",
      wallet: normalizeAddress(input.walletAddress),
      tradingPublicKey: normalizeAddress(input.tradingPublicKey),
      tradingKeyScheme: "ED25519",
      scope: "TRADE",
      challenge: normalizeAddress(input.challenge),
      challengeExpiresAt: BigInt(Math.floor(Number(input.challengeExpiresAt || 0))),
      keyExpiresAt: BigInt(0),
    }
  };
}

async function createSession(account) {
  const sessionPriv = ed.utils.randomPrivateKey();
  const sessionPub = toHex(await ed.getPublicKeyAsync(sessionPriv));

  const challenge = await postJson(`${API_BASE}/api/v1/trading-keys/challenge`, {
    wallet_address: account.address,
    chain_id: CHAIN_ID,
    vault_address: VAULT_ADDR,
  });

  const typedData = buildTradingKeyTypedData({
    walletAddress: account.address,
    tradingPublicKey: sessionPub,
    challenge: challenge.challenge,
    challengeExpiresAt: challenge.challenge_expires_at,
  });
  const walletSig = await account.signTypedData(typedData);

  const result = await postJson(`${API_BASE}/api/v1/trading-keys`, {
    wallet_address: account.address,
    chain_id: CHAIN_ID,
    vault_address: VAULT_ADDR,
    challenge_id: challenge.challenge_id,
    challenge: challenge.challenge,
    challenge_expires_at: challenge.challenge_expires_at,
    trading_public_key: sessionPub,
    trading_key_scheme: "ED25519",
    scope: "TRADE",
    key_expires_at: 0,
    wallet_signature_standard: "EIP712_V4",
    wallet_signature: walletSig,
  });

  return {
    sessionId: String(result.session_id || ""),
    userId: Number(result.user_id || 0),
    walletAddress: account.address,
    lastOrderNonce: Number(result.last_order_nonce || 0),
    sessionPriv,
  };
}

// ─── Order signing (ed25519 intent message) ─────────────────────────
function buildOrderIntentMessage(input) {
  return `FunnyOption Order Authorization

session_id: ${String(input.sessionId ?? "").trim()}
wallet: ${normalizeAddress(input.walletAddress)}
user_id: ${Math.floor(Number(input.userId || 0))}
market_id: ${Math.floor(Number(input.marketId || 0))}
outcome: ${cleanText(input.outcome).toUpperCase()}
side: ${cleanText(input.side).toUpperCase()}
order_type: ${cleanText(input.orderType).toUpperCase()}
time_in_force: ${cleanText(input.timeInForce).toUpperCase()}
price: ${Math.floor(Number(input.price || 0))}
quantity: ${Math.floor(Number(input.quantity || 0))}
client_order_id: ${String(input.clientOrderId ?? "").trim()}
nonce: ${Math.floor(Number(input.nonce || 0))}
requested_at: ${Math.floor(Number(input.requestedAt || 0))}
`;
}

async function submitOrder(session, marketId, side, price, qty, nonceOverride) {
  const requestedAt = Date.now();
  const nonce = nonceOverride || ++session.lastOrderNonce;
  const clientOrderId = `bench_${requestedAt}_${nonce}`;

  const message = buildOrderIntentMessage({
    sessionId: session.sessionId,
    walletAddress: session.walletAddress,
    userId: session.userId,
    marketId,
    outcome: "yes",
    side,
    orderType: "LIMIT",
    timeInForce: "GTC",
    price,
    quantity: qty,
    clientOrderId,
    nonce,
    requestedAt,
  });

  const sig = toHex(await ed.signAsync(textEncoder.encode(message), session.sessionPriv));

  const t0 = performance.now();
  const res = await postJson(`${API_BASE}/api/v1/orders`, {
    user_id: session.userId,
    market_id: marketId,
    outcome: "yes",
    side: side.toLowerCase(),
    type: "limit",
    time_in_force: "gtc",
    price,
    quantity: qty,
    client_order_id: clientOrderId,
    session_id: session.sessionId,
    session_signature: sig,
    order_nonce: nonce,
    requested_at: requestedAt,
  });
  return { orderId: res.order_id, latencyMs: performance.now() - t0 };
}

// ─── Helpers ────────────────────────────────────────────────────────
function percentile(arr, p) {
  if (!arr.length) return 0;
  return arr[Math.max(0, Math.ceil(p / 100 * arr.length) - 1)];
}

// ─── Main ───────────────────────────────────────────────────────────
async function main() {
  const opts = parseArgs();
  console.log("╔═══════════════════════════════════════════════════════╗");
  console.log("║   Matching Engine V2 — Performance Benchmark          ║");
  console.log("╚═══════════════════════════════════════════════════════╝");
  console.log(`  Orders: ${opts.orders}  Concurrency: ${opts.concurrency}  Users: ${opts.users}  Price: ${opts.price}`);
  console.log();

  let opKey = readSecret("bsc-testnet-operator.key");
  if (!opKey.startsWith("0x")) opKey = `0x${opKey}`;
  const operatorAccount = privateKeyToAccount(opKey);
  const publicClient = createPublicClient({ chain: bscTestnet, transport: http(RPC_URL) });
  const operatorWallet = createWalletClient({ account: operatorAccount, chain: bscTestnet, transport: http(RPC_URL) });

  // 1. Create & fund users
  console.log(`[1/5] Creating ${opts.users} users...`);
  const users = [];
  for (let i = 0; i < opts.users; i++) {
    try {
      const privKey = generatePrivateKey();
      const account = privateKeyToAccount(privKey);

      const tbnbTx = await operatorWallet.sendTransaction({ to: account.address, value: parseEther("0.02") });
      await publicClient.waitForTransactionReceipt({ hash: tbnbTx });

      const tokenOwner = await publicClient.readContract({ address: TOKEN_ADDR, abi: tokenAbi, functionName: "owner" });
      let mintWallet = operatorWallet;
      if (normalizeAddress(tokenOwner) !== normalizeAddress(operatorAccount.address)) {
        mintWallet = operatorWallet;
      }

      const mintTx = await mintWallet.writeContract({
        address: TOKEN_ADDR, abi: tokenAbi,
        functionName: "mint", args: [account.address, parseUnits("100000", 6)]
      });
      await publicClient.waitForTransactionReceipt({ hash: mintTx });

      const userWallet = createWalletClient({ account, chain: bscTestnet, transport: http(RPC_URL) });
      const approveTx = await userWallet.writeContract({
        address: TOKEN_ADDR, abi: tokenAbi,
        functionName: "approve", args: [VAULT_ADDR, parseUnits("100000", 6)]
      });
      await publicClient.waitForTransactionReceipt({ hash: approveTx });

      const depTx = await userWallet.writeContract({
        address: VAULT_ADDR, abi: vaultAbi,
        functionName: "deposit", args: [parseUnits("50000", 6)]
      });
      await publicClient.waitForTransactionReceipt({ hash: depTx });

      const session = await createSession(account);
      users.push({ account, session });
      console.log(`  user ${i + 1}/${opts.users} ready (userId=${session.userId})`);
    } catch (e) {
      console.log(`  user ${i + 1}/${opts.users} FAILED: ${e.message.slice(0, 120)}`);
    }
  }
  if (users.length < 2) throw new Error("Need at least 2 users");

  // 2. Use user-proposed market (fast path — avoids admin EIP-191 signing)
  console.log("[2/5] Creating benchmark market via proposal...");
  const proposer = users[0];
  const propRes = await postJson(`${API_BASE}/api/v1/markets/propose`, {
    title: `Perf Bench ${Date.now()}`,
    description: "Automated perf test",
    category_key: "CRYPTO",
    collateral_asset: "USDT",
    options: [
      { key: "YES", label: "Yes", short_label: "Y" },
      { key: "NO", label: "No", short_label: "N" },
    ],
  }, { Authorization: `Bearer ${proposer.session.sessionId}` });
  const marketId = propRes.market_id;
  console.log(`  proposed market_id = ${marketId}, auto-approving...`);

  // Auto-approve via admin API (use the already-tested approve endpoint)
  const approveRequestedAt = Date.now();
  const approveMsg = `approve_market:${marketId}:${approveRequestedAt}`;
  const approveSig = await operatorAccount.signMessage({ message: approveMsg });
  await postJson(`${ADMIN_BASE}/api/operator/markets/${marketId}/approve`, {
    operator: {
      walletAddress: operatorAccount.address,
      requestedAt: approveRequestedAt,
      signature: approveSig,
    },
  });
  console.log(`  market ${marketId} approved`);

  // 3. Wait for deposits and seed maker liquidity
  console.log("[3/5] Waiting for chain deposit propagation...");
  const maker = users[0];
  const makerQty = Math.ceil(opts.orders * 1.5);

  // Poll balance until on-chain deposits are recognized by chain listener
  for (let attempt = 0; attempt < 40; attempt++) {
    try {
      const bal = await fetchJson(
        `${API_BASE}/api/v1/balances?user_id=${maker.session.userId}&asset=USDT&limit=5`,
        { headers: { Authorization: `Bearer ${maker.session.sessionId}` } }
      );
      const items = bal.items || [];
      const usdtItem = items.find(i => String(i.asset || "").toUpperCase() === "USDT");
      const available = Number(usdtItem?.available ?? 0);
      if (available > 0) {
        console.log(`  balance detected: ${available} USDT (attempt ${attempt + 1})`);
        break;
      }
    } catch (e) {
      // ignore fetch errors during polling
    }
    if (attempt === 39) {
      console.log("  warning: balance still 0 after 200s polling, proceeding anyway...");
    } else {
      process.stdout.write(`  polling balance... attempt ${attempt + 1}/40\r`);
    }
    await sleep(5000);
  }

  console.log("  Seeding maker liquidity...");
  const batches = Math.ceil(makerQty / 100);
  let seeded = 0;
  for (let b = 0; b < batches; b++) {
    const qty = Math.min(100, makerQty - b * 100);
    try {
      await submitOrder(maker.session, marketId, "SELL", opts.price, qty);
      seeded += qty;
    } catch (e) {
      console.log(`  maker batch ${b} failed: ${e.message.slice(0, 100)}`);
    }
  }
  console.log(`  seeded ${seeded} sell qty at price ${opts.price}`);
  await sleep(3000);

  // 4. Burst taker orders — pre-assign nonces to avoid conflicts
  console.log(`[4/5] Sending ${opts.orders} taker BUY orders...`);
  const takerUsers = users.slice(1);
  const latencies = [];
  const errors = [];

  // Pre-assign nonces so concurrent orders from the same user don't collide
  const nonceCounters = new Map();
  for (const u of takerUsers) {
    nonceCounters.set(u.session.sessionId, u.session.lastOrderNonce);
  }

  function nextNonce(sessionId) {
    const n = (nonceCounters.get(sessionId) || 0) + 1;
    nonceCounters.set(sessionId, n);
    return n;
  }

  const burstStart = performance.now();
  const pool = [];

  for (let i = 0; i < opts.orders; i++) {
    const user = takerUsers[i % takerUsers.length];
    const nonce = nextNonce(user.session.sessionId);
    const p = submitOrder(user.session, marketId, "BUY", opts.price, 1, nonce)
      .then(r => latencies.push(r.latencyMs))
      .catch(e => errors.push(e.message.slice(0, 100)));
    pool.push(p);

    if (pool.length >= opts.concurrency) {
      await Promise.allSettled(pool.splice(0, opts.concurrency));
    }
  }
  if (pool.length) await Promise.allSettled(pool);

  // Update sessions' nonce counters
  for (const u of takerUsers) {
    u.session.lastOrderNonce = nonceCounters.get(u.session.sessionId) || u.session.lastOrderNonce;
  }

  const burstMs = performance.now() - burstStart;

  // 5. Report
  latencies.sort((a, b) => a - b);
  const placed = latencies.length;
  const throughput = placed / (burstMs / 1000);

  console.log();
  console.log("╔═══════════════════════════════════════════════════════╗");
  console.log("║                   BENCHMARK RESULTS                   ║");
  console.log("╠═══════════════════════════════════════════════════════╣");
  console.log(`║  Orders sent:         ${String(opts.orders).padStart(8)}`);
  console.log(`║  Placed OK:           ${String(placed).padStart(8)}`);
  console.log(`║  Failed:              ${String(errors.length).padStart(8)}`);
  console.log(`║  Burst duration:      ${burstMs.toFixed(0).padStart(6)}ms`);
  console.log(`║  Throughput:          ${throughput.toFixed(1).padStart(6)} orders/sec`);
  console.log("╠═══════════════════════════════════════════════════════╣");
  console.log("║  Latency (placement HTTP round-trip):                 ║");
  if (placed > 0) {
    console.log(`║    p50:               ${percentile(latencies, 50).toFixed(1).padStart(6)}ms`);
    console.log(`║    p95:               ${percentile(latencies, 95).toFixed(1).padStart(6)}ms`);
    console.log(`║    p99:               ${percentile(latencies, 99).toFixed(1).padStart(6)}ms`);
    console.log(`║    min:               ${latencies[0].toFixed(1).padStart(6)}ms`);
    console.log(`║    max:               ${latencies[latencies.length - 1].toFixed(1).padStart(6)}ms`);
    console.log(`║    mean:              ${(latencies.reduce((a, b) => a + b, 0) / placed).toFixed(1).padStart(6)}ms`);
  }
  console.log("╚═══════════════════════════════════════════════════════╝");

  if (errors.length) {
    console.log(`\nErrors (${errors.length}):`);
    for (const e of [...new Set(errors)].slice(0, 5)) console.log(`  - ${e}`);
  }

  console.log("\nPipeline stats: check matching service logs for 'pipeline stats'");
}

main().catch(err => { console.error("Benchmark failed:", err.message); process.exit(1); });
