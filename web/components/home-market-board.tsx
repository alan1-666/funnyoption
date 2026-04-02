"use client";

import type { Route } from "next";
import Link from "next/link";
import { useDeferredValue, useState } from "react";

import { HomeMarketCard, resolveHomeMarketCardVariant } from "@/components/home-market-card";
import { ShellTopBar } from "@/components/shell-top-bar";
import { formatTimestamp } from "@/lib/format";
import { presentMarketCategory, presentMarketTitle } from "@/lib/market-display";
import type { Market, Trade } from "@/lib/types";
import styles from "@/components/home-market-board.module.css";

type FilterKey = "all" | "hot" | "ending" | "crypto" | "sports";

const FILTERS: Array<{ key: FilterKey; label: string }> = [
  { key: "all", label: "全部" },
  { key: "hot", label: "热门" },
  { key: "ending", label: "即将结束" },
  { key: "crypto", label: "加密" },
  { key: "sports", label: "体育" }
];

function readSearchText(market: Market) {
  const category = presentMarketCategory(market);
  const title = presentMarketTitle(market);
  const description = market.description ?? "";
  const options = (market.options ?? []).map((option) => `${option.label} ${option.short_label ?? ""}`).join(" ");
  return `${title} ${category} ${description} ${options}`.toLowerCase();
}

function readBannerCopy(market: Market) {
  const category = presentMarketCategory(market);
  return `${category} · ${market.status === "OPEN" ? "交易进行中" : "市场跟踪中"} · 最近更新 ${formatTimestamp(market.updated_at)}`;
}

function isEndingSoon(market: Market) {
  if (market.status !== "OPEN" || !market.close_at) {
    return false;
  }
  return market.close_at * 1000 - Date.now() <= 1000 * 60 * 60 * 18;
}

export function HomeMarketBoard({
  markets,
  trades,
  chainName,
  marketsUnavailable,
  tradesUnavailable,
  marketsError,
  tradesError
}: {
  markets: Market[];
  trades: Trade[];
  chainName: string;
  marketsUnavailable: boolean;
  tradesUnavailable: boolean;
  marketsError?: string;
  tradesError?: string;
}) {
  const [query, setQuery] = useState("");
  const [activeFilter, setActiveFilter] = useState<FilterKey>("all");
  const deferredQuery = useDeferredValue(query.trim().toLowerCase());
  const deferredFilter = useDeferredValue(activeFilter);

  const sortedMarkets = [...markets].sort((left, right) => {
    if (left.status === right.status) {
      return right.updated_at - left.updated_at;
    }
    return left.status === "OPEN" ? -1 : 1;
  });

  const filteredMarkets = sortedMarkets.filter((market) => {
    if (deferredFilter === "hot" && market.runtime.trade_count <= 0 && market.runtime.matched_notional <= 0) {
      return false;
    }
    if (deferredFilter === "ending" && !isEndingSoon(market)) {
      return false;
    }
    if (deferredFilter === "crypto" && !/crypto|加密/i.test(presentMarketCategory(market))) {
      return false;
    }
    if (deferredFilter === "sports" && !/sports|体育/i.test(presentMarketCategory(market))) {
      return false;
    }
    if (deferredQuery && !readSearchText(market).includes(deferredQuery)) {
      return false;
    }
    return true;
  });

  const focusMarket = filteredMarkets[0] ?? sortedMarkets[0] ?? null;
  const latestTrades = trades.slice(0, 6);
  const focusTitle = focusMarket ? presentMarketTitle(focusMarket) : "";

  return (
    <section className={`${styles.shell} float-in`}>
      <ShellTopBar query={query} onQueryChange={setQuery} />

      <div className={styles.filterRow}>
        <div className={styles.filterRail}>
          {FILTERS.map((filter) => (
            <button
              key={filter.key}
              className={`${styles.filterChip} ${activeFilter === filter.key ? styles.filterChipActive : ""}`}
              onClick={() => setActiveFilter(filter.key)}
            >
              {filter.label}
            </button>
          ))}
        </div>
      </div>

      {focusMarket ? (
        <Link href={`/markets/${focusMarket.market_id}` as Route} className={styles.banner}>
          <div className={styles.bannerBackdrop} />
          <div className={styles.bannerCopy}>
            <span className={styles.bannerEyebrow}>焦点市场</span>
            <strong className={styles.bannerTitle}>{focusTitle}</strong>
            <span className={styles.bannerMeta}>{readBannerCopy(focusMarket)}</span>
          </div>
          <div className={styles.bannerStats}>
            <div className={styles.bannerMetric}>
              <span>交易中</span>
              <strong>{markets.filter((market) => market.status === "OPEN").length}</strong>
            </div>
            <div className={styles.bannerMetric}>
              <span>最近成交</span>
              <strong>{tradesUnavailable ? "—" : latestTrades.length}</strong>
            </div>
            <div className={styles.bannerMetric}>
              <span>网络</span>
              <strong>{chainName}</strong>
            </div>
          </div>
        </Link>
      ) : null}

      {latestTrades.length > 0 && !tradesUnavailable ? (
        <div className={styles.tradeTape}>
          <div className={styles.tradeTrack}>
            {[...latestTrades, ...latestTrades].map((trade, index) => (
              <span key={`${trade.trade_id}-${index}`} className={styles.tradeChip}>
                <strong>{trade.price}¢</strong>
                <span>{trade.quantity} 份</span>
                <span>{formatTimestamp(trade.occurred_at)}</span>
              </span>
            ))}
          </div>
        </div>
      ) : null}

      <div className={styles.board}>
        {marketsUnavailable ? (
          <div className={styles.emptyState}>市场读取暂时不可用。{marketsError ?? "请稍后刷新。"}</div>
        ) : filteredMarkets.length > 0 ? (
          filteredMarkets.map((market, index) => (
            <HomeMarketCard
              key={market.market_id}
              market={market}
              variant={resolveHomeMarketCardVariant(market, index)}
            />
          ))
        ) : (
          <div className={styles.emptyState}>
            {deferredQuery || deferredFilter !== "all"
              ? "没有匹配的市场，换个关键词或分类试试。"
              : "当前还没有市场，先去后台创建一条。"}
          </div>
        )}
      </div>

      {tradesUnavailable ? (
        <div className={styles.footerNote}>成交流暂时不可用。{tradesError ?? "请稍后刷新。"}</div>
      ) : null}
    </section>
  );
}
