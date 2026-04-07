#!/usr/bin/env node

import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { setTimeout as sleep } from "node:timers/promises";
import * as ed from "../web/node_modules/@noble/ed25519/index.js";
import {
  createPublicClient,
  createWalletClient,
  formatEther,
  formatUnits,
  http,
  parseAbi,
  parseEther,
  parseUnits
} from "../web/node_modules/viem/_esm/index.js";
import { bscTestnet } from "../web/node_modules/viem/_esm/chains/index.js";
import { generatePrivateKey, privateKeyToAccount } from "../web/node_modules/viem/_esm/accounts/index.js";

const DEFAULT_SECRET_FILE = fileURLToPath(new URL("../.secrets", import.meta.url));
const DEFAULT_SECRET_KEY = "bsc-testnet-operator.key";
const DEFAULT_API_BASE = "https://funnyoption.xyz";
const DEFAULT_ADMIN_BASE = "https://admin.funnyoption.xyz";
const DEFAULT_RPC_URL = "https://data-seed-prebsc-1-s1.bnbchain.org:8545";
const DEFAULT_TOKEN_ADDRESS = "0x0ADa04558decC14671D565562Aeb8D1096F71dDc";
const DEFAULT_VAULT_ADDRESS = "0x7665d943c62268d27ffcbed29c6a8281f7364534";
const DEFAULT_MAKER_USER_ID = 1002;
const DEFAULT_BOOTSTRAP_PRICE = 58;
const DEFAULT_MATCH_PRICE = 58;
const DEFAULT_USERS = 4;
const DEFAULT_ORDERS_PER_USER = 2;
const DEFAULT_CONCURRENCY = 4;
const DEFAULT_FUND_TBNB = "0.03";
const DEFAULT_HTTP_TIMEOUT_MS = 15_000;
const DEFAULT_POLL_TIMEOUT_MS = 240_000;
const DEFAULT_POLL_INTERVAL_MS = 3_000;
const MAX_USERS = 24;
const MAX_ORDERS_PER_USER = 20;
const MAX_CONCURRENCY = 16;
const MAX_TOTAL_BURST_ORDERS = 120;
const FETCH_LIMIT = 200;

const TERMINAL_ORDER_STATUSES = new Set(["FILLED", "CANCELLED", "REJECTED", "FAILED", "REVOKED"]);
const OPEN_ORDER_STATUSES = new Set(["NEW", "PARTIALLY_FILLED", "QUEUED"]);
const HEALTHY_FREEZE_TERMINAL_STATUSES = new Set(["CONSUMED", "RELEASED"]);

const textEncoder = new TextEncoder();

const binaryOptions = [
  { key: "YES", label: "是", shortLabel: "是", sortOrder: 10, isActive: true },
  { key: "NO", label: "否", shortLabel: "否", sortOrder: 20, isActive: true }
];

const tokenAbi = parseAbi([
  "function owner() view returns (address)",
  "function balanceOf(address) view returns (uint256)",
  "function mint(address,uint256) returns (bool)",
  "function transfer(address,uint256) returns (bool)",
  "function approve(address,uint256) returns (bool)"
]);

const vaultAbi = parseAbi(["function deposit(uint256)"]);

function usage() {
  return `Usage:
  node scripts/staging-concurrency-orders.mjs [options]

Options:
  --api-base <url>              Public API base URL. Default: ${DEFAULT_API_BASE}
  --admin-base <url>            Admin route base URL. Default: ${DEFAULT_ADMIN_BASE}
  --rpc-url <url>               BSC Testnet RPC URL. Default: ${DEFAULT_RPC_URL}
  --token-address <address>     Collateral token address. Default: ${DEFAULT_TOKEN_ADDRESS}
  --vault-address <address>     Vault address. Default: ${DEFAULT_VAULT_ADDRESS}
  --secret-file <path>          Operator secret file. Default: ${DEFAULT_SECRET_FILE}
  --secret-key <label>          Key label in the secret file. Default: ${DEFAULT_SECRET_KEY}
  --maker-user-id <n>           Maker user used for first liquidity. Default: ${DEFAULT_MAKER_USER_ID}
  --users <n>                   Number of generated taker users. Default: ${DEFAULT_USERS} (2..${MAX_USERS})
  --seller-users <n>            Number of takers preseeded with YES inventory. Default: floor(users / 2), min 1
  --orders-per-user <n>         Number of burst orders per user. Default: ${DEFAULT_ORDERS_PER_USER} (1..${MAX_ORDERS_PER_USER})
  --concurrency <n>             Max parallel user order pipelines. Default: ${DEFAULT_CONCURRENCY} (1..${MAX_CONCURRENCY})
  --bootstrap-price <cents>     Bootstrap YES sell price in cents. Default: ${DEFAULT_BOOTSTRAP_PRICE}
  --match-price <cents>         Burst order cross price in cents. Default: ${DEFAULT_MATCH_PRICE}
  --bootstrap-quantity <n>      First bootstrap quantity. Default: seller-users * orders-per-user
  --deposit-usdt <amount>       Per-user deposit amount in USDT, 2 decimals. Default: auto-sized from orders-per-user and match price
  --fund-usdt <amount>          Per-user token funding amount in USDT, 2 decimals. Default: deposit-usdt + 5.00
  --fund-tbnb <amount>          Per-user native funding amount. Default: ${DEFAULT_FUND_TBNB}
  --http-timeout-ms <ms>        Request timeout. Default: ${DEFAULT_HTTP_TIMEOUT_MS}
  --poll-timeout-ms <ms>        Async state timeout. Default: ${DEFAULT_POLL_TIMEOUT_MS}
  --poll-interval-ms <ms>       Poll interval. Default: ${DEFAULT_POLL_INTERVAL_MS}
  --help                        Show this help message
`;
}

function parseCliArgs(argv) {
  const parsed = {};
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (token === "--help") {
      parsed.help = true;
      continue;
    }
    if (!token.startsWith("--")) {
      throw new Error(`unexpected positional argument: ${token}`);
    }
    const eqIndex = token.indexOf("=");
    if (eqIndex > 2) {
      parsed[token.slice(2, eqIndex)] = token.slice(eqIndex + 1);
      continue;
    }
    const key = token.slice(2);
    const next = argv[index + 1];
    if (!next || next.startsWith("--")) {
      throw new Error(`missing value for --${key}`);
    }
    parsed[key] = next;
    index += 1;
  }
  return parsed;
}

function parseIntegerOption(raw, name, fallback, min, max) {
  const value = raw === undefined ? fallback : Number(raw);
  if (!Number.isInteger(value) || value < min || value > max) {
    throw new Error(`${name} must be an integer between ${min} and ${max}, got ${String(raw ?? fallback)}`);
  }
  return value;
}

function parsePositiveIntegerOption(raw, name, fallback) {
  const value = raw === undefined ? fallback : Number(raw);
  if (!Number.isInteger(value) || value <= 0) {
    throw new Error(`${name} must be a positive integer, got ${String(raw ?? fallback)}`);
  }
  return value;
}

function parsePositiveMsOption(raw, name, fallback) {
  const value = parsePositiveIntegerOption(raw, name, fallback);
  if (value < 100) {
    throw new Error(`${name} must be at least 100ms, got ${value}`);
  }
  return value;
}

function parseAccountingAmount(raw, name) {
  const value = String(raw ?? "").trim();
  const match = /^(\d+)(?:\.(\d{1,2}))?$/.exec(value);
  if (!match) {
    throw new Error(`${name} must be a decimal with up to 2 places, got ${value || "<empty>"}`);
  }
  return Number(match[1]) * 100 + Number((match[2] ?? "").padEnd(2, "0"));
}

function accountingAmountToHuman(units) {
  const normalized = Math.max(0, Math.floor(Number(units || 0)));
  return `${Math.floor(normalized / 100)}.${String(normalized % 100).padStart(2, "0")}`;
}

