#!/usr/bin/env node

/**
 * Seed Markets — Create curated prediction markets on staging
 *
 * Creates ~12 markets across CRYPTO and SPORTS categories
 * with cover images sourced from worm.wtf.
 *
 * Usage:
 *   node scripts/staging-seed-markets.mjs
 *
 * Environment:
 *   ADMIN_BASE  — admin API origin (default: https://admin.funnyoption.xyz)
 *   FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY — operator wallet key (or read from .secrets)
 */

import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { setTimeout as sleep } from "node:timers/promises";
import {
  privateKeyToAccount,
} from "../web/node_modules/viem/_esm/accounts/index.js";

const ADMIN_BASE = (process.env.ADMIN_BASE || "https://admin.funnyoption.xyz").replace(/\/+$/, "");
const SECRET_FILE = resolve(fileURLToPath(new URL(".", import.meta.url)), "../.secrets");
const HTTP_TIMEOUT = 30_000;

const binaryOptions = [
  { key: "YES", label: "Yes", shortLabel: "Yes", sortOrder: 10, isActive: true },
  { key: "NO", label: "No", shortLabel: "No", sortOrder: 20, isActive: true },
];

function norm(v) { return String(v ?? "").trim().toLowerCase(); }
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
function die(msg) { console.error(`\n❌ FAIL: ${msg}\n`); process.exit(1); }
async function postOk(url, body, label) { const r = await fetchJson(url, { method: "POST", body }); if (!r.ok) die(`${label}: HTTP ${r.status} ${r.text}`); return r.data ?? {}; }

async function signOp(account, message, requestedAt) {
  return { walletAddress: account.address, requestedAt, signature: await account.signMessage({ message }) };
}

function buildCreateMarketMsg({ walletAddress, market, requestedAt }) {
  return `FunnyOption Operator Authorization\n\naction: CREATE_MARKET\nwallet: ${norm(walletAddress)}\ntitle: ${clean(market.title)}\ndescription: ${clean(market.description)}\ncategory: ${clean(market.categoryKey).toUpperCase() || "CRYPTO"}\nsource_kind: ${clean(market.sourceKind).toLowerCase() || "manual"}\nsource_url: ${String(market.sourceUrl ?? "").trim()}\nsource_slug: ${clean(market.sourceSlug)}\nsource_name: ${clean(market.sourceName) || "FunnyOption"}\ncover_image: ${String(market.coverImage ?? "").trim()}\nstatus: ${clean(market.status).toUpperCase() || "OPEN"}\ncollateral_asset: ${clean(market.collateralAsset).toUpperCase() || "USDT"}\nopen_at: ${Math.max(0, Math.floor(market.openAt || 0))}\nclose_at: ${Math.max(0, Math.floor(market.closeAt || 0))}\nresolve_at: ${Math.max(0, Math.floor(market.resolveAt || 0))}\nrequested_at: ${Math.floor(requestedAt)}\noptions: ${optionFrag(market.options)}\n`;
}

const IMG_BASE = "https://api.worm.wtf/media";

