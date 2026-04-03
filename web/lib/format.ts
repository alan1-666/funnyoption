export function formatToken(amount: number, digits = 2) {
  return new Intl.NumberFormat("zh-CN", {
    minimumFractionDigits: 0,
    maximumFractionDigits: digits
  }).format(amount);
}

const COLLATERAL_SYMBOL = (process.env.NEXT_PUBLIC_COLLATERAL_SYMBOL ?? "USDT").toUpperCase();
const COLLATERAL_ACCOUNTING_DECIMALS = Number(process.env.NEXT_PUBLIC_COLLATERAL_ACCOUNTING_DECIMALS ?? "2");
const UI_TIME_ZONE = process.env.NEXT_PUBLIC_DISPLAY_TIME_ZONE ?? "Asia/Shanghai";

export function getAssetDecimals(asset?: string) {
  if (!asset) return 0;
  if (asset.toUpperCase() === COLLATERAL_SYMBOL) {
    return COLLATERAL_ACCOUNTING_DECIMALS;
  }
  return 0;
}

export function toDisplayAssetAmount(amount: number, asset?: string) {
  const decimals = getAssetDecimals(asset);
  if (decimals <= 0) {
    return amount;
  }
  return amount / 10 ** decimals;
}

export function formatAssetAmount(amount: number, asset?: string, digits?: number) {
  const decimals = getAssetDecimals(asset);
  const maxDigits = digits ?? (decimals > 0 ? decimals : 0);
  return formatToken(toDisplayAssetAmount(amount, asset), maxDigits);
}

export function formatTimestamp(value: number) {
  if (!value) return "—";
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
    timeZone: UI_TIME_ZONE
  }).format(new Date(value * 1000));
}

export function formatHeadlineTimestamp(value: number) {
  if (!value) return "待定";
  return new Intl.DateTimeFormat("zh-CN", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: UI_TIME_ZONE
  }).format(new Date(value * 1000));
}

export function formatClockTimestamp(valueMillis: number) {
  if (!valueMillis) return "—";
  return new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
    timeZone: UI_TIME_ZONE
  }).format(new Date(valueMillis));
}

export function fromBasisPrice(value: number) {
  return value / 100;
}

export function shortenAddress(value: string) {
  if (!value) return "—";
  return `${value.slice(0, 6)}…${value.slice(-4)}`;
}
