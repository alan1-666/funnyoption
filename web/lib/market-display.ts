import type { Market } from "@/lib/types";

const INTERNAL_CATEGORY_LABELS = new Set(["LOCAL QA", "MANUAL", "OPERATIONS", "DEDICATED ADMIN"]);

const INTERNAL_TITLE_REWRITES: Array<[RegExp, string]> = [
  [/^local lifecycle proof$/i, "本地验证市场"],
  [/^admin hardened create$/i, "后台创建验证市场"],
  [/^admin service proof$/i, "后台服务验证市场"],
  [/^codex bootstrap proof$/i, "流动性验证市场"],
  [/^resolved finality regression/i, "结算终态验证市场"]
];

function readSourceKind(market: Market) {
  const metadata = market.metadata ?? {};
  const raw = metadata.sourceKind ?? metadata.source_kind;
  return typeof raw === "string" ? raw.trim().toLowerCase() : "";
}

export function presentMarketCategory(market: Market) {
  if (market.category?.display_name) {
    return market.category.display_name;
  }

  const raw = typeof market.metadata?.category === "string" ? market.metadata.category.trim() : "";
  if (!raw) {
    return "市场";
  }
  if (INTERNAL_CATEGORY_LABELS.has(raw.toUpperCase())) {
    return "加密";
  }
  return raw;
}

export function presentMarketTitle(market: Market) {
  const rawTitle = market.title.trim();
  if (!rawTitle) {
    return "未命名市场";
  }

  const stripped = rawTitle.replace(/\s+\d{6,}$/, "").trim();
  for (const [pattern, label] of INTERNAL_TITLE_REWRITES) {
    if (pattern.test(stripped)) {
      return label;
    }
  }
  return stripped;
}

export function presentMarketDescription(market: Market) {
  const description = market.description.trim();
  if (!description) {
    return "这是一个本地预测市场，可在这里查看价格、成交与结算进度。";
  }

  const sourceKind = readSourceKind(market);
  if (sourceKind === "local-lifecycle") {
    return "这是一个本地验证市场，用于检查下单、撮合、结算和赔付链路是否正常。";
  }

  if (/cmd\/local-lifecycle|off-chain path|bootstrap proof|admin service proof/i.test(description)) {
    return "这是一个本地验证市场，用于检查交易链路和终态数据是否正常。";
  }

  return description;
}
