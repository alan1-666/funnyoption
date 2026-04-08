#!/usr/bin/env node

/**
 * Oracle E2E smoke test — creates a CRYPTO oracle market on staging with
 * resolve_at ~2 minutes from now, then polls until the oracle worker
 * auto-resolves it via Binance price feed.
 *
 * Usage:
 *   node scripts/staging-oracle-e2e.mjs
 *   node scripts/staging-oracle-e2e.mjs --resolve-delay 180
 */

import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { setTimeout as sleep } from "node:timers/promises";
import {
  createWalletClient,
  http
} from "../web/node_modules/viem/_esm/index.js";
import { bscTestnet } from "../web/node_modules/viem/_esm/chains/index.js";
import { privateKeyToAccount } from "../web/node_modules/viem/_esm/accounts/index.js";

const DEFAULT_SECRET_FILE = fileURLToPath(new URL("../.secrets", import.meta.url));
const DEFAULT_API_BASE = "https://funnyoption.xyz";
const DEFAULT_ADMIN_BASE = "https://admin.funnyoption.xyz";
const DEFAULT_RESOLVE_DELAY_SEC = 120; // 2 minutes
const POLL_INTERVAL_MS = 5_000;
const MAX_WAIT_AFTER_RESOLVE_AT_SEC = 120; // wait up to 2 min after resolve_at

// ── helpers ──

function nowIso() { return new Date().toISOString(); }
function log(step, payload) {
  if (payload === undefined) console.log(`[${nowIso()}] ${step}`);
  else console.log(`[${nowIso()}] ${step}`, JSON.stringify(payload));
}

function normalizeAddress(v) { return String(v ?? "").trim().toLowerCase(); }
function cleanText(v) { return String(v ?? "").trim().replace(/\s+/g, " "); }

function optionFragment(options) {
  return [...options]
    .map((o, i) => ({
      key: cleanText(o.key).toUpperCase().replace(/\s+/g, "_"),
      label: cleanText(o.label),
      shortLabel: cleanText(o.shortLabel ?? o.label),
      sortOrder: Math.max(1, Math.floor(Number(o.sortOrder || (i + 1) * 10))),
      isActive: o.isActive !== false
    }))
    .sort((a, b) => a.sortOrder - b.sortOrder || a.key.localeCompare(b.key))
    .map(o => `${o.key}:${o.label}:${o.shortLabel}:${o.sortOrder}:${o.isActive ? "1" : "0"}`)
    .join("|");
}

