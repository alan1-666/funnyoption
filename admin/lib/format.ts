export function formatToken(amount: number, digits = 2) {
  return new Intl.NumberFormat("zh-CN", {
    minimumFractionDigits: 0,
    maximumFractionDigits: digits
  }).format(amount);
}

const COLLATERAL_SYMBOL = (process.env.NEXT_PUBLIC_COLLATERAL_SYMBOL ?? "USDT").toUpperCase();
const COLLATERAL_ACCOUNTING_DECIMALS = Number(process.env.NEXT_PUBLIC_COLLATERAL_ACCOUNTING_DECIMALS ?? "2");

export function formatAssetAmount(amount: number, asset?: string, digits?: number) {
  if (!asset || asset.toUpperCase() !== COLLATERAL_SYMBOL) {
    return formatToken(amount, digits ?? 0);
  }
  return formatToken(amount / 10 ** COLLATERAL_ACCOUNTING_DECIMALS, digits ?? COLLATERAL_ACCOUNTING_DECIMALS);
}

export function formatTimestamp(value: number) {
  if (!value) return "—";
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false
  }).format(new Date(value * 1000));
}

export function shortenAddress(value: string) {
  if (!value) return "—";
  return `${value.slice(0, 6)}…${value.slice(-4)}`;
}
