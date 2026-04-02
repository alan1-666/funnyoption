import type { Route } from "next";
import Link from "next/link";

import { formatTimestamp, formatToken } from "@/lib/format";
import { presentMarketCategory } from "@/lib/market-display";
import { zhMarketStatus, zhOutcome } from "@/lib/locale";
import type { Market } from "@/lib/types";
import styles from "@/components/market-card.module.css";

const CARD_ACCENTS = [
  "#80f7ff",
  "#b9ff42",
  "#ffcb6b",
  "#ff8080"
];

function pickCardGradient(seed: string) {
  let hash = 0;
  for (let index = 0; index < seed.length; index += 1) {
    hash = (hash * 31 + seed.charCodeAt(index)) >>> 0;
  }
  return CARD_ACCENTS[hash % CARD_ACCENTS.length];
}

export function MarketCard({ market }: { market: Market }) {
  const metadata = market.metadata ?? {};
  const yesOdds = Number(metadata.yesOdds ?? (market.runtime.last_price_yes ? market.runtime.last_price_yes / 100 : 0.5));
  const noOdds = Number(metadata.noOdds ?? (market.runtime.last_price_no ? market.runtime.last_price_no / 100 : 0.5));
  const yesPercent = Math.max(4, Math.min(96, Math.round(yesOdds * 100)));
  const noPercent = Math.max(4, 100 - yesPercent);
  const volume = Number(metadata.volume ?? metadata.matchedNotional ?? market.runtime.matched_notional ?? 0);
  const category = presentMarketCategory(market);
  const optionSummary = (market.options ?? [])
    .filter((option) => option.is_active !== false)
    .sort((left, right) => left.sort_order - right.sort_order || left.key.localeCompare(right.key))
    .slice(0, 4)
    .map((option) => option.short_label ?? option.label)
    .join(" / ");
  const coverImage = String(metadata.coverImage ?? metadata.coverImageUrl ?? metadata.cover_image_url ?? "");
  const gradient = pickCardGradient(`${market.market_id}:${category}`);
  const href = `/markets/${market.market_id}` as Route;
  const timeLabel = market.status === "OPEN" ? "停止交易" : "最近更新";
  const timeValue = market.status === "OPEN" ? formatTimestamp(market.close_at) : formatTimestamp(market.updated_at);

  return (
    <Link href={href} className={`panel ${styles.card}`} style={{ ["--card-accent" as string]: gradient }}>
      <div className={styles.visual} style={coverImage ? { backgroundImage: `linear-gradient(180deg, rgba(9, 9, 11, 0.08), rgba(9, 9, 11, 0.76)), url(${encodeURI(coverImage)})` } : undefined}>
        <div className={styles.meta}>
          <span>{category}</span>
          <span>{zhMarketStatus(market.status)}</span>
        </div>
        <div className={styles.sheen} />
      </div>

      <div className={styles.body}>
        <h3 className={styles.title}>{market.title}</h3>
        <div className={styles.summaryRow}>
          <span>{timeLabel} {timeValue}</span>
          {optionSummary ? <span className={styles.summarySub}>选项 {optionSummary}</span> : null}
        </div>
      </div>

      <div className={styles.odds}>
        <div className={styles.oddsCard}>
          <span className={styles.oddsLabel}>{zhOutcome("YES")}</span>
          <strong className={styles.oddsValue}>{Math.round(yesOdds * 100)}¢</strong>
        </div>
        <div className={styles.oddsCard}>
          <span className={styles.oddsLabel}>{zhOutcome("NO")}</span>
          <strong className={styles.oddsValue}>{Math.round(noOdds * 100)}¢</strong>
        </div>
      </div>

      <div className={styles.flowRail} aria-hidden="true">
        <span className={styles.flowYes} style={{ width: `${yesPercent}%` }} />
        <span className={styles.flowNo} style={{ width: `${noPercent}%` }} />
      </div>

      <div className={styles.footer}>
        <span className={styles.asset}>{market.collateral_asset}</span>
        <span className={styles.volume}>成交量 {formatToken(volume, 0)}</span>
      </div>
    </Link>
  );
}