function buildResolutionSignatureFragment(resolution) {
  if (!resolution) return "";
  const oracle = resolution.oracle ?? {};
  const instrument = oracle.instrument ?? {};
  const price = oracle.price ?? {};
  const window = oracle.window ?? {};
  const rule = oracle.rule ?? {};
  return `resolution_version: ${Math.floor(resolution.version ?? 0)}
resolution_mode: ${cleanText(resolution.mode ?? "").toUpperCase()}
resolution_market_kind: ${cleanText(resolution.market_kind ?? "").toUpperCase()}
resolution_manual_fallback_allowed: ${resolution.manual_fallback_allowed === true}
oracle_source_kind: ${cleanText(oracle.source_kind ?? "").toUpperCase()}
oracle_provider_key: ${cleanText(oracle.provider_key ?? "").toUpperCase()}
oracle_instrument_kind: ${cleanText(instrument.kind ?? "").toUpperCase()}
oracle_instrument_base_asset: ${cleanText(instrument.base_asset ?? "").toUpperCase()}
oracle_instrument_quote_asset: ${cleanText(instrument.quote_asset ?? "").toUpperCase()}
oracle_instrument_symbol: ${cleanText(instrument.symbol ?? "").toUpperCase()}
oracle_price_field: ${cleanText(price.field ?? "").toUpperCase()}
oracle_price_scale: ${Math.floor(price.scale ?? 0)}
oracle_price_rounding_mode: ${cleanText(price.rounding_mode ?? "").toUpperCase()}
oracle_price_max_data_age_sec: ${Math.floor(price.max_data_age_sec ?? 0)}
oracle_window_anchor: ${cleanText(window.anchor ?? "").toUpperCase()}
oracle_window_before_sec: ${Math.floor(window.before_sec ?? 0)}
oracle_window_after_sec: ${Math.floor(window.after_sec ?? 0)}
oracle_rule_type: ${cleanText(rule.type ?? "").toUpperCase()}
oracle_rule_comparator: ${cleanText(rule.comparator ?? "").toUpperCase()}
oracle_rule_threshold_price: ${(rule.threshold_price ?? "").trim()}
`;
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
${buildResolutionSignatureFragment(market.resolution)}options: ${optionFragment(market.options)}
`;
}

async function postJson(base, path, body) {
  const url = `${base}${path}`;
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body)
  });
  const text = await res.text();
  let json;
  try { json = JSON.parse(text); } catch { json = null; }
  if (!res.ok) throw new Error(`POST ${path} → ${res.status}: ${text.slice(0, 300)}`);
  return json;
}

async function getJson(base, path) {
  const url = `${base}${path}`;
  const res = await fetch(url);
  const text = await res.text();
  let json;
  try { json = JSON.parse(text); } catch { json = null; }
  if (!res.ok) throw new Error(`GET ${path} → ${res.status}: ${text.slice(0, 300)}`);
  return json;
}

// ── config ──

function parseArgs() {
  const args = process.argv.slice(2);
  const raw = {};
  for (let i = 0; i < args.length; i++) {
    const key = args[i].replace(/^--/, "");
    raw[key] = args[i + 1] ?? "true";
    i++;
  }
  return {
    apiBase: String(raw["api-base"] ?? DEFAULT_API_BASE).replace(/\/+$/, ""),
    adminBase: String(raw["admin-base"] ?? DEFAULT_ADMIN_BASE).replace(/\/+$/, ""),
    secretFile: resolve(process.cwd(), String(raw["secret-file"] ?? DEFAULT_SECRET_FILE)),
    secretKey: String(raw["secret-key"] ?? "bsc-testnet-operator.key").trim(),
    resolveDelaySec: Math.max(60, Number(raw["resolve-delay"] ?? DEFAULT_RESOLVE_DELAY_SEC)),
    // BTC threshold — set low enough that current BTC price will be above it → YES
    thresholdPrice: String(raw["threshold-price"] ?? "10000.00000000")
  };
}

function loadOperatorKey(secretFile, secretKey) {
  const content = readFileSync(secretFile, "utf-8").trim();
  for (const line of content.split("\n")) {
    const [key, value] = line.split(":").map(s => s.trim());
    if (key === secretKey) return value;
  }
  throw new Error(`key "${secretKey}" not found in ${secretFile}`);
}

// ── main ──

async function main() {
  const config = parseArgs();
  log("config", {
    apiBase: config.apiBase,
    adminBase: config.adminBase,
    resolveDelaySec: config.resolveDelaySec,
    thresholdPrice: config.thresholdPrice
  });

  // 1. Load operator
  const operatorPrivateKey = loadOperatorKey(config.secretFile, config.secretKey);
  const operatorAccount = privateKeyToAccount(`0x${operatorPrivateKey}`);
  log("operator", { address: operatorAccount.address });

  // 2. Create oracle market
  const nowSec = Math.floor(Date.now() / 1000);
  const resolveAt = nowSec + config.resolveDelaySec;
  const closeAt = resolveAt; // close = resolve for oracle markets

  const market = {
    title: `Oracle E2E ${Date.now()}`,
    description: "Automated oracle E2E test: BTC/USDT price threshold via Binance.",
    categoryKey: "CRYPTO",
    coverImage: "",
    sourceUrl: `${config.apiBase}/`,
    sourceSlug: `oracle-e2e-${Date.now()}`,
    sourceName: "FunnyOption",
    sourceKind: "manual",
    status: "OPEN",
    collateralAsset: "USDT",
    openAt: nowSec - 60,
    closeAt,
    resolveAt,
    options: [
      { key: "YES", label: "是", shortLabel: "是", sortOrder: 10, isActive: true },
      { key: "NO", label: "否", shortLabel: "否", sortOrder: 20, isActive: true }
    ],
    resolution: {
      version: 1,
      mode: "ORACLE_PRICE",
      market_kind: "CRYPTO_PRICE_THRESHOLD",
      manual_fallback_allowed: true,
      oracle: {
        source_kind: "HTTP_JSON",
        provider_key: "BINANCE",
        instrument: {
          kind: "SPOT",
          base_asset: "BTC",
          quote_asset: "USDT",
          symbol: "BTCUSDT"
        },
        price: {
          field: "LAST_PRICE",
          scale: 8,
          rounding_mode: "ROUND_HALF_UP",
          max_data_age_sec: 300
        },
        window: {
          anchor: "RESOLVE_AT",
          before_sec: 600,
          after_sec: 600
        },
        rule: {
          type: "PRICE_THRESHOLD",
          comparator: "GTE",
          threshold_price: config.thresholdPrice
        }
      }
    }
  };

  const requestedAt = Date.now();
  const operator = {
    walletAddress: operatorAccount.address,
    requestedAt,
    signature: await operatorAccount.signMessage({
      message: buildCreateMarketMessage({ walletAddress: operatorAccount.address, market, requestedAt })
    })
  };

  log("creating_oracle_market", { resolveAt, closeAt, thresholdPrice: config.thresholdPrice });
  const createResult = await postJson(config.adminBase, "/api/operator/markets", { market, operator });
  const marketId = Number(createResult.market_id ?? 0);
  if (!marketId) throw new Error(`create market failed: ${JSON.stringify(createResult)}`);
  log("market_created", { marketId, resolveAt: new Date(resolveAt * 1000).toISOString() });

  // 3. Poll until resolved or timeout
  const deadline = (resolveAt + MAX_WAIT_AFTER_RESOLVE_AT_SEC) * 1000;
  let resolved = false;
  let lastStatus = "";
  let pollCount = 0;

  log("waiting_for_resolution", {
    resolveAt: new Date(resolveAt * 1000).toISOString(),
    deadline: new Date(deadline).toISOString()
  });

  while (Date.now() < deadline) {
    pollCount++;
    try {
      const marketData = await getJson(config.apiBase, `/api/v1/markets/${marketId}`);
      const status = marketData.status ?? marketData.item?.status ?? "UNKNOWN";
      if (status !== lastStatus) {
        log("status_change", { marketId, status, poll: pollCount });
        lastStatus = status;
      }
      if (status === "RESOLVED") {
        const outcome = marketData.resolved_outcome ?? marketData.item?.resolved_outcome ?? "?";
        log("RESOLVED", { marketId, outcome, polls: pollCount });
        resolved = true;
        break;
      }
    } catch (err) {
      log("poll_error", { marketId, error: err.message, poll: pollCount });
    }
    await sleep(POLL_INTERVAL_MS);
  }

  // 4. Check oracle debug stats
  try {
    const oracleDebug = await getJson(config.apiBase.replace(/:\d+$/, ":9191").replace("https://funnyoption.xyz", "http://127.0.0.1:9191"), "/debug/oracle");
    log("oracle_debug", oracleDebug?.stats ?? oracleDebug);
  } catch {
    log("oracle_debug", { note: "could not reach oracle debug endpoint (expected if running remotely)" });
  }

  // 5. Verdict
  if (resolved) {
    log("✅ PASS", { marketId, message: "Oracle auto-resolved the market via Binance price feed" });
    process.exit(0);
  } else {
    log("❌ FAIL", { marketId, lastStatus, message: `Market not resolved after ${MAX_WAIT_AFTER_RESOLVE_AT_SEC}s past resolve_at` });
    process.exit(1);
  }
}

main().catch(err => {
  console.error(`[${nowIso()}] FATAL:`, err);
  process.exit(1);
});
