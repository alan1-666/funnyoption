export const OPERATOR_SIGNATURE_WINDOW_MS = 5 * 60 * 1000;

export interface CreateMarketDraft {
  title: string;
  description: string;
  categoryKey: string;
  coverImage: string;
  sourceUrl: string;
  sourceSlug: string;
  sourceName: string;
  sourceKind: string;
  status: string;
  collateralAsset: string;
  openAt: number;
  closeAt: number;
  resolveAt: number;
  options: MarketOptionDraft[];
}

export interface MarketOptionDraft {
  key: string;
  label: string;
  shortLabel?: string;
  sortOrder: number;
  isActive: boolean;
}

export interface ResolveMarketDraft {
  marketId: number;
  outcome: "YES" | "NO";
}

export interface BootstrapMarketDraft {
  marketId: number;
  userId: number;
  quantity: number;
  outcome: "YES" | "NO";
  price: number;
}

export interface SignedOperatorAction {
  walletAddress: string;
  requestedAt: number;
  signature: string;
}

function cleanText(value: string) {
  return value.trim().replace(/\s+/g, " ");
}

export function normalizeAddress(value: string) {
  return value.trim().toLowerCase();
}

export function parseOperatorWallets(raw: string) {
  return raw
    .split(",")
    .map((item) => normalizeAddress(item))
    .filter(Boolean);
}

export function normalizeCreateMarketDraft(input: CreateMarketDraft): CreateMarketDraft {
  return {
    title: cleanText(input.title),
    description: cleanText(input.description),
    categoryKey: normalizeCategoryKey(input.categoryKey),
    coverImage: input.coverImage.trim(),
    sourceUrl: input.sourceUrl.trim(),
    sourceSlug: cleanText(input.sourceSlug),
    sourceName: cleanText(input.sourceName) || "Polymarket",
    sourceKind: cleanText(input.sourceKind).toLowerCase() || "manual",
    status: cleanText(input.status).toUpperCase() || "OPEN",
    collateralAsset: cleanText(input.collateralAsset).toUpperCase() || "USDT",
    openAt: Math.max(0, Math.floor(input.openAt || 0)),
    closeAt: Math.max(0, Math.floor(input.closeAt || 0)),
    resolveAt: Math.max(0, Math.floor(input.resolveAt || 0)),
    options: normalizeMarketOptions(input.options)
  };
}

export function normalizeResolveMarketDraft(input: ResolveMarketDraft): ResolveMarketDraft {
  return {
    marketId: Math.max(0, Math.floor(input.marketId || 0)),
    outcome: cleanText(input.outcome).toUpperCase() === "NO" ? "NO" : "YES"
  };
}

export function normalizeBootstrapMarketDraft(input: BootstrapMarketDraft): BootstrapMarketDraft {
  return {
    marketId: Math.max(0, Math.floor(input.marketId || 0)),
    userId: Math.max(0, Math.floor(input.userId || 0)),
    quantity: Math.max(0, Math.floor(input.quantity || 0)),
    outcome: cleanText(input.outcome).toUpperCase() === "NO" ? "NO" : "YES",
    price: Math.max(0, Math.floor(input.price || 0))
  };
}

export function buildCreateMarketMessage(input: {
  walletAddress: string;
  market: CreateMarketDraft;
  requestedAt: number;
}) {
  const walletAddress = normalizeAddress(input.walletAddress);
  const market = normalizeCreateMarketDraft(input.market);
  return `FunnyOption Operator Authorization

action: CREATE_MARKET
wallet: ${walletAddress}
title: ${market.title}
description: ${market.description}
category: ${market.categoryKey}
source_kind: ${market.sourceKind}
source_url: ${market.sourceUrl}
source_slug: ${market.sourceSlug}
source_name: ${market.sourceName}
cover_image: ${market.coverImage}
status: ${market.status}
collateral_asset: ${market.collateralAsset}
open_at: ${market.openAt}
close_at: ${market.closeAt}
resolve_at: ${market.resolveAt}
requested_at: ${Math.floor(input.requestedAt)}
options: ${buildMarketOptionSignatureFragment(market.options)}
`;
}

export function normalizeCategoryKey(value: string) {
  const normalized = cleanText(value).toUpperCase();
  if (normalized === "SPORTS" || normalized === "体育") {
    return "SPORTS";
  }
  return "CRYPTO";
}

export function normalizeMarketOptions(input: MarketOptionDraft[]) {
  if (!Array.isArray(input) || input.length === 0) {
    return defaultBinaryOptions();
  }
  return input
    .map((option, index) => ({
      key: cleanText(option.key).toUpperCase().replace(/\s+/g, "_"),
      label: cleanText(option.label),
      shortLabel: cleanText(option.shortLabel ?? option.label),
      sortOrder: Math.max(1, Math.floor(option.sortOrder || (index + 1) * 10)),
      isActive: option.isActive !== false
    }))
    .filter((option) => option.key && option.label)
    .sort((left, right) => left.sortOrder - right.sortOrder || left.key.localeCompare(right.key));
}

export function defaultBinaryOptions(): MarketOptionDraft[] {
  return [
    { key: "YES", label: "是", shortLabel: "是", sortOrder: 10, isActive: true },
    { key: "NO", label: "否", shortLabel: "否", sortOrder: 20, isActive: true }
  ];
}

function buildMarketOptionSignatureFragment(options: MarketOptionDraft[]) {
  return options
    .map((option) => `${option.key}:${option.label}:${option.shortLabel ?? option.label}:${option.sortOrder}:${option.isActive ? "1" : "0"}`)
    .join("|");
}

export function buildBootstrapMarketMessage(input: {
  walletAddress: string;
  bootstrap: BootstrapMarketDraft;
  requestedAt: number;
}) {
  const walletAddress = normalizeAddress(input.walletAddress);
  const bootstrap = normalizeBootstrapMarketDraft(input.bootstrap);
  return `FunnyOption Operator Authorization

action: ISSUE_FIRST_LIQUIDITY
wallet: ${walletAddress}
market_id: ${bootstrap.marketId}
user_id: ${bootstrap.userId}
quantity: ${bootstrap.quantity}
outcome: ${bootstrap.outcome}
price: ${bootstrap.price}
requested_at: ${Math.floor(input.requestedAt)}
`;
}

export function buildResolveMarketMessage(input: {
  walletAddress: string;
  market: ResolveMarketDraft;
  requestedAt: number;
}) {
  const walletAddress = normalizeAddress(input.walletAddress);
  const market = normalizeResolveMarketDraft(input.market);
  return `FunnyOption Operator Authorization

action: RESOLVE_MARKET
wallet: ${walletAddress}
market_id: ${market.marketId}
outcome: ${market.outcome}
requested_at: ${Math.floor(input.requestedAt)}
`;
}
