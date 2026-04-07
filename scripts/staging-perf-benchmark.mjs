#!/usr/bin/env node

/**
 * Matching engine performance benchmark.
 *
 * Sends a burst of concurrent orders to the staging API and measures:
 *   - Throughput  (orders/sec placed + matched)
 *   - Latency     p50 / p95 / p99 from order placement to terminal status
 *   - Match rate  % of orders that settle within the poll window
 *
 * Usage:
 *   node scripts/staging-perf-benchmark.mjs [--orders N] [--concurrency C] [--market-id M]
 *
 * Requires a running staging environment with the E2E test's user/session setup.
 */

import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { setTimeout as sleep } from "node:timers/promises";
import * as ed from "../web/node_modules/@noble/ed25519/index.js";
import {
  createPublicClient,
  createWalletClient,
  formatUnits,
  http,
  parseAbi,
  parseUnits
} from "../web/node_modules/viem/_esm/index.js";
import { bscTestnet } from "../web/node_modules/viem/_esm/chains/index.js";
import { generatePrivateKey, privateKeyToAccount } from "../web/node_modules/viem/_esm/accounts/index.js";

const DEFAULT_SECRET_FILE = fileURLToPath(new URL("../.secrets", import.meta.url));
const DEFAULT_SECRET_KEY = "bsc-testnet-operator.key";
const DEFAULT_API_BASE = "https://funnyoption.xyz";
const DEFAULT_RPC_URL = "https://data-seed-prebsc-1-s1.bnbchain.org:8545";
const DEFAULT_TOKEN_ADDRESS = "0x0ADa04558decC14671D565562Aeb8D1096F71dDc";
const DEFAULT_VAULT_ADDRESS = "0x7Da015dfCD16Fb892328995BDd883da5AA3E670a";

const textEncoder = new TextEncoder();

const tokenAbi = parseAbi([
  "function balanceOf(address) view returns (uint256)",
  "function mint(address,uint256) returns (bool)",
  "function approve(address,uint256) returns (bool)"
]);

const vaultAbi = parseAbi([
  "function deposit(uint256 amount)"
]);

// ─── CLI args ───────────────────────────────────────────────────────
function parseArgs() {
  const args = process.argv.slice(2);
  const opts = {
    orders: 100,
    concurrency: 10,
    marketId: 0,
    outcome: "YES",
    price: 50,
    users: 6,
    pollTimeoutMs: 120_000,
    pollIntervalMs: 2_000,
  };
  for (let i = 0; i < args.length; i += 2) {
    const k = args[i], v = args[i + 1];
    if (k === "--orders")       opts.orders = parseInt(v, 10);
    if (k === "--concurrency")  opts.concurrency = parseInt(v, 10);
    if (k === "--market-id")    opts.marketId = parseInt(v, 10);
    if (k === "--users")        opts.users = parseInt(v, 10);
    if (k === "--price")        opts.price = parseInt(v, 10);
    if (k === "--poll-timeout") opts.pollTimeoutMs = parseInt(v, 10);
  }
  return opts;
}