function buildConfig(argv) {
  const raw = parseCliArgs(argv);
  if (raw.help) {
    return { help: true };
  }

  const users = parseIntegerOption(raw.users, "--users", DEFAULT_USERS, 2, MAX_USERS);
  const sellerUsers = parseIntegerOption(
    raw["seller-users"],
    "--seller-users",
    Math.max(1, Math.floor(users / 2)),
    1,
    users - 1
  );
  const ordersPerUser = parseIntegerOption(
    raw["orders-per-user"],
    "--orders-per-user",
    DEFAULT_ORDERS_PER_USER,
    1,
    MAX_ORDERS_PER_USER
  );
  const concurrency = parseIntegerOption(raw.concurrency, "--concurrency", DEFAULT_CONCURRENCY, 1, MAX_CONCURRENCY);
  const bootstrapPrice = parseIntegerOption(raw["bootstrap-price"], "--bootstrap-price", DEFAULT_BOOTSTRAP_PRICE, 1, 99);
  const matchPrice = parseIntegerOption(raw["match-price"], "--match-price", DEFAULT_MATCH_PRICE, 1, 99);
  const bootstrapQuantity = parsePositiveIntegerOption(
    raw["bootstrap-quantity"],
    "--bootstrap-quantity",
    sellerUsers * ordersPerUser
  );
  const totalBurstOrders = users * ordersPerUser;
  if (totalBurstOrders > MAX_TOTAL_BURST_ORDERS) {
    throw new Error(
      `users * orders-per-user must be <= ${MAX_TOTAL_BURST_ORDERS} to keep staging load bounded, got ${totalBurstOrders}`
    );
  }
  if (bootstrapQuantity < sellerUsers * ordersPerUser) {
    throw new Error(
      `--bootstrap-quantity must be >= seller-users * orders-per-user (${sellerUsers * ordersPerUser}) so preseed sellers can acquire inventory`
    );
  }

  const suggestedDepositAccounting = Math.max(
    500,
    ordersPerUser * Math.max(bootstrapPrice, matchPrice) + 300
  );
  const depositUsdt = String(raw["deposit-usdt"] ?? accountingAmountToHuman(suggestedDepositAccounting)).trim();
  const depositAccounting = parseAccountingAmount(depositUsdt, "--deposit-usdt");
  const minDepositAccounting = ordersPerUser * Math.max(bootstrapPrice, matchPrice) + 100;
  if (depositAccounting < minDepositAccounting) {
    throw new Error(
      `--deposit-usdt is too small for ${ordersPerUser} order(s) at ${Math.max(bootstrapPrice, matchPrice)}¢; need at least ${accountingAmountToHuman(minDepositAccounting)}`
    );
  }

  const fundUsdt = String(raw["fund-usdt"] ?? accountingAmountToHuman(depositAccounting + 500)).trim();
  const fundAccounting = parseAccountingAmount(fundUsdt, "--fund-usdt");
  if (fundAccounting < depositAccounting) {
    throw new Error(`--fund-usdt must be >= --deposit-usdt (${depositUsdt}), got ${fundUsdt}`);
  }

  return {
    help: false,
    apiBase: String(raw["api-base"] ?? DEFAULT_API_BASE).replace(/\/+$/, ""),
    adminBase: String(raw["admin-base"] ?? DEFAULT_ADMIN_BASE).replace(/\/+$/, ""),
    rpcUrl: String(raw["rpc-url"] ?? process.env.FUNNYOPTION_CHAIN_RPC_URL ?? DEFAULT_RPC_URL).trim(),
    tokenAddress: String(
      raw["token-address"] ?? process.env.FUNNYOPTION_COLLATERAL_TOKEN_ADDRESS ?? DEFAULT_TOKEN_ADDRESS
    ).trim(),
    vaultAddress: String(
      raw["vault-address"] ?? process.env.FUNNYOPTION_VAULT_ADDRESS ?? DEFAULT_VAULT_ADDRESS
    ).trim(),
    secretFile: resolve(process.cwd(), String(raw["secret-file"] ?? DEFAULT_SECRET_FILE)),
    secretKey: String(raw["secret-key"] ?? DEFAULT_SECRET_KEY).trim(),
    makerUserId: parsePositiveIntegerOption(raw["maker-user-id"], "--maker-user-id", DEFAULT_MAKER_USER_ID),
    users,
    sellerUsers,
    buyerUsers: users - sellerUsers,
    ordersPerUser,
    totalBurstOrders,
    concurrency,
    bootstrapPrice,
    matchPrice,
    bootstrapQuantity,
    depositUsdt,
    depositAccounting,
    fundUsdt,
    fundTbnb: String(raw["fund-tbnb"] ?? DEFAULT_FUND_TBNB).trim(),
    httpTimeoutMs: parsePositiveMsOption(raw["http-timeout-ms"], "--http-timeout-ms", DEFAULT_HTTP_TIMEOUT_MS),
    pollTimeoutMs: parsePositiveMsOption(raw["poll-timeout-ms"], "--poll-timeout-ms", DEFAULT_POLL_TIMEOUT_MS),
    pollIntervalMs: parsePositiveMsOption(
      raw["poll-interval-ms"],
      "--poll-interval-ms",
      DEFAULT_POLL_INTERVAL_MS
    )
  };
}

function nowIso() {
  return new Date().toISOString();
}

function logStep(step, payload) {
  if (payload === undefined) {
    console.log(`[${nowIso()}] ${step}`);
    return;
  }
  console.log(`[${nowIso()}] ${step} ${JSON.stringify(payload)}`);
}

function normalizeAddress(value) {
  return String(value ?? "").trim().toLowerCase();
}

function normalizeHex(value) {
  return String(value ?? "").trim().toLowerCase().replace(/^0x/, "");
}

function cleanText(value) {
  return String(value ?? "").trim().replace(/\s+/g, " ");
}

