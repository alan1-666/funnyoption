import type { Route } from "next";
import Link from "next/link";

import { MarketCard } from "@/components/market-card";
import { SiteHeader } from "@/components/site-header";
import { formatTimestamp, formatToken } from "@/lib/format";
import { getMarketsRead, getTradesRead } from "@/lib/api";
import styles from "@/app/page.module.css";

export default async function HomePage() {
  const [marketsResult, tradesResult] = await Promise.all([getMarketsRead(), getTradesRead()]);
  const markets = marketsResult.items;
  const trades = tradesResult.items;
  const marketsUnavailable = marketsResult.state === "unavailable";
  const tradesUnavailable = tradesResult.state === "unavailable";
  const featured = marketsUnavailable ? null : markets.find((market) => market.status === "OPEN") ?? markets[0];
  const featuredHref = featured ? (`/markets/${featured.market_id}` as Route) : "/portfolio";
  const totalMatchedNotional = markets.reduce((sum, market) => sum + market.runtime.matched_notional, 0);
  const totalOpen = markets.filter((market) => market.status === "OPEN").length;
  const totalResolved = markets.filter((market) => market.status === "RESOLVED").length;
  const degraded = marketsUnavailable || tradesUnavailable;

  return (
    <main className="page-shell">
      <SiteHeader />

      <section className={styles.hero}>
        <div className={`${styles.heroMain} float-in`}>
          <span className="eyebrow">FunnyOption / BSC testnet</span>
          <h1 className={`${styles.heroTitle} section-title`}>
            Build a faster
            <span className="gradient-text"> prediction tape</span>
            without dragging every click on-chain.
          </h1>
          <p className={`${styles.heroCopy} section-copy`}>
            {degraded
              ? "Market data is temporarily unavailable. Reload in a moment for the latest prices and activity."
              : "Browse live markets, recent trades, and payout progress from the current FunnyOption market state."}
          </p>
          <div className={styles.heroActions}>
            <Link href={featuredHref} className={styles.primaryAction}>
              {featured ? "Open Lead Market" : "Open Portfolio"}
            </Link>
            <Link href="/portfolio" className={styles.secondaryAction}>
              View Portfolio
            </Link>
          </div>
        </div>

        <aside className={`${styles.heroRail} float-in float-in-delay-1`}>
          <div className={`panel ${styles.metricPanel}`}>
            <span className={styles.metricLabel}>Open markets</span>
            <strong className="metric-value">{marketsUnavailable ? "—" : totalOpen}</strong>
            <p className={styles.metricCopy}>
              {marketsUnavailable
                ? `Market list unavailable: ${marketsResult.error?.message ?? "please try again shortly"}.`
                : "The lead action always points to an actively tradable market when one is available."}
            </p>
          </div>

          <div className={`panel ${styles.metricPanel}`}>
            <span className={styles.metricLabel}>Resolved markets</span>
            <strong className="metric-value">{marketsUnavailable ? "—" : totalResolved}</strong>
            <p className={styles.metricCopy}>
              {marketsUnavailable
                ? "Resolved market data is unavailable right now."
                : "Resolved markets remain visible so payout history stays easy to review."}
            </p>
          </div>

          <div className={`panel ${styles.metricPanelAlt}`}>
            <span className={styles.metricLabel}>Matched flow</span>
            <strong className={styles.flowValue}>{marketsUnavailable ? "API unavailable" : `${formatToken(totalMatchedNotional / 100, 0)} USDT`}</strong>
            <div className={styles.signalGrid}>
              {tradesUnavailable ? (
                <div className={styles.signalItem}>
                  <span>tape</span>
                  <strong>offline</strong>
                </div>
              ) : trades.length > 0 ? (
                trades.slice(0, 3).map((trade) => (
                  <div key={trade.trade_id} className={styles.signalItem}>
                    <span>{trade.outcome}</span>
                    <strong>{trade.price}c</strong>
                  </div>
                ))
              ) : (
                <div className={styles.signalItem}>
                  <span>tape</span>
                  <strong>quiet</strong>
                </div>
              )}
            </div>
          </div>
        </aside>
      </section>

      <section className={`${styles.featured} float-in float-in-delay-2`}>
        <div className={`panel ${styles.featuredPanel}`}>
          <div className={styles.featuredHeader}>
            <div>
              <span className="eyebrow">Lead market</span>
              <h2 className={styles.featuredTitle}>{marketsUnavailable ? "Market board unavailable" : featured?.title ?? "No live tape yet"}</h2>
            </div>
            {featured ? <span className="pill">{featured.status}</span> : null}
          </div>
          <div className={styles.featuredGrid}>
            <p className={styles.featuredCopy}>
              {marketsUnavailable
                ? `The lead market could not be loaded right now. ${marketsResult.error?.message ?? "Please reload shortly."}`
                : featured?.description ?? "No featured market is available yet."}
            </p>
            <div className={styles.featuredMeta}>
              <div>
                <span className={styles.metricLabel}>Last trade</span>
                <strong>{featured ? formatTimestamp(featured.runtime.last_trade_at) : "—"}</strong>
              </div>
              <div>
                <span className={styles.metricLabel}>Matched notional</span>
                <strong>{featured ? `${formatToken(featured.runtime.matched_notional / 100, 0)} USDT` : "—"}</strong>
              </div>
              <div>
                <span className={styles.metricLabel}>Payout status</span>
                <strong>{featured ? `${featured.runtime.completed_payout_count}/${featured.runtime.payout_count} completed` : "—"}</strong>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className={`${styles.tape} float-in float-in-delay-3`}>
        <div className={styles.sectionHeader}>
          <div>
            <span className="eyebrow">Live board</span>
            <h2 className="section-title">Markets</h2>
          </div>
          <p className="section-copy">Browse current markets with live prices, trading volume, and cover imagery when available.</p>
        </div>
        <div className={styles.marketGrid}>
          {marketsUnavailable ? (
            <div className={`panel ${styles.tradePanel}`}>
              <div className={styles.tradeHeadline}>Market board unavailable.</div>
              <div className={styles.tradeMeta}>
                {marketsResult.error?.message ?? "The homepage could not load markets right now."}
              </div>
            </div>
          ) : markets.length > 0 ? (
            markets.map((market) => (
              <MarketCard key={market.market_id} market={market} />
            ))
          ) : (
            <div className={`panel ${styles.tradePanel}`}>
              <div className={styles.tradeHeadline}>No markets in the local database.</div>
              <div className={styles.tradeMeta}>No markets are available yet.</div>
            </div>
          )}
        </div>
      </section>

      <section className={styles.bottomRow}>
        <div className={`panel ${styles.tradePanel}`}>
          <div className={styles.sectionHeader}>
            <div>
              <span className="eyebrow">Pulse</span>
              <h2 className={styles.subTitle}>Trade tape</h2>
            </div>
          </div>
          <div className={styles.tradeList}>
            {tradesUnavailable ? (
              <div className={styles.tradeMeta}>
                Trade activity is temporarily unavailable. {tradesResult.error?.message ?? "Please reload shortly."}
              </div>
            ) : trades.length > 0 ? (
              trades.map((trade) => (
                <div key={trade.trade_id} className={styles.tradeRow}>
                  <div>
                    <div className={styles.tradeHeadline}>#{trade.market_id} {trade.outcome}</div>
                    <div className={styles.tradeMeta}>{formatTimestamp(trade.occurred_at)} · seq {trade.sequence_no}</div>
                  </div>
                  <div className={styles.tradeAmounts}>
                    <strong>{trade.price}c</strong>
                    <span>{formatToken(trade.quantity, 0)}</span>
                  </div>
                </div>
              ))
            ) : (
              <div className={styles.tradeMeta}>No trades have matched in the current local DB yet.</div>
            )}
          </div>
        </div>

        <div className={`panel ${styles.archPanel}`}>
          <div className={styles.sectionHeader}>
            <div>
              <span className="eyebrow">Why People Use It</span>
              <h2 className={styles.subTitle}>Built for active traders</h2>
            </div>
          </div>
          <div className={styles.archList}>
            <div className={styles.archItem}>
              <strong>01</strong>
              <div>
                <h3>Quick market access</h3>
                <p>Connect once, enable trading, and keep placing orders without repeated wallet prompts on every click.</p>
              </div>
            </div>
            <div className={styles.archItem}>
              <strong>02</strong>
              <div>
                <h3>Direct vault funding</h3>
                <p>Deposit from your wallet into the vault, then trade against your credited balance from the same portfolio view.</p>
              </div>
            </div>
            <div className={styles.archItem}>
              <strong>03</strong>
              <div>
                <h3>Clear settlement</h3>
                <p>Resolved markets stay visible with payout progress, so it is easy to see what is ready to claim.</p>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}