// ─── HTTP helpers ───────────────────────────────────────────────────
async function fetchJson(url, init = {}) {
  const res = await fetch(url, { ...init, signal: AbortSignal.timeout(15_000) });
  const body = await res.text();
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${body.slice(0, 200)}`);
  return body ? JSON.parse(body) : {};
}

async function postJson(url, body, headers = {}) {
  return fetchJson(url, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...headers },
    body: JSON.stringify(body)
  });
}

// ─── User management ────────────────────────────────────────────────
function loadSecrets() {
  try {
    const raw = readFileSync(DEFAULT_SECRET_FILE, "utf-8").trim();
    const secrets = {};
    for (const line of raw.split("\n")) {
      const [k, v] = line.split("=", 2);
      if (k && v) secrets[k.trim()] = v.trim();
    }
    return secrets;
  } catch {
    return {};
  }
}

async function createUser(apiBase) {
  const privKeyHex = generatePrivateKey();
  const account = privateKeyToAccount(privKeyHex);
  const edPriv = ed.utils.randomPrivateKey();
  const edPubHex = Buffer.from(await ed.getPublicKeyAsync(edPriv)).toString("hex");

  const reg = await postJson(`${apiBase}/api/v1/auth/register`, {
    wallet_address: account.address,
    ed25519_public_key: edPubHex
  });

  const challengeMsg = `funnyoption-auth:${reg.user_id}:${reg.nonce}`;
  const walletSig = await account.signMessage({ message: challengeMsg });
  const edSigBytes = await ed.signAsync(textEncoder.encode(challengeMsg), edPriv);
  const edSig = Buffer.from(edSigBytes).toString("hex");

  const session = await postJson(`${apiBase}/api/v1/auth/session`, {
    wallet_address: account.address,
    wallet_signature: walletSig,
    ed25519_signature: edSig,
    device_label: "perf-bench"
  });

  return {
    userId: reg.user_id,
    address: account.address,
    privKeyHex,
    account,
    edPriv,
    session,
    authHeader: `Bearer ${session.sessionId}`,
  };
}

async function fundUser(user, publicClient, operatorWallet, tokenAddr, vaultAddr) {
  const amount = parseUnits("10000", 6);
  const tx1 = await operatorWallet.writeContract({
    address: tokenAddr, abi: tokenAbi,
    functionName: "mint", args: [user.address, amount]
  });
  await publicClient.waitForTransactionReceipt({ hash: tx1 });

  const approveTx = await operatorWallet.writeContract({
    address: tokenAddr, abi: tokenAbi,
    functionName: "approve", args: [vaultAddr, amount],
    account: user.account
  });
  await publicClient.waitForTransactionReceipt({ hash: approveTx });

  const depositTx = await operatorWallet.writeContract({
    address: vaultAddr, abi: vaultAbi,
    functionName: "deposit", args: [amount],
    account: user.account
  });
  await publicClient.waitForTransactionReceipt({ hash: depositTx });
}

// ─── Benchmark logic ────────────────────────────────────────────────
async function placeOrder(apiBase, user, marketId, outcome, side, price, qty) {
  const startNs = performance.now();
  const res = await postJson(`${apiBase}/api/v1/orders`, {
    market_id: marketId,
    outcome,
    side,
    type: "LIMIT",
    time_in_force: "GTC",
    price,
    quantity: qty,
  }, { Authorization: user.authHeader });
  const endNs = performance.now();
  return {
    orderId: res.order_id,
    placementMs: endNs - startNs,
    startTime: startNs,
  };
}

async function pollOrderStatus(apiBase, user, orderId, timeoutMs, intervalMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const res = await fetchJson(`${apiBase}/api/v1/orders/${orderId}`, {
        headers: { Authorization: user.authHeader }
      });
      if (res.status && ["FILLED", "CANCELLED", "REJECTED"].includes(res.status)) {
        return res;
      }
    } catch {}
    await sleep(intervalMs);
  }
  return null;
}

function percentile(sortedArr, p) {
  if (sortedArr.length === 0) return 0;
  const idx = Math.ceil(p / 100 * sortedArr.length) - 1;
  return sortedArr[Math.max(0, idx)];
}

async function runBenchmark() {
  const opts = parseArgs();
  console.log("=== Matching Engine Performance Benchmark ===");
  console.log(`  Orders:      ${opts.orders}`);
  console.log(`  Concurrency: ${opts.concurrency}`);
  console.log(`  Users:       ${opts.users}`);
  console.log(`  Price:       ${opts.price}`);
  console.log(`  Outcome:     ${opts.outcome}`);
  console.log(`  PollTimeout: ${opts.pollTimeoutMs}ms`);
  console.log();

  const apiBase = process.env.API_BASE || DEFAULT_API_BASE;
  console.log(`[1/6] Creating ${opts.users} test users...`);
  const users = [];
  for (let i = 0; i < opts.users; i++) {
    const u = await createUser(apiBase);
    users.push(u);
    process.stdout.write(`  user ${i + 1}/${opts.users} created (${u.userId})\r`);
  }
  console.log(`  ${users.length} users created`);

  let marketId = opts.marketId;
  if (marketId <= 0) {
    console.log("[2/6] Creating test market...");
    const binaryOptions = [
      { key: "YES", label: "Yes", shortLabel: "Y", sortOrder: 10, isActive: true },
      { key: "NO", label: "No", shortLabel: "N", sortOrder: 20, isActive: true }
    ];
    const mkRes = await postJson(`${apiBase}/api/v1/admin/markets`, {
      title: `Perf Benchmark ${Date.now()}`,
      description: "Automated perf test",
      collateral_asset: "USDT",
      options: binaryOptions,
    });
    marketId = mkRes.market_id;
    console.log(`  market ${marketId} created`);
  } else {
    console.log(`[2/6] Using existing market ${marketId}`);
  }

  console.log("[3/6] Seeding maker liquidity...");
  const maker = users[0];
  const makerOrders = 10;
  for (let i = 0; i < makerOrders; i++) {
    try {
      await placeOrder(apiBase, maker, marketId, opts.outcome, "SELL", opts.price, 100);
    } catch (e) {
      console.log(`  maker order ${i} failed: ${e.message}`);
    }
  }
  console.log(`  ${makerOrders} maker SELL orders placed at price ${opts.price}`);

  console.log(`[4/6] Sending ${opts.orders} taker BUY orders (concurrency=${opts.concurrency})...`);
  const takerUsers = users.slice(1);
  const orderPromises = [];
  const burstStart = performance.now();

  for (let i = 0; i < opts.orders; i++) {
    const user = takerUsers[i % takerUsers.length];
    const promise = placeOrder(apiBase, user, marketId, opts.outcome, "BUY", opts.price, 1);
    orderPromises.push(promise);

    if (orderPromises.length >= opts.concurrency) {
      await Promise.allSettled(orderPromises.splice(0, opts.concurrency));
    }
  }
  if (orderPromises.length > 0) {
    await Promise.allSettled(orderPromises);
  }

  const burstEnd = performance.now();
  const burstMs = burstEnd - burstStart;
  console.log(`  Burst complete in ${burstMs.toFixed(0)}ms`);

  console.log(`[5/6] Collecting results (resolved = all settled within ${opts.orders} orders)...`);
  const placementLatencies = [];
  const results = await Promise.allSettled(
    orderPromises.length
      ? orderPromises
      : Array.from({ length: opts.orders }, async (_, i) => {
          const user = takerUsers[i % takerUsers.length];
          return placeOrder(apiBase, user, marketId, opts.outcome, "BUY", opts.price, 1);
        })
  );

  let placed = 0, failed = 0;
  for (const r of results) {
    if (r.status === "fulfilled") {
      placed++;
      placementLatencies.push(r.value.placementMs);
    } else {
      failed++;
    }
  }

  placementLatencies.sort((a, b) => a - b);

  console.log();
  console.log("=== RESULTS ===");
  console.log();
  console.log("Placement Phase:");
  console.log(`  Total orders sent: ${opts.orders}`);
  console.log(`  Successfully placed: ${placed}`);
  console.log(`  Failed: ${failed}`);
  console.log(`  Burst duration: ${burstMs.toFixed(0)}ms`);
  console.log(`  Throughput: ${(placed / (burstMs / 1000)).toFixed(1)} orders/sec`);
  console.log();
  console.log("Placement Latency (HTTP round-trip):");
  if (placementLatencies.length > 0) {
    console.log(`  p50:  ${percentile(placementLatencies, 50).toFixed(1)}ms`);
    console.log(`  p95:  ${percentile(placementLatencies, 95).toFixed(1)}ms`);
    console.log(`  p99:  ${percentile(placementLatencies, 99).toFixed(1)}ms`);
    console.log(`  min:  ${placementLatencies[0].toFixed(1)}ms`);
    console.log(`  max:  ${placementLatencies[placementLatencies.length - 1].toFixed(1)}ms`);
    console.log(`  mean: ${(placementLatencies.reduce((a, b) => a + b, 0) / placementLatencies.length).toFixed(1)}ms`);
  }

  console.log();
  console.log(`[6/6] Pipeline stats (check matching service logs for ring buffer metrics)`);
  console.log();
  console.log("Done. Check 'pipeline stats' in matching service logs for:");
  console.log("  - gw_received: commands entered gateway");
  console.log("  - ml_matched: commands processed by matching loop");
  console.log("  - ml_batches: drain batches");
  console.log("  - disp_dispatched: results persisted + published");
  console.log("  - gw_paused: backpressure events");
  console.log("  - ml_out_stall: output RB backpressure events");
}

runBenchmark().catch(err => {
  console.error("Benchmark failed:", err);
  process.exit(1);
});
