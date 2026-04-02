"use client";

import type { Route } from "next";
import Link from "next/link";

import { formatAssetAmount, formatTimestamp } from "@/lib/format";
import { presentMarketCategory, presentMarketTitle } from "@/lib/market-display";
import type { Market } from "@/lib/types";
import styles from "@/components/home-market-card.module.css";

type HomeMarketCardVariant = "square" | "portrait";

function readImageUrl(market: Market) {
  const metadata = market.metadata ?? {};
  const value = metadata.coverImage ?? metadata.coverImageUrl ?? metadata.cover_image_url;
  return typeof value === "string" ? value.trim() : "";
}

function readVideoUrl(market: Market) {
  const metadata = market.metadata ?? {};
  const value =
    metadata.homeVideoUrl ??
    metadata.home_video_url ??
    metadata.cardVideoUrl ??
    metadata.card_video_url ??
    metadata.videoUrl ??
    metadata.video_url;
  return typeof value === "string" ? value.trim() : "";
}

function readBinaryPrice(market: Market, side: "yes" | "no") {
  const metadata = market.metadata ?? {};
  const runtimeValue = side === "yes" ? market.runtime.last_price_yes : market.runtime.last_price_no;
  if (runtimeValue > 0) {
    return runtimeValue;
  }

  const oddsValue = side === "yes" ? metadata.yesOdds : metadata.noOdds;
  if (typeof oddsValue === "number" && Number.isFinite(oddsValue)) {
    return Math.round(oddsValue * 100);
  }

  return side === "yes" ? 50 : 50;
}

function readOptions(market: Market) {
  return (market.options ?? [])
    .filter((option) => option.is_active !== false)
    .sort((left, right) => left.sort_order - right.sort_order || left.key.localeCompare(right.key));
}

function buildCardTone(market: Market) {
  const category = presentMarketCategory(market);
  const title = presentMarketTitle(market);
  if (category.includes("体育")) {
    return "sports";
  }
  return "crypto";
}

function isEndingSoon(market: Market) {
  if (market.status !== "OPEN" || !market.close_at) {
    return false;
  }
  const closeAt = market.close_at * 1000;
  return closeAt - Date.now() <= 1000 * 60 * 60 * 18;
}

export function HomeMarketCard({
  market,
  variant
}: {
  market: Market;
  variant: HomeMarketCardVariant;
}) {
  const href = `/markets/${market.market_id}` as Route;
  const imageUrl = readImageUrl(market);
  const videoUrl = readVideoUrl(market);
  const category = presentMarketCategory(market);
  const title = presentMarketTitle(market);
  const tone = buildCardTone(market);
  const volume = market.runtime.matched_notional;
  const options = readOptions(market);
  const isBinary = options.length <= 2;
  const endingSoon = isEndingSoon(market);
  const yesPrice = readBinaryPrice(market, "yes");
  const noPrice = readBinaryPrice(market, "no");
  const closeLabel = market.status === "OPEN" ? "截止" : "更新";
  const closeValue = market.status === "OPEN" ? formatTimestamp(market.close_at) : formatTimestamp(market.updated_at);
  const primaryOption = options[0]?.short_label ?? options[0]?.label ?? "YES";
  const secondaryOption = options[1]?.short_label ?? options[1]?.label ?? "NO";
  const tertiaryOptions = options.slice(0, 3);

  return (
    <Link
      href={href}
      className={`${styles.card} ${variant === "portrait" ? styles.portrait : styles.square}`}
      data-tone={tone}
    >
      <div className={styles.media}>
        {videoUrl ? (
          <video
            className={styles.mediaVideo}
            src={videoUrl}
            muted
            playsInline
            autoPlay
            loop
            preload="metadata"
          />
        ) : imageUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img className={styles.mediaImage} src={imageUrl} alt="" loading="lazy" />
        ) : (
          <div className={styles.mediaFallback} aria-hidden="true" />
        )}
        <div className={styles.mediaShade} />
      </div>

      <div className={styles.topRow}>
        <span className={styles.metaTag}>{category}</span>
        {endingSoon ? <span className={styles.alertTag}>即将结束</span> : null}
      </div>

      <div className={styles.content}>
        <div className={styles.headingBlock}>
          <span className={styles.timeMeta}>
            {closeLabel} · {closeValue}
          </span>
          <h3 className={styles.title}>{title}</h3>
        </div>

        {isBinary ? (
          <div className={styles.binaryBoard}>
            <div className={styles.priceRow}>
              <span>{primaryOption}</span>
              <strong>{yesPrice}¢</strong>
            </div>
            <div className={styles.priceTrack}>
              <span className={styles.priceFill} style={{ width: `${Math.max(8, Math.min(92, yesPrice))}%` }} />
            </div>
            <div className={styles.priceRow}>
              <span>{secondaryOption}</span>
              <strong>{noPrice}¢</strong>
            </div>
          </div>
        ) : (
          <div className={styles.multiBoard}>
            {tertiaryOptions.map((option, index) => (
              <div key={`${market.market_id}-${option.key}`} className={styles.optionRow}>
                <span>{option.label}</span>
                <strong>{index === 0 ? "热门" : `${index + 1} 号`}</strong>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className={styles.footer}>
        <span>{formatAssetAmount(volume, market.collateral_asset)} {market.collateral_asset}</span>
        <span>{market.runtime.trade_count} 笔成交</span>
      </div>
    </Link>
  );
}

export function resolveHomeMarketCardVariant(market: Market, index: number): HomeMarketCardVariant {
  const metadata = market.metadata ?? {};
  const raw =
    typeof metadata.homeCardStyle === "string"
      ? metadata.homeCardStyle
      : typeof metadata.cardStyle === "string"
        ? metadata.cardStyle
        : typeof metadata.card_style === "string"
          ? metadata.card_style
          : "";
  const normalized = raw.trim().toLowerCase();

  if (normalized === "portrait" || normalized === "tall" || normalized === "video") {
    return "portrait";
  }

  if (normalized === "square" || normalized === "standard") {
    return "square";
  }

  if (readVideoUrl(market)) {
    return "portrait";
  }

  return index % 7 === 0 ? "portrait" : "square";
}