const MARKETS = [
  // ── CRYPTO ──
  {
    title: "Will Bitcoin (BTC) hit $100,000 by December 2026?",
    description: "Resolves YES if BTC/USD trades at or above $100,000 on any major exchange before December 31, 2026 23:59 UTC.",
    categoryKey: "CRYPTO",
    coverImage: `${IMG_BASE}/market_groups/logos/event_what-price-will-bitcoin-hit-in-april-2026.jpg`,
    sourceSlug: "btc-100k-2026",
  },
  {
    title: "Will Ethereum (ETH) hit $5,000 by December 2026?",
    description: "Resolves YES if ETH/USD trades at or above $5,000 on any major exchange before December 31, 2026 23:59 UTC.",
    categoryKey: "CRYPTO",
    coverImage: `${IMG_BASE}/market_groups/logos/event_what-price-will-ethereum-hit-in-april-2026.jpg`,
    sourceSlug: "eth-5k-2026",
  },
  {
    title: "Will MegaETH FDV exceed $1B on launch day?",
    description: "Resolves YES if MegaETH's fully diluted valuation exceeds $1 billion within 24 hours of its token launch.",
    categoryKey: "CRYPTO",
    coverImage: `${IMG_BASE}/market_groups/logos/event_megaeth-market-cap-fdv-one-day-after-launch.jpg`,
    sourceSlug: "megaeth-fdv-1b",
  },
  {
    title: "Will OpenSea launch a token by end of 2026?",
    description: "Resolves YES if OpenSea officially launches a fungible token tradeable on any exchange before December 31, 2026.",
    categoryKey: "CRYPTO",
    coverImage: `${IMG_BASE}/market_groups/logos/event_opensea-fdv-above-one-day-after-launch.jpg`,
    sourceSlug: "opensea-token-2026",
  },
  {
    title: "Will MetaMask launch a token by end of 2026?",
    description: "Resolves YES if ConsenSys / MetaMask officially launches a fungible token before December 31, 2026.",
    categoryKey: "CRYPTO",
    coverImage: `${IMG_BASE}/market_groups/logos/event_will-metamask-launch-a-token-in-2025.jpg`,
    sourceSlug: "metamask-token-2026",
  },
  {
    title: "Will Solana (SOL) hit $300 by December 2026?",
    description: "Resolves YES if SOL/USD trades at or above $300 on any major exchange before December 31, 2026 23:59 UTC.",
    categoryKey: "CRYPTO",
    coverImage: `${IMG_BASE}/market_groups/logos/event_what-price-will-bitcoin-hit-in-april-2026.jpg`,
    sourceSlug: "sol-300-2026",
  },
  {
    title: "Will the Fed cut rates before July 2026?",
    description: "Resolves YES if the US Federal Reserve announces at least one 25+ bps rate cut at any FOMC meeting before July 1, 2026.",
    categoryKey: "CRYPTO",
    coverImage: `${IMG_BASE}/market_groups/logos/event_fed-decision-in-april.jpg`,
    sourceSlug: "fed-cut-h1-2026",
  },
  {
    title: "Will WTI Crude Oil hit $120 in 2026?",
    description: "Resolves YES if WTI front-month futures trade at or above $120/barrel at any point before December 31, 2026.",
    categoryKey: "CRYPTO",
    coverImage: `${IMG_BASE}/market_groups/logos/event_what-price-will-wti-hit-in-april-2026.jpg`,
    sourceSlug: "wti-120-2026",
  },
  // ── SPORTS ──
  {
    title: "Will Spain win the 2026 FIFA World Cup?",
    description: "Resolves YES if the Spain national football team wins the 2026 FIFA World Cup final.",
    categoryKey: "SPORTS",
    coverImage: `${IMG_BASE}/events/logos/fifa.png`,
    sourceSlug: "spain-wc-2026",
  },
  {
    title: "Will Arsenal win the UEFA Champions League 2025-26?",
    description: "Resolves YES if Arsenal FC wins the 2025-26 UEFA Champions League final.",
    categoryKey: "SPORTS",
    coverImage: `${IMG_BASE}/market_groups/logos/unnamed_5.png`,
    sourceSlug: "arsenal-ucl-2026",
  },
  {
    title: "Will the Democrats win the US Senate in 2026?",
    description: "Resolves YES if the Democratic Party holds a majority (or 50 seats + VP tiebreaker) in the US Senate after the 2026 midterm elections.",
    categoryKey: "SPORTS",
    coverImage: `${IMG_BASE}/events/logos/event_which-party-will-win-the-senate-in-2026.jpg`,
    sourceSlug: "dems-senate-2026",
  },
  {
    title: "Will the US confirm alien existence before 2027?",
    description: "Resolves YES if any official US government agency publicly confirms the existence of extraterrestrial life before January 1, 2027.",
    categoryKey: "CRYPTO",
    coverImage: `${IMG_BASE}/market_groups/logos/77bf8ef5-08b9-400e-bf6b-4c7c32a29fa4.jpg`,
    sourceSlug: "aliens-2027",
  },
];

async function main() {
  console.log("╔════════════════════════════════════════════════╗");
  console.log("║  Seed Markets — Curated Staging Data           ║");
  console.log("╚════════════════════════════════════════════════╝\n");

  const operatorAccount = privateKeyToAccount(`0x${readSecret().replace(/^0x/, "")}`);
  console.log(`Operator: ${operatorAccount.address}`);
  console.log(`Admin API: ${ADMIN_BASE}`);
  console.log(`Markets to create: ${MARKETS.length}\n`);

  const nowSec = Math.floor(Date.now() / 1000);

  for (let i = 0; i < MARKETS.length; i++) {
    const m = MARKETS[i];
    const market = {
      title: m.title,
      description: m.description,
      categoryKey: m.categoryKey,
      coverImage: m.coverImage,
      sourceUrl: "",
      sourceSlug: m.sourceSlug,
      sourceName: "Worm",
      sourceKind: "manual",
      status: "OPEN",
      collateralAsset: "USDT",
      openAt: nowSec - 3600,
      closeAt: nowSec + 90 * 24 * 3600,
      resolveAt: nowSec + 91 * 24 * 3600,
      options: binaryOptions,
    };

    const requestedAt = Date.now();
    const msg = buildCreateMarketMsg({ walletAddress: operatorAccount.address, market, requestedAt });
    const operator = await signOp(operatorAccount, msg, requestedAt);

    let created = false;
    for (let attempt = 1; attempt <= 5; attempt++) {
      const r = await fetchJson(`${ADMIN_BASE}/api/operator/markets`, { method: "POST", body: { market, operator } });
      if (r.ok) {
        console.log(`  [${i + 1}/${MARKETS.length}] ✅ ${m.title}  →  market_id=${r.data?.market_id}`);
        created = true;
        break;
      }
      if (r.status === 429) {
        const wait = 3000 * attempt;
        console.log(`  [${i + 1}/${MARKETS.length}] ⏳ rate limited, waiting ${wait / 1000}s (attempt ${attempt}/5)...`);
        await sleep(wait);
        continue;
      }
      console.error(`  [${i + 1}/${MARKETS.length}] ❌ ${m.title}  →  HTTP ${r.status} ${r.text}`);
      break;
    }
    if (!created) console.error(`  [${i + 1}/${MARKETS.length}] ⚠️  skipped: ${m.title}`);

    if (i < MARKETS.length - 1) await sleep(2000);
  }

  console.log("\n══════════════════════════════════════════════════");
  console.log("  Seed complete.");
  console.log("══════════════════════════════════════════════════\n");
}

main().catch(e => { console.error(`\n❌ ${e.message}\n`); if (e.stack) console.error(e.stack); process.exit(1); });