function toHex(bytes) {
  return `0x${Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
}

function optionFragment(options) {
  return [...options]
    .map((option, index) => ({
      key: cleanText(option.key).toUpperCase().replace(/\s+/g, "_"),
      label: cleanText(option.label),
      shortLabel: cleanText(option.shortLabel ?? option.label),
      sortOrder: Math.max(1, Math.floor(Number(option.sortOrder || (index + 1) * 10))),
      isActive: option.isActive !== false
    }))
    .sort((left, right) => left.sortOrder - right.sortOrder || left.key.localeCompare(right.key))
    .map((option) => `${option.key}:${option.label}:${option.shortLabel}:${option.sortOrder}:${option.isActive ? "1" : "0"}`)
    .join("|");
}

function buildCreateMarketMessage({ walletAddress, market, requestedAt }) {
  return `FunnyOption Operator Authorization

action: CREATE_MARKET
wallet: ${normalizeAddress(walletAddress)}
title: ${cleanText(market.title)}
description: ${cleanText(market.description)}
category: ${cleanText(market.categoryKey).toUpperCase() || "CRYPTO"}
source_kind: ${cleanText(market.sourceKind).toLowerCase() || "manual"}
source_url: ${String(market.sourceUrl ?? "").trim()}
source_slug: ${cleanText(market.sourceSlug)}
source_name: ${cleanText(market.sourceName) || "FunnyOption"}
cover_image: ${String(market.coverImage ?? "").trim()}
status: ${cleanText(market.status).toUpperCase() || "OPEN"}
collateral_asset: ${cleanText(market.collateralAsset).toUpperCase() || "USDT"}
open_at: ${Math.max(0, Math.floor(Number(market.openAt || 0)))}
close_at: ${Math.max(0, Math.floor(Number(market.closeAt || 0)))}
resolve_at: ${Math.max(0, Math.floor(Number(market.resolveAt || 0)))}
requested_at: ${Math.floor(requestedAt)}
options: ${optionFragment(market.options)}
`;
}

function buildBootstrapMarketMessage({ walletAddress, bootstrap, requestedAt }) {
  return `FunnyOption Operator Authorization

action: ISSUE_FIRST_LIQUIDITY
wallet: ${normalizeAddress(walletAddress)}
market_id: ${Math.max(0, Math.floor(Number(bootstrap.marketId || 0)))}
user_id: ${Math.max(0, Math.floor(Number(bootstrap.userId || 0)))}
quantity: ${Math.max(0, Math.floor(Number(bootstrap.quantity || 0)))}
outcome: ${cleanText(bootstrap.outcome).toUpperCase() === "NO" ? "NO" : "YES"}
price: ${Math.max(0, Math.floor(Number(bootstrap.price || 0)))}
requested_at: ${Math.floor(requestedAt)}
`;
}

function buildResolveMarketMessage({ walletAddress, market, requestedAt }) {
  return `FunnyOption Operator Authorization

action: RESOLVE_MARKET
wallet: ${normalizeAddress(walletAddress)}
market_id: ${Math.max(0, Math.floor(Number(market.marketId || 0)))}
outcome: ${cleanText(market.outcome).toUpperCase() === "NO" ? "NO" : "YES"}
requested_at: ${Math.floor(requestedAt)}
`;
}

function buildTradingKeyAuthorizationTypedData(input) {
  return {
    domain: {
      name: "FunnyOption Trading Authorization",
      version: "2",
      chainId: Math.floor(Number(input.chainId || 0)),
      verifyingContract: normalizeAddress(input.vaultAddress)
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
        { name: "keyExpiresAt", type: "uint64" }
      ]
    },
    primaryType: "AuthorizeTradingKey",
    message: {
      action: "AUTHORIZE_TRADING_KEY",
      wallet: normalizeAddress(input.walletAddress),
      tradingPublicKey: normalizeAddress(input.tradingPublicKey),
      tradingKeyScheme: cleanText(input.tradingKeyScheme).toUpperCase() || "ED25519",
      scope: cleanText(input.scope).toUpperCase() || "TRADE",
      challenge: normalizeAddress(input.challenge),
      challengeExpiresAt: BigInt(Math.floor(Number(input.challengeExpiresAt || 0))),
      keyExpiresAt: BigInt(Math.floor(Number(input.keyExpiresAt || 0)))
    }
  };
}

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

function readSecretValue(secretFile, keyLabel) {
  if (process.env.FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY?.trim()) {
    return process.env.FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY.trim();
  }

  let raw;
  try {
    raw = readFileSync(secretFile, "utf8");
  } catch (error) {
    throw new Error(
      `cannot read operator secret from ${secretFile}; set FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY or provide --secret-file`
    );
  }

  for (const line of raw.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }
    if (trimmed.startsWith(`${keyLabel}:`)) {
      const value = trimmed.slice(keyLabel.length + 1).trim();
      if (!value) {
        throw new Error(`secret ${keyLabel} exists in ${secretFile} but has an empty value`);
      }
      return value;
    }
    if (trimmed.startsWith(`${keyLabel}=`)) {
      const value = trimmed.slice(keyLabel.length + 1).trim();
      if (!value) {
        throw new Error(`secret ${keyLabel} exists in ${secretFile} but has an empty value`);
      }
      return value;
    }
  }

  throw new Error(
    `secret ${keyLabel} not found in ${secretFile}; set FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY or pass --secret-key`
  );
}

function readOperatorAccount(config) {
  const secret = readSecretValue(config.secretFile, config.secretKey);
  return privateKeyToAccount(`0x${secret.trim().replace(/^0x/, "")}`);
}

async function requestJson(config, url, { method = "GET", body, headers: extraHeaders } = {}) {
  let lastError = null;
  for (let attempt = 1; attempt <= 3; attempt += 1) {
    try {
      const headers = { ...extraHeaders };
      if (body !== undefined) headers["Content-Type"] = "application/json";
      const response = await fetch(url, {
        method,
        headers: Object.keys(headers).length > 0 ? headers : undefined,
        body: body === undefined ? undefined : JSON.stringify(body),
        signal: AbortSignal.timeout(config.httpTimeoutMs)
      });
      const text = await response.text();
      let data = null;
      if (text) {
        try {
          data = JSON.parse(text);
        } catch {
          data = { raw: text };
        }
      }
      return {
        ok: response.ok,
        status: response.status,
        data,
        text
      };
    } catch (error) {
      lastError = error;
      if (attempt < 3) {
        await sleep(500 * attempt);
      }
    }
  }
  const message = lastError instanceof Error ? lastError.message : String(lastError);
  const cause = lastError?.cause ? ` cause=${String(lastError.cause)}` : "";
  throw new Error(`fetch ${method} ${url} failed: ${message}${cause}`);
}

async function getJsonOrThrow(config, url, label, { headers } = {}) {
  const response = await requestJson(config, url, { headers });
  if (!response.ok) {
    throw new Error(`${label} failed: HTTP ${response.status} ${response.text}`);
  }
  return response.data ?? {};
}

async function postJsonOrThrow(config, url, body, label, expectedStatuses = [200, 201, 202]) {
  const response = await requestJson(config, url, {
    method: "POST",
    body
  });
  if (!expectedStatuses.includes(response.status)) {
    throw new Error(`${label} failed: HTTP ${response.status} ${response.text}`);
  }
  return response.data ?? {};
}

async function waitFor(label, fn, timeoutMs, intervalMs) {
  const startedAt = Date.now();
  let lastValue = null;
  let lastError = null;

  while (Date.now() - startedAt < timeoutMs) {
    try {
      lastValue = await fn();
      if (lastValue) {
        return lastValue;
      }
    } catch (error) {
      lastError = error;
    }
    await sleep(intervalMs);
  }

  const detail = lastError ? String(lastError.message ?? lastError) : JSON.stringify(lastValue);
  throw new Error(`${label} timeout after ${timeoutMs}ms; last=${detail}`);
}

async function waitTx(publicClient, hash, label) {
  const receipt = await publicClient.waitForTransactionReceipt({
    hash,
    confirmations: 1,
    timeout: 180_000
  });
  if (receipt.status !== "success") {
    throw new Error(`${label} tx failed: ${hash}`);
  }
  return {
    tx_hash: hash,
    block_number: Number(receipt.blockNumber)
  };
}

async function runWithConcurrency(items, limit, fn) {
  const results = new Array(items.length);
  let nextIndex = 0;
  const workers = Array.from({ length: Math.min(limit, items.length) }, async () => {
    while (true) {
      const index = nextIndex;
      nextIndex += 1;
      if (index >= items.length) {
        return;
      }
      results[index] = await fn(items[index], index);
    }
  });
  await Promise.all(workers);
  return results;
}

function summarizeBalance(item) {
  if (!item) {
    return null;
  }
  return {
    user_id: Number(item.user_id),
    asset: String(item.asset),
    available: Number(item.available ?? 0),
    frozen: Number(item.frozen ?? 0)
  };
}

function summarizeOrder(item) {
  if (!item) {
    return null;
  }
  return {
    order_id: String(item.order_id ?? ""),
    user_id: Number(item.user_id ?? 0),
    market_id: Number(item.market_id ?? 0),
    outcome: String(item.outcome ?? ""),
    side: String(item.side ?? ""),
    price: Number(item.price ?? 0),
    quantity: Number(item.quantity ?? 0),
    filled_quantity: Number(item.filled_quantity ?? 0),
    remaining_quantity: Number(item.remaining_quantity ?? 0),
    status: String(item.status ?? ""),
    freeze_id: String(item.freeze_id ?? ""),
    freeze_amount: Number(item.freeze_amount ?? 0),
    cancel_reason: String(item.cancel_reason ?? "")
  };
}

function summarizePosition(item) {
  if (!item) {
    return null;
  }
  return {
    market_id: Number(item.market_id ?? 0),
    user_id: Number(item.user_id ?? 0),
    outcome: String(item.outcome ?? ""),
    position_asset: String(item.position_asset ?? ""),
    quantity: Number(item.quantity ?? 0),
    settled_quantity: Number(item.settled_quantity ?? 0)
  };
}

function summarizePayout(item) {
  if (!item) {
    return null;
  }
  return {
    event_id: String(item.event_id ?? ""),
    market_id: Number(item.market_id ?? 0),
    user_id: Number(item.user_id ?? 0),
    winning_outcome: String(item.winning_outcome ?? ""),
    payout_asset: String(item.payout_asset ?? ""),
    payout_amount: Number(item.payout_amount ?? 0),
    settled_quantity: Number(item.settled_quantity ?? 0),
    status: String(item.status ?? "")
  };
}

function summarizeFreeze(item) {
  if (!item) {
    return null;
  }
  return {
    freeze_id: String(item.freeze_id ?? ""),
    user_id: Number(item.user_id ?? 0),
    asset: String(item.asset ?? ""),
    ref_type: String(item.ref_type ?? ""),
    ref_id: String(item.ref_id ?? ""),
    original_amount: Number(item.original_amount ?? 0),
    remaining_amount: Number(item.remaining_amount ?? 0),
    status: String(item.status ?? "")
  };
}

function summarizeTrade(item) {
  if (!item) {
    return null;
  }
  return {
    trade_id: String(item.trade_id ?? ""),
    sequence_no: Number(item.sequence_no ?? 0),
    market_id: Number(item.market_id ?? 0),
    outcome: String(item.outcome ?? ""),
    price: Number(item.price ?? 0),
    quantity: Number(item.quantity ?? 0),
    taker_order_id: String(item.taker_order_id ?? ""),
    maker_order_id: String(item.maker_order_id ?? ""),
    taker_user_id: Number(item.taker_user_id ?? 0),
    maker_user_id: Number(item.maker_user_id ?? 0),
    taker_side: String(item.taker_side ?? ""),
    maker_side: String(item.maker_side ?? "")
  };
}

function latencySummary(values) {
  if (values.length === 0) {
    return {
      count: 0,
      min_ms: 0,
      p50_ms: 0,
      p95_ms: 0,
      p99_ms: 0,
      max_ms: 0,
      avg_ms: 0
    };
  }

  const sorted = [...values].sort((left, right) => left - right);
  const pick = (ratio) => {
    const index = Math.min(sorted.length - 1, Math.max(0, Math.floor((sorted.length - 1) * ratio)));
    return Number(sorted[index].toFixed(2));
  };
  const sum = sorted.reduce((acc, value) => acc + value, 0);
  return {
    count: sorted.length,
    min_ms: Number(sorted[0].toFixed(2)),
    p50_ms: pick(0.5),
    p95_ms: pick(0.95),
    p99_ms: pick(0.99),
    max_ms: Number(sorted[sorted.length - 1].toFixed(2)),
    avg_ms: Number((sum / sorted.length).toFixed(2))
  };
}

function rpcTransport(config) {
  return http(config.rpcUrl, {
    timeout: config.httpTimeoutMs
  });
}

function authHeaders(sessionId) {
  if (!sessionId) return {};
  return { Authorization: `Bearer ${sessionId}` };
}

async function fetchBalance(config, userId, sessionId) {
  const payload = await getJsonOrThrow(
    config,
    `${config.apiBase}/api/v1/balances?user_id=${userId}&asset=USDT&limit=5`,
    `get balances user=${userId}`,
    { headers: authHeaders(sessionId) }
  );
  return (payload.items ?? []).find((item) => String(item.asset ?? "").toUpperCase() === "USDT") ?? null;
}

async function fetchOrders(config, userId, marketId, sessionId) {
  const payload = await getJsonOrThrow(
    config,
    `${config.apiBase}/api/v1/orders?user_id=${userId}&market_id=${marketId}&limit=${FETCH_LIMIT}`,
    `get orders user=${userId} market=${marketId}`,
    { headers: authHeaders(sessionId) }
  );
  return payload.items ?? [];
}

async function fetchPositions(config, userId, marketId, sessionId) {
  const payload = await getJsonOrThrow(
    config,
    `${config.apiBase}/api/v1/positions?user_id=${userId}&market_id=${marketId}&limit=${FETCH_LIMIT}`,
    `get positions user=${userId} market=${marketId}`,
    { headers: authHeaders(sessionId) }
  );
  return payload.items ?? [];
}

async function fetchPayouts(config, userId, marketId, sessionId) {
  const payload = await getJsonOrThrow(
    config,
    `${config.apiBase}/api/v1/payouts?user_id=${userId}&market_id=${marketId}&limit=${FETCH_LIMIT}`,
    `get payouts user=${userId} market=${marketId}`,
    { headers: authHeaders(sessionId) }
  );
  return payload.items ?? [];
}

async function fetchFreezes(config, userId, sessionId) {
  const payload = await getJsonOrThrow(
    config,
    `${config.apiBase}/api/v1/freezes?user_id=${userId}&limit=${FETCH_LIMIT}`,
    `get freezes user=${userId}`,
    { headers: authHeaders(sessionId) }
  );
  return payload.items ?? [];
}

async function fetchDeposits(config, userId, sessionId) {
  const payload = await getJsonOrThrow(
    config,
    `${config.apiBase}/api/v1/deposits?user_id=${userId}&limit=${FETCH_LIMIT}`,
    `get deposits user=${userId}`,
    { headers: authHeaders(sessionId) }
  );
  return payload.items ?? [];
}

async function fetchTrades(config, marketId) {
  const payload = await getJsonOrThrow(
    config,
    `${config.apiBase}/api/v1/trades?market_id=${marketId}&limit=${FETCH_LIMIT}`,
    `get trades market=${marketId}`
  );
  return payload.items ?? [];
}

async function fetchMarket(config, marketId) {
  return getJsonOrThrow(
    config,
    `${config.apiBase}/api/v1/markets/${marketId}`,
    `get market ${marketId}`
  );
}

async function signOperatorAction(operatorAccount, message, requestedAt) {
  return {
    walletAddress: operatorAccount.address,
    requestedAt,
    signature: await operatorAccount.signMessage({ message })
  };
}

async function createTradingKeyChallenge(config, userAccount) {
  return postJsonOrThrow(
    config,
    `${config.apiBase}/api/v1/trading-keys/challenge`,
    {
      wallet_address: userAccount.address,
      chain_id: bscTestnet.id,
      vault_address: config.vaultAddress
    },
    `create trading key challenge wallet=${userAccount.address}`,
    [200, 201]
  );
}

async function createSession(config, userAccount, userId) {
  const sessionPrivateKey = ed.utils.randomPrivateKey();
  const sessionPublicKey = toHex(await ed.getPublicKeyAsync(sessionPrivateKey));
  const challenge = await createTradingKeyChallenge(config, userAccount);
  const typedData = buildTradingKeyAuthorizationTypedData({
    walletAddress: userAccount.address,
    tradingPublicKey: sessionPublicKey,
    tradingKeyScheme: "ED25519",
    scope: "TRADE",
    chainId: bscTestnet.id,
    vaultAddress: config.vaultAddress,
    challenge: challenge.challenge,
    challengeExpiresAt: challenge.challenge_expires_at,
    keyExpiresAt: 0
  });
  const walletSignature = await userAccount.signTypedData(typedData);

  const payload = await postJsonOrThrow(
    config,
    `${config.apiBase}/api/v1/trading-keys`,
    {
      wallet_address: userAccount.address,
      chain_id: bscTestnet.id,
      vault_address: config.vaultAddress,
      challenge_id: challenge.challenge_id,
      challenge: challenge.challenge,
      challenge_expires_at: challenge.challenge_expires_at,
      trading_public_key: sessionPublicKey,
      trading_key_scheme: "ED25519",
      scope: "TRADE",
      key_expires_at: 0,
      wallet_signature_standard: "EIP712_V4",
      wallet_signature: walletSignature
    },
    `register trading key user=${userId}`,
    [200, 201]
  );

  return {
    sessionId: String(payload.session_id ?? ""),
    walletAddress: normalizeAddress(payload.wallet_address ?? userAccount.address),
    userId: Number(payload.user_id ?? userId),
    lastOrderNonce: Number(payload.last_order_nonce ?? 0),
    sessionPrivateKey
  };
}

async function submitSignedOrder(config, sessionContext, marketId, order) {
  const requestedAt = Date.now();
  const orderNonce = sessionContext.lastOrderNonce + 1;
  const message = buildOrderIntentMessage({
    sessionId: sessionContext.sessionId,
    walletAddress: sessionContext.walletAddress,
    userId: sessionContext.userId,
    marketId,
    outcome: order.outcome,
    side: order.side,
    orderType: "LIMIT",
    timeInForce: "GTC",
    price: order.price,
    quantity: order.quantity,
    clientOrderId: order.clientOrderId,
    nonce: orderNonce,
    requestedAt
  });
  const signature = toHex(
    await ed.signAsync(textEncoder.encode(message), sessionContext.sessionPrivateKey)
  );
  const payload = await postJsonOrThrow(
    config,
    `${config.apiBase}/api/v1/orders`,
    {
      user_id: sessionContext.userId,
      market_id: marketId,
      outcome: String(order.outcome).toLowerCase(),
      side: String(order.side).toLowerCase(),
      type: "limit",
      time_in_force: "gtc",
      price: order.price,
      quantity: order.quantity,
      client_order_id: order.clientOrderId,
      session_id: sessionContext.sessionId,
      session_signature: signature,
      order_nonce: orderNonce,
      requested_at: requestedAt
    },
    `create order user=${sessionContext.userId} side=${order.side} outcome=${order.outcome} price=${order.price} qty=${order.quantity}`,
    [200, 202]
  );
  sessionContext.lastOrderNonce = orderNonce;
  return payload;
}

async function detectTokenOwner(publicClient, tokenAddress) {
  try {
    return normalizeAddress(
      await publicClient.readContract({
        address: tokenAddress,
        abi: tokenAbi,
        functionName: "owner"
      })
    );
  } catch {
    return "";
  }
}

async function assertOperatorFunds(config, publicClient, operatorAccount, tokenOwner) {
  const operatorNative = await publicClient.getBalance({
    address: operatorAccount.address
  });
  const operatorToken = await publicClient.readContract({
    address: config.tokenAddress,
    abi: tokenAbi,
    functionName: "balanceOf",
    args: [operatorAccount.address]
  });

  const userFundNative = parseEther(config.fundTbnb);
  const nativeNeeded = userFundNative * BigInt(config.users) + parseEther("0.02");
  const tokenNeeded = parseUnits(config.fundUsdt, 6) * BigInt(config.users);

  if (operatorNative < nativeNeeded) {
    throw new Error(
      `operator native balance too low: have ${formatEther(operatorNative)} tBNB, need at least ${formatEther(nativeNeeded)} tBNB`
    );
  }
  if (
    tokenOwner !== normalizeAddress(operatorAccount.address) &&
    operatorToken < tokenNeeded
  ) {
    throw new Error(
      `operator token balance too low: have ${formatUnits(operatorToken, 6)} USDT, need at least ${formatUnits(tokenNeeded, 6)} USDT`
    );
  }

  return {
    operator_wallet: operatorAccount.address,
    native_tbnb: formatEther(operatorNative),
    native_needed_tbnb: formatEther(nativeNeeded),
    usdt: formatUnits(operatorToken, 6),
    usdt_needed: formatUnits(tokenNeeded, 6),
    token_owner: tokenOwner || null,
    token_funding_mode: tokenOwner === normalizeAddress(operatorAccount.address) ? "mint" : "transfer"
  };
}

async function createMarket(config, operatorAccount) {
  const nowSec = Math.floor(Date.now() / 1000);
  const market = {
    title: `Staging Concurrency ${Date.now()}`,
    description: `Script-driven concurrent matching market for TASK-STAGING-001, users=${config.users}, ordersPerUser=${config.ordersPerUser}.`,
    categoryKey: "CRYPTO",
    coverImage: "https://images.unsplash.com/photo-1621761191319-c6fb62004040?auto=format&fit=crop&w=1200&q=80",
    sourceUrl: `${config.apiBase}/`,
    sourceSlug: `staging-concurrency-${Date.now()}`,
    sourceName: "FunnyOption",
    sourceKind: "manual",
    status: "OPEN",
    collateralAsset: "USDT",
    openAt: nowSec - 60,
    closeAt: nowSec + 180,
    resolveAt: nowSec + 240,
    options: binaryOptions
  };

  const requestedAt = Date.now();
  const operator = await signOperatorAction(
    operatorAccount,
    buildCreateMarketMessage({
      walletAddress: operatorAccount.address,
      market,
      requestedAt
    }),
    requestedAt
  );
  const payload = await postJsonOrThrow(
    config,
    `${config.adminBase}/api/operator/markets`,
    {
      market,
      operator
    },
    "create market via admin route",
    [200, 201]
  );

  const marketId = Number(payload.market_id ?? 0);
  if (!marketId) {
    throw new Error(`admin create market response missing market_id: ${JSON.stringify(payload)}`);
  }

  return {
    market,
    marketId,
    response: payload
  };
}

async function issueFirstLiquidity(config, operatorAccount, marketId, quantity) {
  const bootstrap = {
    marketId,
    userId: config.makerUserId,
    quantity,
    outcome: "YES",
    price: config.bootstrapPrice
  };
  const requestedAt = Date.now();
  const operator = await signOperatorAction(
    operatorAccount,
    buildBootstrapMarketMessage({
      walletAddress: operatorAccount.address,
      bootstrap,
      requestedAt
    }),
    requestedAt
  );

  const payload = await postJsonOrThrow(
    config,
    `${config.adminBase}/api/operator/markets/${marketId}/first-liquidity`,
    {
      bootstrap,
      operator
    },
    `issue first liquidity market=${marketId}`,
    [200, 202]
  );

  return {
    bootstrap,
    response: payload,
    firstLiquidityId: String(payload.first_liquidity_id ?? ""),
    bootstrapOrderId: String(payload.order_id ?? "")
  };
}

async function verifyDuplicateBootstrap(config, operatorAccount, marketId, bootstrap, makerPositionAfterFirst, makerBalanceAfterFirst) {
  await sleep(25);
  const requestedAt = Date.now();
  const operator = await signOperatorAction(
    operatorAccount,
    buildBootstrapMarketMessage({
      walletAddress: operatorAccount.address,
      bootstrap,
      requestedAt
    }),
    requestedAt
  );

  const response = await requestJson(
    config,
    `${config.adminBase}/api/operator/markets/${marketId}/first-liquidity`,
    {
      method: "POST",
      body: {
        bootstrap,
        operator
      }
    }
  );

  const [makerPositionAfterDuplicate, makerBalanceAfterDuplicate] = await Promise.all([
    fetchPositions(config, config.makerUserId, marketId, config.makerSessionId).then((items) =>
      items.find((item) => String(item.outcome ?? "").toUpperCase() === "YES") ?? null
    ),
    fetchBalance(config, config.makerUserId, config.makerSessionId)
  ]);

  const anomalies = [];
  const beforeQty = Number(makerPositionAfterFirst?.quantity ?? 0);
  const afterQty = Number(makerPositionAfterDuplicate?.quantity ?? 0);
  const beforeBalance = Number(makerBalanceAfterFirst?.available ?? 0);
  const afterBalance = Number(makerBalanceAfterDuplicate?.available ?? 0);
  if (response.status === 409 && afterQty > beforeQty) {
    anomalies.push({
      code: "duplicate-bootstrap-side-effect-position",
      message: `duplicate bootstrap rejected with HTTP 409 but maker YES position increased from ${beforeQty} to ${afterQty}`,
      first_liquidity_id: String(response.data?.first_liquidity_id ?? "")
    });
  }
  if (response.status === 409 && afterBalance < beforeBalance) {
    anomalies.push({
      code: "duplicate-bootstrap-side-effect-balance",
      message: `duplicate bootstrap rejected with HTTP 409 but maker USDT available decreased from ${beforeBalance} to ${afterBalance}`,
      first_liquidity_id: String(response.data?.first_liquidity_id ?? "")
    });
  }

  return {
    response,
    makerPositionAfterDuplicate,
    makerBalanceAfterDuplicate,
    anomalies
  };
}

async function fundUserWallet(config, publicClient, operatorWallet, tokenOwner, userAccount) {
  const txFundNative = await operatorWallet.sendTransaction({
    to: userAccount.address,
    value: parseEther(config.fundTbnb)
  });
  logStep("fund_user_tbnb_broadcast", {
    wallet_address: userAccount.address,
    tx_hash: txFundNative
  });
  await waitTx(publicClient, txFundNative, `fund tBNB to ${userAccount.address}`);

  let txFundToken;
  if (tokenOwner === normalizeAddress(operatorWallet.account.address)) {
    txFundToken = await operatorWallet.writeContract({
      address: config.tokenAddress,
      abi: tokenAbi,
      functionName: "mint",
      args: [userAccount.address, parseUnits(config.fundUsdt, 6)]
    });
  } else {
    txFundToken = await operatorWallet.writeContract({
      address: config.tokenAddress,
      abi: tokenAbi,
      functionName: "transfer",
      args: [userAccount.address, parseUnits(config.fundUsdt, 6)]
    });
  }
  logStep("fund_user_usdt_broadcast", {
    wallet_address: userAccount.address,
    tx_hash: txFundToken
  });
  await waitTx(publicClient, txFundToken, `fund USDT to ${userAccount.address}`);

  return {
    fund_tbnb_tx: txFundNative,
    fund_usdt_tx: txFundToken
  };
}

async function approveAndDeposit(config, publicClient, userAccount, userId, sessionId) {
  const userWallet = createWalletClient({
    account: userAccount,
    chain: bscTestnet,
    transport: rpcTransport(config)
  });

  const approveTx = await userWallet.writeContract({
    address: config.tokenAddress,
    abi: tokenAbi,
    functionName: "approve",
    args: [config.vaultAddress, parseUnits(config.depositUsdt, 6)]
  });
  logStep("user_approve_broadcast", {
    user_id: userId,
    wallet_address: userAccount.address,
    tx_hash: approveTx
  });
  await waitTx(publicClient, approveTx, `approve vault user=${userId}`);

  const depositTx = await userWallet.writeContract({
    address: config.vaultAddress,
    abi: vaultAbi,
    functionName: "deposit",
    args: [parseUnits(config.depositUsdt, 6)]
  });
  logStep("user_deposit_broadcast", {
    user_id: userId,
    wallet_address: userAccount.address,
    tx_hash: depositTx
  });
  await waitTx(publicClient, depositTx, `deposit user=${userId}`);

  const deposit = await waitFor(
    `wait deposit credited user=${userId}`,
    async () => {
      const deposits = await fetchDeposits(config, userId, sessionId);
      return deposits.find((item) =>
        normalizeAddress(item.wallet_address) === normalizeAddress(userAccount.address) &&
        normalizeHex(item.tx_hash) === normalizeHex(depositTx) &&
        Number(item.credited_at ?? 0) > 0
      ) ?? null;
    },
    config.pollTimeoutMs,
    config.pollIntervalMs
  );

  const balance = await waitFor(
    `wait deposited balance user=${userId}`,
    async () => {
      const item = await fetchBalance(config, userId, sessionId);
      if (item && Number(item.available ?? 0) >= config.depositAccounting) {
        return item;
      }
      return null;
    },
    config.pollTimeoutMs,
    config.pollIntervalMs
  );

  return {
    approveTx,
    depositTx,
    deposit,
    balance
  };
}

async function prepareUsers(config, publicClient, operatorWallet, tokenOwner) {
  const baseUserId = 600_000 + Math.floor(Date.now() % 1_000_000);
  const users = [];

  for (let index = 0; index < config.users; index += 1) {
    const account = privateKeyToAccount(generatePrivateKey());
    users.push({
      userId: baseUserId + index,
      account,
      role: index < config.sellerUsers ? "SELLER" : "BUYER",
      tx: {},
      setup: null,
      session: null,
      preseedOrderId: "",
      burstOrderResults: [],
      final: null
    });
  }

  for (const user of users) {
    logStep("fund_user_wallet_start", {
      user_id: user.userId,
      role: user.role,
      wallet_address: user.account.address,
      fund_tbnb: config.fundTbnb,
      fund_usdt: config.fundUsdt
    });
    user.tx = {
      ...(await fundUserWallet(config, publicClient, operatorWallet, tokenOwner, user.account))
    };
    user.session = await createSession(config, user.account, user.userId);
    if (user.session.userId && user.session.userId !== user.userId) {
      user.userId = user.session.userId;
    }
    user.setup = await approveAndDeposit(config, publicClient, user.account, user.userId, user.session.sessionId);
    logStep("fund_user_wallet_done", {
      user_id: user.userId,
      role: user.role,
      wallet_address: user.account.address,
      session_id: user.session.sessionId,
      deposit_id: user.setup.deposit.deposit_id,
      balance_after_deposit: summarizeBalance(user.setup.balance)
    });
  }

  return users;
}

async function preseedSellerPositions(config, marketId, users) {
  const sellerUsers = users.filter((user) => user.role === "SELLER");
  for (const user of sellerUsers) {
    const clientOrderId = `stg_preseed_${marketId}_${user.userId}_${Date.now()}_${Math.random().toString(16).slice(2, 8)}`;
    const payload = await submitSignedOrder(config, user.session, marketId, {
      side: "BUY",
      outcome: "YES",
      price: config.bootstrapPrice,
      quantity: config.ordersPerUser,
      clientOrderId
    });
    user.preseedOrderId = String(payload.order_id ?? "");
    await waitFor(
      `wait seller preseed fill user=${user.userId}`,
      async () => {
        const [orders, positions] = await Promise.all([
          fetchOrders(config, user.userId, marketId, user.session.sessionId),
          fetchPositions(config, user.userId, marketId, user.session.sessionId)
        ]);
        const order = orders.find((item) => String(item.order_id ?? "") === user.preseedOrderId);
        const position = positions.find((item) => String(item.outcome ?? "").toUpperCase() === "YES");
        if (
          order &&
          String(order.status ?? "").toUpperCase() === "FILLED" &&
          position &&
          Number(position.quantity ?? 0) >= config.ordersPerUser
        ) {
          return {
            order: summarizeOrder(order),
            position: summarizePosition(position)
          };
        }
        return null;
      },
      config.pollTimeoutMs,
      config.pollIntervalMs
    );
    logStep("seller_preseed_done", {
      user_id: user.userId,
      preseed_order_id: user.preseedOrderId,
      quantity: config.ordersPerUser
    });
  }
}

async function submitBurstOrders(config, marketId, users) {
  const userResults = await runWithConcurrency(users, config.concurrency, async (user) => {
    const orderResults = [];
    for (let orderIndex = 0; orderIndex < config.ordersPerUser; orderIndex += 1) {
      const side = user.role === "SELLER" ? "SELL" : "BUY";
      const clientOrderId = `stg_burst_${marketId}_${user.userId}_${side.toLowerCase()}_${orderIndex}_${Date.now()}_${Math.random().toString(16).slice(2, 8)}`;
      const startedAt = performance.now();
      try {
        const payload = await submitSignedOrder(config, user.session, marketId, {
          side,
          outcome: "YES",
          price: config.matchPrice,
          quantity: 1,
          clientOrderId
        });
        orderResults.push({
          ok: true,
          user_id: user.userId,
          role: user.role,
          side,
          order_index: orderIndex,
          order_id: String(payload.order_id ?? ""),
          status: String(payload.status ?? ""),
          latency_ms: Number((performance.now() - startedAt).toFixed(2))
        });
      } catch (error) {
        orderResults.push({
          ok: false,
          user_id: user.userId,
          role: user.role,
          side,
          order_index: orderIndex,
          order_id: "",
          status: "FAILED",
          latency_ms: Number((performance.now() - startedAt).toFixed(2)),
          error: error instanceof Error ? error.message : String(error)
        });
      }
    }
    user.burstOrderResults = orderResults;
    return {
      user_id: user.userId,
      role: user.role,
      order_results: orderResults
    };
  });

  return {
    userResults,
    orderResults: userResults.flatMap((item) => item.order_results)
  };
}

async function collectMarketSnapshot(config, marketId, users) {
  const [market, trades, makerOrders, makerPositions, makerFreezes, makerBalance, makerPayouts] = await Promise.all([
    fetchMarket(config, marketId),
    fetchTrades(config, marketId),
    fetchOrders(config, config.makerUserId, marketId, config.makerSessionId),
    fetchPositions(config, config.makerUserId, marketId, config.makerSessionId),
    fetchFreezes(config, config.makerUserId, config.makerSessionId),
    fetchBalance(config, config.makerUserId, config.makerSessionId),
    fetchPayouts(config, config.makerUserId, marketId, config.makerSessionId)
  ]);

  const userSnapshots = await runWithConcurrency(
    users,
    Math.min(config.concurrency, users.length),
    async (user) => {
      const [orders, positions, freezes, balance, payouts] = await Promise.all([
        fetchOrders(config, user.userId, marketId, user.session.sessionId),
        fetchPositions(config, user.userId, marketId, user.session.sessionId),
        fetchFreezes(config, user.userId, user.session.sessionId),
        fetchBalance(config, user.userId, user.session.sessionId),
        fetchPayouts(config, user.userId, marketId, user.session.sessionId)
      ]);
      return {
        user_id: user.userId,
        role: user.role,
        wallet_address: user.account.address,
        session_id: user.session.sessionId,
        preseed_order_id: user.preseedOrderId,
        orders: orders.map((item) => summarizeOrder(item)),
        positions: positions.map((item) => summarizePosition(item)),
        freezes: freezes.map((item) => summarizeFreeze(item)),
        balance: summarizeBalance(balance),
        payouts: payouts.map((item) => summarizePayout(item))
      };
    }
  );

  return {
    market: {
      market_id: Number(market.market_id ?? 0),
      title: String(market.title ?? ""),
      status: String(market.status ?? ""),
      resolved_outcome: String(market.resolved_outcome ?? ""),
      runtime: market.runtime ?? {}
    },
    trades: trades.map((item) => summarizeTrade(item)),
    maker: {
      user_id: config.makerUserId,
      orders: makerOrders.map((item) => summarizeOrder(item)),
      positions: makerPositions.map((item) => summarizePosition(item)),
      freezes: makerFreezes.map((item) => summarizeFreeze(item)),
      balance: summarizeBalance(makerBalance),
      payouts: makerPayouts.map((item) => summarizePayout(item))
    },
    users: userSnapshots
  };
}

function tradeSignature(trade) {
  return [
    trade.market_id,
    trade.outcome,
    trade.price,
    trade.quantity,
    trade.taker_order_id,
    trade.maker_order_id,
    trade.taker_user_id,
    trade.maker_user_id
  ].join("|");
}

function analyzeConsistency(snapshot, submittedOrderIds, bootstrapOrderId) {
  const anomalies = [];
  const matrix = {};
  const allOrders = [
    ...snapshot.maker.orders,
    ...snapshot.users.flatMap((item) => item.orders)
  ];
  const allFreezes = [
    ...snapshot.maker.freezes,
    ...snapshot.users.flatMap((item) => item.freezes)
  ];
  const allBalances = [
    snapshot.maker.balance,
    ...snapshot.users.map((item) => item.balance)
  ].filter(Boolean);
  const orderById = new Map(allOrders.map((order) => [order.order_id, order]));
  const freezeById = new Map(allFreezes.map((freeze) => [freeze.freeze_id, freeze]));

  const duplicateTradeIds = new Set();
  const tradeIdSeen = new Set();
  const tradeSignatureCount = new Map();
  const orderFilledFromTrades = new Map();
  for (const trade of snapshot.trades) {
    if (tradeIdSeen.has(trade.trade_id)) {
      duplicateTradeIds.add(trade.trade_id);
    }
    tradeIdSeen.add(trade.trade_id);
    const signature = tradeSignature(trade);
    tradeSignatureCount.set(signature, (tradeSignatureCount.get(signature) ?? 0) + 1);
    for (const orderId of [trade.taker_order_id, trade.maker_order_id]) {
      orderFilledFromTrades.set(
        orderId,
        (orderFilledFromTrades.get(orderId) ?? 0) + Number(trade.quantity ?? 0)
      );
    }
  }

  for (const tradeId of duplicateTradeIds) {
    anomalies.push({
      code: "duplicate-trade-id",
      message: `trade_id ${tradeId} appears more than once in trade readback`
    });
  }
  for (const [signature, count] of tradeSignatureCount.entries()) {
    if (count > 1) {
      anomalies.push({
        code: "duplicate-fill",
        message: `trade signature ${signature} appears ${count} times`
      });
    }
  }
  matrix.duplicate_fill = anomalies.some((item) => item.code === "duplicate-fill" || item.code === "duplicate-trade-id") ? "FAIL" : "PASS";

  for (const [orderId, tradedQty] of orderFilledFromTrades.entries()) {
    const order = orderById.get(orderId);
    if (!order) {
      continue;
    }
    if (tradedQty > order.quantity) {
      anomalies.push({
        code: "overfill",
        message: `order ${orderId} traded quantity ${tradedQty} exceeds order quantity ${order.quantity}`,
        order: summarizeOrder(order)
      });
    }
    if (order.filled_quantity !== tradedQty) {
      anomalies.push({
        code: "fill-quantity-mismatch",
        message: `order ${orderId} reports filled_quantity=${order.filled_quantity}, but trade sum=${tradedQty}`,
        order: summarizeOrder(order)
      });
    }
  }
  matrix.overfill = anomalies.some((item) => item.code === "overfill" || item.code === "fill-quantity-mismatch") ? "FAIL" : "PASS";

  for (const balance of allBalances) {
    if (Number(balance.available ?? 0) < 0 || Number(balance.frozen ?? 0) < 0) {
      anomalies.push({
        code: "negative-balance",
        message: `user ${balance.user_id} ${balance.asset} balance has available=${balance.available}, frozen=${balance.frozen}`,
        balance
      });
    }
  }
  matrix.negative_balance = anomalies.some((item) => item.code === "negative-balance") ? "FAIL" : "PASS";

  const trackedOrderIds = new Set([bootstrapOrderId, ...submittedOrderIds].filter(Boolean));
  for (const orderId of trackedOrderIds) {
    const order = orderById.get(orderId);
    if (!order || !order.freeze_id) {
      continue;
    }
    const freeze = freezeById.get(order.freeze_id);
    if (!freeze) {
      continue;
    }
    const orderStatus = String(order.status ?? "").toUpperCase();
    const freezeStatus = String(freeze.status ?? "").toUpperCase();
    if (
      TERMINAL_ORDER_STATUSES.has(orderStatus) &&
      (!HEALTHY_FREEZE_TERMINAL_STATUSES.has(freezeStatus) || Number(freeze.remaining_amount ?? 0) !== 0)
    ) {
      anomalies.push({
        code: "stale-freeze",
        message: `order ${orderId} is terminal (${orderStatus}) but freeze ${freeze.freeze_id} has status=${freezeStatus}, remaining_amount=${freeze.remaining_amount}`,
        order: summarizeOrder(order),
        freeze: summarizeFreeze(freeze)
      });
    }
  }
  matrix.stale_freeze = anomalies.some((item) => item.code === "stale-freeze") ? "FAIL" : "PASS";

  return {
    matrix,
    anomalies
  };
}

function collectSubmittedOrderIds(users) {
  return users.flatMap((user) => [
    user.preseedOrderId,
    ...user.burstOrderResults.filter((item) => item.ok && item.order_id).map((item) => item.order_id)
  ]).filter(Boolean);
}

function collectPayoutIds(snapshot) {
  return [
    ...snapshot.maker.payouts.map((item) => item.event_id),
    ...snapshot.users.flatMap((item) => item.payouts.map((payout) => payout.event_id))
  ].filter(Boolean);
}

function allKnownOrdersTerminal(snapshot, submittedOrderIds, bootstrapOrderId) {
  const orderById = new Map([
    ...snapshot.maker.orders,
    ...snapshot.users.flatMap((item) => item.orders)
  ].map((order) => [order.order_id, order]));
  for (const orderId of [bootstrapOrderId, ...submittedOrderIds].filter(Boolean)) {
    const order = orderById.get(orderId);
    if (!order) {
      return false;
    }
    const status = String(order.status ?? "").toUpperCase();
    if (!TERMINAL_ORDER_STATUSES.has(status)) {
      return false;
    }
  }
  return true;
}

function countOpenOrders(snapshot) {
  return [
    ...snapshot.maker.orders,
    ...snapshot.users.flatMap((item) => item.orders)
  ].filter((order) => OPEN_ORDER_STATUSES.has(String(order.status ?? "").toUpperCase())).length;
}

function countSettledUsers(snapshot) {
  return snapshot.users.filter((user) =>
    user.positions.some((position) =>
      String(position.outcome ?? "").toUpperCase() === "YES" &&
      Number(position.quantity ?? 0) > 0 &&
      Number(position.settled_quantity ?? 0) >= Number(position.quantity ?? 0)
    )
  ).length;
}

async function resolveMarket(config, operatorAccount, marketId) {
  const requestedAt = Date.now();
  const operator = await signOperatorAction(
    operatorAccount,
    buildResolveMarketMessage({
      walletAddress: operatorAccount.address,
      market: {
        marketId,
        outcome: "YES"
      },
      requestedAt
    }),
    requestedAt
  );

  await postJsonOrThrow(
    config,
    `${config.adminBase}/api/operator/markets/${marketId}/resolve`,
    {
      market: {
        marketId,
        outcome: "YES"
      },
      operator
    },
    `resolve market ${marketId}`,
    [200, 202]
  );
}

async function main() {
  const startedAt = nowIso();
  const config = buildConfig(process.argv.slice(2));
  if (config.help) {
    console.log(usage());
    return;
  }

  const matrix = {};
  const anomalies = [];
  const errors = [];
  const ids = {
    market_id: 0,
    first_liquidity_id: "",
    bootstrap_order_id: "",
    duplicate_first_liquidity_id: "",
    payout_event_ids: [],
    trade_ids: [],
    concurrent_trade_ids: [],
    taker_users: []
  };
  const tx = {};
  const metrics = {
    submitted_orders: 0,
    success_orders: 0,
    failed_orders: 0,
    matched_trade_count: 0,
    matched_quantity: 0,
    remaining_open_orders_after_match: 0,
    remaining_open_orders_after_resolve: 0,
    latency_summary_ms: latencySummary([])
  };
  const reads = {};

  try {
    logStep("staging_concurrency_script_start", {
      api_base: config.apiBase,
      admin_base: config.adminBase,
      users: config.users,
      seller_users: config.sellerUsers,
      buyer_users: config.buyerUsers,
      orders_per_user: config.ordersPerUser,
      concurrency: config.concurrency,
      bootstrap_quantity: config.bootstrapQuantity,
      bootstrap_price: config.bootstrapPrice,
      match_price: config.matchPrice,
      deposit_usdt: config.depositUsdt,
      fund_tbnb: config.fundTbnb,
      fund_usdt: config.fundUsdt,
      secret_file: config.secretFile,
      secret_key: config.secretKey
    });

    const health = await getJsonOrThrow(config, `${config.apiBase}/healthz`, "healthz");
    matrix.healthz = String(health.status ?? "") === "ok" ? "PASS" : `FAIL: ${JSON.stringify(health)}`;

    const operatorAccount = readOperatorAccount(config);
    matrix.operator_secret_loaded = "PASS";

    const publicClient = createPublicClient({
      chain: bscTestnet,
      transport: rpcTransport(config)
    });
    const operatorWallet = createWalletClient({
      account: operatorAccount,
      chain: bscTestnet,
      transport: rpcTransport(config)
    });

    const tokenOwner = await detectTokenOwner(publicClient, config.tokenAddress);
    reads.operator_funds = await assertOperatorFunds(config, publicClient, operatorAccount, tokenOwner);
    matrix.operator_wallet_funded = "PASS";
    ids.operator_wallet = operatorAccount.address;

    const makerSession = await createSession(config, operatorAccount, config.makerUserId);
    if (makerSession.userId && makerSession.userId !== config.makerUserId) {
      config.makerUserId = makerSession.userId;
    }
    config.makerSessionId = makerSession.sessionId;
    logStep("maker_session_created", { session_id: makerSession.sessionId, user_id: config.makerUserId });

    reads.maker_balance_before = summarizeBalance(await fetchBalance(config, config.makerUserId, config.makerSessionId));

    logStep("create_market_start");
    const createdMarket = await createMarket(config, operatorAccount);
    ids.market_id = createdMarket.marketId;
    reads.created_market = {
      market_id: createdMarket.marketId,
      title: createdMarket.market.title,
      response: createdMarket.response
    };
    matrix.admin_create_market = "PASS";
    logStep("create_market_done", {
      market_id: ids.market_id,
      title: createdMarket.market.title
    });

    logStep("first_liquidity_start", {
      market_id: ids.market_id,
      maker_user_id: config.makerUserId,
      quantity: config.bootstrapQuantity,
      price: config.bootstrapPrice
    });
    const firstLiquidity = await issueFirstLiquidity(
      config,
      operatorAccount,
      ids.market_id,
      config.bootstrapQuantity
    );
    ids.first_liquidity_id = firstLiquidity.firstLiquidityId;
    ids.bootstrap_order_id = firstLiquidity.bootstrapOrderId;
    if (!ids.first_liquidity_id || !ids.bootstrap_order_id) {
      throw new Error(`first liquidity response missing ids: ${JSON.stringify(firstLiquidity.response)}`);
    }
    matrix.admin_first_liquidity = "PASS";

    const makerFirstSnapshot = await waitFor(
      "wait maker bootstrap order and inventory",
      async () => {
        const [orders, positions, balance] = await Promise.all([
          fetchOrders(config, config.makerUserId, ids.market_id, config.makerSessionId),
          fetchPositions(config, config.makerUserId, ids.market_id, config.makerSessionId),
          fetchBalance(config, config.makerUserId, config.makerSessionId)
        ]);
        const order = orders.find((item) => String(item.order_id ?? "") === ids.bootstrap_order_id);
        const position = positions.find((item) => String(item.outcome ?? "").toUpperCase() === "YES");
        if (
          order &&
          OPEN_ORDER_STATUSES.has(String(order.status ?? "").toUpperCase()) &&
          Number(order.remaining_quantity ?? 0) >= config.bootstrapQuantity &&
          position &&
          Number(position.quantity ?? 0) >= config.bootstrapQuantity &&
          balance
        ) {
          return {
            order,
            position,
            balance
          };
        }
        return null;
      },
      config.pollTimeoutMs,
      config.pollIntervalMs
    );
    reads.maker_bootstrap_order = summarizeOrder(makerFirstSnapshot.order);
    reads.maker_position_after_first_bootstrap = summarizePosition(makerFirstSnapshot.position);
    reads.maker_balance_after_first_bootstrap = summarizeBalance(makerFirstSnapshot.balance);

    logStep("duplicate_bootstrap_check_start", {
      market_id: ids.market_id,
      quantity: firstLiquidity.bootstrap.quantity,
      price: firstLiquidity.bootstrap.price
    });
    const duplicateBootstrapResult = await verifyDuplicateBootstrap(
      config,
      operatorAccount,
      ids.market_id,
      firstLiquidity.bootstrap,
      makerFirstSnapshot.position,
      makerFirstSnapshot.balance
    );
    matrix.duplicate_bootstrap_reject = duplicateBootstrapResult.response.status === 409
      ? "PASS"
      : `FAIL: HTTP ${duplicateBootstrapResult.response.status} ${duplicateBootstrapResult.response.text}`;
    ids.duplicate_first_liquidity_id = String(
      duplicateBootstrapResult.response.data?.first_liquidity_id ?? ""
    );
    reads.duplicate_bootstrap_response = {
      status: duplicateBootstrapResult.response.status,
      body: duplicateBootstrapResult.response.data
    };
    reads.maker_position_after_duplicate_bootstrap = summarizePosition(
      duplicateBootstrapResult.makerPositionAfterDuplicate
    );
    reads.maker_balance_after_duplicate_bootstrap = summarizeBalance(
      duplicateBootstrapResult.makerBalanceAfterDuplicate
    );
    anomalies.push(...duplicateBootstrapResult.anomalies);
    matrix.bootstrap_atomicity = duplicateBootstrapResult.anomalies.length === 0 ? "PASS" : "FAIL";

    const firstBootstrapDebit = Number(reads.maker_balance_before?.available ?? 0) -
      Number(reads.maker_balance_after_first_bootstrap?.available ?? 0);
    const expectedBootstrapDebit = config.bootstrapQuantity * 100;
    if (firstBootstrapDebit !== expectedBootstrapDebit) {
      anomalies.push({
        code: "first-liquidity-collateral-unit-mismatch",
        message: `maker USDT available decreased by ${firstBootstrapDebit} for first bootstrap quantity=${config.bootstrapQuantity}, expected ${expectedBootstrapDebit} accounting units`,
        maker_position_after_first: reads.maker_position_after_first_bootstrap
      });
      matrix.first_liquidity_collateral = "FAIL";
    } else {
      matrix.first_liquidity_collateral = "PASS";
    }
    logStep("duplicate_bootstrap_check_done", {
      status: duplicateBootstrapResult.response.status,
      first_liquidity_id: ids.duplicate_first_liquidity_id || null
    });

    logStep("prepare_users_start");
    const users = await prepareUsers(config, publicClient, operatorWallet, tokenOwner);
    ids.taker_users = users.map((user) => ({
      user_id: user.userId,
      wallet_address: user.account.address,
      session_id: user.session.sessionId,
      role: user.role
    }));
    matrix.user_sessions_and_deposits = users.every((user) =>
      user.session?.sessionId &&
      user.setup?.deposit?.deposit_id &&
      Number(user.setup?.balance?.available ?? 0) >= config.depositAccounting
    ) ? "PASS" : "FAIL";
    logStep("prepare_users_done", {
      users: ids.taker_users
    });

    logStep("preseed_sellers_start", {
      seller_users: config.sellerUsers,
      quantity_per_user: config.ordersPerUser
    });
    await preseedSellerPositions(config, ids.market_id, users);
    matrix.seller_preseed_inventory = users
      .filter((user) => user.role === "SELLER")
      .every((user) => user.preseedOrderId)
      ? "PASS"
      : "FAIL";
    reads.preseed_order_ids = users
      .filter((user) => user.preseedOrderId)
      .map((user) => ({
        user_id: user.userId,
        role: user.role,
        order_id: user.preseedOrderId
      }));

    const tradesBeforeBurst = await fetchTrades(config, ids.market_id);
    const tradeIdsBeforeBurst = new Set(
      tradesBeforeBurst.map((item) => String(item.trade_id ?? "")).filter(Boolean)
    );

    logStep("concurrent_burst_start", {
      market_id: ids.market_id,
      users: config.users,
      orders_per_user: config.ordersPerUser,
      concurrency: config.concurrency,
      buyer_users: config.buyerUsers,
      seller_users: config.sellerUsers,
      price: config.matchPrice
    });
    const burstResult = await submitBurstOrders(config, ids.market_id, users);
    metrics.submitted_orders = burstResult.orderResults.length;
    metrics.success_orders = burstResult.orderResults.filter((item) => item.ok).length;
    metrics.failed_orders = burstResult.orderResults.filter((item) => !item.ok).length;
    metrics.latency_summary_ms = latencySummary(
      burstResult.orderResults.map((item) => Number(item.latency_ms ?? 0))
    );
    matrix.concurrent_order_submit = metrics.failed_orders === 0
      ? "PASS"
      : `FAIL: ${metrics.failed_orders}/${metrics.submitted_orders} order request(s) failed`;

    const successBuyQty = burstResult.orderResults.filter((item) => item.ok && item.side === "BUY").length;
    const successSellQty = burstResult.orderResults.filter((item) => item.ok && item.side === "SELL").length;
    const expectedBurstMatchQty = Math.min(successBuyQty, successSellQty);

    const snapshotAfterMatch = await waitFor(
      "wait concurrent matching settle",
      async () => {
        const snapshot = await collectMarketSnapshot(config, ids.market_id, users);
        const newTrades = snapshot.trades.filter((trade) => !tradeIdsBeforeBurst.has(trade.trade_id));
        const matchedQty = newTrades.reduce((sum, trade) => sum + Number(trade.quantity ?? 0), 0);
        const knownSubmittedOrders = collectSubmittedOrderIds(users).filter((orderId) =>
          burstResult.orderResults.some((result) => result.order_id === orderId)
        );
        const orderById = new Map([
          ...snapshot.maker.orders,
          ...snapshot.users.flatMap((user) => user.orders)
        ].map((order) => [order.order_id, order]));
        const hasQueuedOrders = knownSubmittedOrders.some((orderId) =>
          String(orderById.get(orderId)?.status ?? "").toUpperCase() === "QUEUED"
        );
        if (!hasQueuedOrders && matchedQty >= expectedBurstMatchQty) {
          return {
            snapshot,
            newTrades,
            matchedQty
          };
        }
        return null;
      },
      config.pollTimeoutMs,
      config.pollIntervalMs
    );

    metrics.matched_trade_count = snapshotAfterMatch.newTrades.length;
    metrics.matched_quantity = snapshotAfterMatch.matchedQty;
    metrics.remaining_open_orders_after_match = countOpenOrders(snapshotAfterMatch.snapshot);
    ids.concurrent_trade_ids = snapshotAfterMatch.newTrades.map((trade) => trade.trade_id);
    ids.trade_ids = snapshotAfterMatch.snapshot.trades.map((trade) => trade.trade_id);
    matrix.concurrent_matching = metrics.matched_quantity >= expectedBurstMatchQty ? "PASS" : "FAIL";
    reads.snapshot_after_match = snapshotAfterMatch.snapshot;
    reads.burst_orders = burstResult.userResults;
    logStep("concurrent_burst_done", {
      submitted_orders: metrics.submitted_orders,
      success_orders: metrics.success_orders,
      failed_orders: metrics.failed_orders,
      expected_match_quantity: expectedBurstMatchQty,
      matched_trade_count: metrics.matched_trade_count,
      matched_quantity: metrics.matched_quantity,
      remaining_open_orders_after_match: metrics.remaining_open_orders_after_match,
      latency_summary_ms: metrics.latency_summary_ms
    });

    const marketDetail = await fetchMarket(config, ids.market_id);
    const resolveAtSec = Number(marketDetail.resolve_at ?? 0);
    const resolveWaitSec = Math.max(0, resolveAtSec - Math.floor(Date.now() / 1000) + 2);
    if (resolveWaitSec > 0) {
      logStep("wait_market_resolution_window", { resolve_at: resolveAtSec, wait_seconds: resolveWaitSec });
      await sleep(resolveWaitSec * 1000);
    }

    logStep("resolve_market_start", {
      market_id: ids.market_id,
      outcome: "YES"
    });
    await resolveMarket(config, operatorAccount, ids.market_id);
    matrix.admin_resolve_market = "PASS";

    const submittedOrderIds = collectSubmittedOrderIds(users);
    const finalSnapshot = await waitFor(
      "wait final settlement",
      async () => {
        const snapshot = await collectMarketSnapshot(config, ids.market_id, users);
        if (
          String(snapshot.market.status ?? "").toUpperCase() === "RESOLVED" &&
          String(snapshot.market.resolved_outcome ?? "").toUpperCase() === "YES" &&
          allKnownOrdersTerminal(snapshot, submittedOrderIds, ids.bootstrap_order_id)
        ) {
          return snapshot;
        }
        return null;
      },
      config.pollTimeoutMs,
      config.pollIntervalMs
    );
    metrics.remaining_open_orders_after_resolve = countOpenOrders(finalSnapshot);
    ids.payout_event_ids = collectPayoutIds(finalSnapshot);
    matrix.final_market_resolved = finalSnapshot.market.status === "RESOLVED" ? "PASS" : "FAIL";
    matrix.final_payouts_read = countSettledUsers(finalSnapshot) > 0 && ids.payout_event_ids.length > 0 ? "PASS" : "FAIL";

    const consistency = analyzeConsistency(
      finalSnapshot,
      submittedOrderIds,
      ids.bootstrap_order_id
    );
    Object.assign(matrix, consistency.matrix);
    anomalies.push(...consistency.anomalies);
    reads.final_snapshot = finalSnapshot;

    const status = Object.values(matrix).every((value) => value === "PASS") && anomalies.length === 0
      ? "PASS"
      : "FAIL";

    logStep("staging_concurrency_script_done", {
      status,
      market_id: ids.market_id,
      matched_trade_count: metrics.matched_trade_count,
      remaining_open_orders_after_match: metrics.remaining_open_orders_after_match,
      remaining_open_orders_after_resolve: metrics.remaining_open_orders_after_resolve,
      payout_event_ids: ids.payout_event_ids,
      anomalies: anomalies.map((item) => item.code)
    });
    console.log(
      `SUMMARY status=${status} market_id=${ids.market_id} success_orders=${metrics.success_orders} failed_orders=${metrics.failed_orders} matched_trade_count=${metrics.matched_trade_count} remaining_open_orders_after_match=${metrics.remaining_open_orders_after_match} p95_latency_ms=${metrics.latency_summary_ms.p95_ms} anomalies=${anomalies.length}`
    );
    for (const anomaly of anomalies) {
      console.log(`ANOMALY code=${anomaly.code} message=${anomaly.message}`);
    }
    console.log("###FUNNYOPTION_STAGING_CONCURRENCY_JSON###");
    console.log(JSON.stringify({
      status,
      started_at: startedAt,
      finished_at: nowIso(),
      config: {
        api_base: config.apiBase,
        admin_base: config.adminBase,
        rpc_url: config.rpcUrl,
        token_address: config.tokenAddress,
        vault_address: config.vaultAddress,
        secret_file: config.secretFile,
        secret_key: config.secretKey,
        maker_user_id: config.makerUserId,
        users: config.users,
        seller_users: config.sellerUsers,
        buyer_users: config.buyerUsers,
        orders_per_user: config.ordersPerUser,
        total_burst_orders: config.totalBurstOrders,
        concurrency: config.concurrency,
        bootstrap_price: config.bootstrapPrice,
        match_price: config.matchPrice,
        bootstrap_quantity: config.bootstrapQuantity,
        deposit_usdt: config.depositUsdt,
        fund_tbnb: config.fundTbnb,
        fund_usdt: config.fundUsdt,
        http_timeout_ms: config.httpTimeoutMs,
        poll_timeout_ms: config.pollTimeoutMs,
        poll_interval_ms: config.pollIntervalMs
      },
      matrix,
      metrics,
      ids,
      tx,
      reads,
      anomalies,
      errors
    }, null, 2));

    if (status !== "PASS") {
      process.exitCode = 1;
    }
  } catch (error) {
    errors.push(error instanceof Error ? error.message : String(error));
    console.log("###FUNNYOPTION_STAGING_CONCURRENCY_JSON###");
    console.log(JSON.stringify({
      status: "FAIL",
      started_at: startedAt,
      finished_at: nowIso(),
      config: {
        api_base: config.apiBase,
        admin_base: config.adminBase,
        rpc_url: config.rpcUrl,
        token_address: config.tokenAddress,
        vault_address: config.vaultAddress,
        secret_file: config.secretFile,
        secret_key: config.secretKey,
        maker_user_id: config.makerUserId,
        users: config.users,
        seller_users: config.sellerUsers,
        buyer_users: config.buyerUsers,
        orders_per_user: config.ordersPerUser,
        total_burst_orders: config.totalBurstOrders,
        concurrency: config.concurrency,
        bootstrap_price: config.bootstrapPrice,
        match_price: config.matchPrice,
        bootstrap_quantity: config.bootstrapQuantity,
        deposit_usdt: config.depositUsdt,
        fund_tbnb: config.fundTbnb,
        fund_usdt: config.fundUsdt,
        http_timeout_ms: config.httpTimeoutMs,
        poll_timeout_ms: config.pollTimeoutMs,
        poll_interval_ms: config.pollIntervalMs
      },
      matrix,
      metrics,
      ids,
      tx,
      reads,
      anomalies,
      errors
    }, null, 2));
    process.exitCode = 1;
  }
}

await main();
