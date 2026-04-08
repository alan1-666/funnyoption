#!/usr/bin/env node

/**
 * Staging smoke: BNB depositNative → wait for USDT (accounting) balance > 0.
 * Requires operator key in .secrets or FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY.
 *
 * Run after chain service deploy (listener must use ChainToAccountingAmountFloor for deposits).
 */

import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { setTimeout as sleep } from "node:timers/promises";
import * as ed from "../web/node_modules/@noble/ed25519/index.js";
import {
  createPublicClient,
  createWalletClient,
  encodeFunctionData,
  http,
  parseAbi,
  parseEther,
} from "../web/node_modules/viem/_esm/index.js";
import { bscTestnet } from "../web/node_modules/viem/_esm/chains/index.js";
import { generatePrivateKey, privateKeyToAccount } from "../web/node_modules/viem/_esm/accounts/index.js";

const API_BASE = (process.env.API_BASE || "https://funnyoption.xyz").replace(/\/+$/, "");
const RPC_URL = process.env.FUNNYOPTION_CHAIN_RPC_URL || "https://data-seed-prebsc-1-s1.bnbchain.org:8545";
const VAULT_ADDRESS = process.env.FUNNYOPTION_VAULT_ADDRESS || "0xf47e6e19DC896ff8C9137C19782eb22411d0d1Bb";
const SECRET_FILE = resolve(fileURLToPath(new URL(".", import.meta.url)), "../.secrets");
const NATIVE_AMOUNT = process.env.NATIVE_DEPOSIT_TEST_AMOUNT || "0.02";
const GAS_FUND_BNB = process.env.NATIVE_DEPOSIT_GAS_FUND || "0.08";

const vaultAbi = parseAbi(["function depositNative() payable"]);

const textEncoder = new TextEncoder();
function norm(v) { return String(v ?? "").trim().toLowerCase(); }
function toHex(bytes) { return `0x${Array.from(bytes, b => b.toString(16).padStart(2, "0")).join("")}`; }

function readSecret() {
  if (process.env.FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY?.trim()) {
    return process.env.FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY.trim();
  }
  const raw = readFileSync(SECRET_FILE, "utf-8");
  for (const line of raw.split("\n")) {
    const [k, ...v] = line.split(":");
    if (k?.trim() === "bsc-testnet-operator.key" && v.join(":").trim()) return v.join(":").trim();
  }
  throw new Error("operator key not found");
}

async function fetchJson(url, opts = {}) {
  const h = { ...opts.headers };
  if (opts.body !== undefined) h["Content-Type"] = "application/json";
  const r = await fetch(url, {
    method: opts.method || "GET",
    headers: Object.keys(h).length ? h : undefined,
    body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
    signal: AbortSignal.timeout(30_000),
  });
  const text = await r.text();
  let data;
  try { data = JSON.parse(text); } catch { data = { raw: text }; }
  return { ok: r.ok, status: r.status, data };
}

async function postOk(url, body, label) {
  const r = await fetchJson(url, { method: "POST", body });
  if (!r.ok) throw new Error(`${label}: HTTP ${r.status} ${JSON.stringify(r.data)}`);
  return r.data ?? {};
}

async function waitTx(pc, hash, label) {
  const rec = await pc.waitForTransactionReceipt({ hash, confirmations: 1, timeout: 120_000 });
  if (rec.status !== "success") throw new Error(`${label} reverted ${hash}`);
}

