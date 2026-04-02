import type { Route } from "next";
import Link from "next/link";

import { formatToken } from "@/lib/format";
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
  const volume = Number(metadata.volume ?? metadata.matchedNotional ?? market.runtime.matched_notional ?? 0);
  const category = String(metadata.category ?? "tape");
  const coverImage = String(metadata.coverImage ?? metadata.coverImageUrl ?? metadata.cover_image_url ?? "");
  const gradient = pickCardGradient(`${market.market_id}:${category}`);
  const href = `/markets/${market.market_id}` as Route;

  return (
    <Link href={href} className={`panel ${styles.card}`} style={{ ["--card-accent" as string]: gradient }}>
      <div className={styles.visual} style={coverImage ? { backgroundImage: `linear-gradient(180deg, rgba(9, 9, 11, 0.08), rgba(9, 9, 11, 0.76)), url(${encodeURI(coverImage)})` } : undefined}>
        <div className={styles.meta}>
          <span>{category}</span>
          <span>{market.status}</span>
        </div>
        <div className={styles.sheen} />
      </div>

      <div>
        <h3 className={styles.title}>{market.title}</h3>
        <p className={styles.description}>{market.description}</p>
      </div>

      <div className={styles.odds}>
        <div className={styles.oddsCard}>
          <span className={styles.oddsLabel}>Yes</span>
          <strong className={styles.oddsValue}>{Math.round(yesOdds * 100)}¢</strong>
        </div>
        <div className={styles.oddsCard}>
          <span className={styles.oddsLabel}>No</span>
          <strong className={styles.oddsValue}>{Math.round(noOdds * 100)}¢</strong>
        </div>
      </div>

      <div className={styles.footer}>
        <span className="pill">{market.collateral_asset}</span>
        <span className={styles.volume}>Vol {formatToken(volume, 0)}</span>
      </div>
    </Link>
  );
}