async function createSession(account, walletClient) {
  const sessionPrivateKey = ed.utils.randomPrivateKey();
  const sessionPublicKey = toHex(await ed.getPublicKeyAsync(sessionPrivateKey));

  let challenge;
  for (let i = 0; i < 40; i++) {
    const r = await fetchJson(`${API_BASE}/api/v1/trading-keys/challenge`, {
      method: "POST",
      body: { wallet_address: account.address, chain_id: bscTestnet.id, vault_address: VAULT_ADDRESS },
    });
    if (r.ok) { challenge = r.data; break; }
    await sleep(3000);
  }
  if (!challenge) throw new Error("challenge failed");

  const typedData_domain = {
    name: "FunnyOption Trading Authorization",
    version: "2",
    chainId: bscTestnet.id,
    verifyingContract: norm(VAULT_ADDRESS),
  };
  const typedData = {
    domain: typedData_domain,
    types: {
      AuthorizeTradingKey: [
        { name: "action", type: "string" }, { name: "wallet", type: "address" },
        { name: "tradingPublicKey", type: "bytes32" }, { name: "tradingKeyScheme", type: "string" },
        { name: "scope", type: "string" }, { name: "challenge", type: "bytes32" },
        { name: "challengeExpiresAt", type: "uint64" }, { name: "keyExpiresAt", type: "uint64" },
      ],
    },
    primaryType: "AuthorizeTradingKey",
    message: {
      action: "AUTHORIZE_TRADING_KEY",
      wallet: norm(account.address),
      tradingPublicKey: norm(sessionPublicKey),
      tradingKeyScheme: "ED25519",
      scope: "TRADE",
      challenge: norm(challenge.challenge),
      challengeExpiresAt: BigInt(Math.floor(Number(challenge.challenge_expires_at || 0))),
      keyExpiresAt: BigInt(0),
    },
  };
  const walletSig = await account.signTypedData(typedData);
  const res = await postOk(`${API_BASE}/api/v1/trading-keys`, {
    wallet_address: account.address,
    chain_id: bscTestnet.id,
    vault_address: VAULT_ADDRESS,
    challenge_id: challenge.challenge_id,
    challenge: challenge.challenge,
    challenge_expires_at: challenge.challenge_expires_at,
    trading_public_key: sessionPublicKey,
    trading_key_scheme: "ED25519",
    scope: "TRADE",
    key_expires_at: 0,
    wallet_signature_standard: "EIP712_V4",
    wallet_signature: walletSig,
  }, "trading-keys");

  return {
    userId: Number(res.user_id),
    sessionId: String(res.session_id),
    sessionPrivateKey,
    walletAddress: account.address,
  };
}

async function waitBalance(userId, sessionId) {
  const headers = { Authorization: `Bearer ${sessionId}` };
  for (let i = 0; i < 45; i++) {
    const r = await fetchJson(`${API_BASE}/api/v1/balances?user_id=${userId}&asset=USDT&limit=5`, { headers });
    if (r.ok) {
      const items = r.data?.items || r.data?.balances || [];
      const usdt = items.find((b) => String(b.asset).toUpperCase() === "USDT");
      if (usdt && Number(usdt.available) > 0) return Number(usdt.available);
    }
    await sleep(2000);
  }
  return 0;
}

async function main() {
  console.log("Native deposit → balance smoke test");
  console.log(`API=${API_BASE} vault=${VAULT_ADDRESS} amount=${NATIVE_AMOUNT} tBNB\n`);

  const operatorAccount = privateKeyToAccount(`0x${readSecret().replace(/^0x/, "")}`);
  const pc = createPublicClient({ chain: bscTestnet, transport: http(RPC_URL) });
  const opWallet = createWalletClient({ account: operatorAccount, chain: bscTestnet, transport: http(RPC_URL) });

  const userKey = generatePrivateKey();
  const userAccount = privateKeyToAccount(userKey);
  const userWallet = createWalletClient({ account: userAccount, chain: bscTestnet, transport: http(RPC_URL) });

  console.log(`Fund gas (${GAS_FUND_BNB} tBNB)...`);
  await waitTx(
    pc,
    await opWallet.sendTransaction({ to: userAccount.address, value: parseEther(GAS_FUND_BNB) }),
    "gas",
  );

  console.log("Session...");
  const session = await createSession(userAccount, userWallet);

  const data = encodeFunctionData({ abi: vaultAbi, functionName: "depositNative" });
  console.log(`depositNative ${NATIVE_AMOUNT} tBNB...`);
  const hash = await userWallet.sendTransaction({
    to: VAULT_ADDRESS,
    data,
    value: parseEther(NATIVE_AMOUNT),
  });
  await waitTx(pc, hash, "depositNative");
  console.log("tx ok", hash);

  const bal = await waitBalance(session.userId, session.sessionId);
  if (bal <= 0) {
    console.error("FAIL: USDT accounting balance still 0 after depositNative (check chain listener / deploy).");
    process.exit(1);
  }
  console.log(`PASS: available USDT (accounting units) = ${bal}`);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
