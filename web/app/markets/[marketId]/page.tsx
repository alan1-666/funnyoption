import Link from "next/link";

import { LiveMarketPanel } from "@/components/live-market-panel";
import { OrderTicket } from "@/components/order-ticket";
import { SiteHeader } from "@/components/site-header";
import { formatTimestamp, formatToken } from "@/lib/format";
import { getMarketRead, getTradesRead } from "@/lib/api";
import styles from "@/app/markets/[marketId]/page.module.css";

export default async function MarketDetailPage({ params }: { params: Promise<{ marketId: string }> }) {
  const { marketId } = await params;
  const numericMarketId = Number(marketId);
  if (!Number.isInteger(numericMarketId) || numericMarketId <= 0) {
    return (
      <main className="page-shell">
        <SiteHeader />
        <div className={styles.backLinkWrap}>
          <Link href="/" className={styles.backLink}>← Back to tape</Link>
        </div>
        <section className={`panel ${styles.hero}`}>
          <span className="eyebrow">Market not found</span>
          <h1 className={styles.title}>Market #{marketId} is not a valid local market id.</h1>
          <p className={styles.copy}>Return to the homepage and open a market from the live list.</p>
        </section>
      </main>
    );
  }

  const [marketResult, tradesResult] = await Promise.all([getMarketRead(numericMarketId), getTradesRead(numericMarketId)]);
  const trades = tradesResult.items;

  if (marketResult.state === "unavailable") {
    return (
      <main className="page-shell">
        <SiteHeader />
        <div className={styles.backLinkWrap}>
          <Link href="/" className={styles.backLink}>← Back to tape</Link>
        </div>
        <section className={`panel ${styles.hero}`}>
          <span className="eyebrow">API unavailable</span>
          <h1 className={styles.title}>This market detail could not be loaded.</h1>
          <p className={styles.copy}>
            {marketResult.error?.message ?? "The market data could not be loaded right now."} Please reload in a moment.
          </p>
        </section>
      </main>
    );
  }

  if (marketResult.state === "not-found" || !marketResult.item) {
    return (
      <main className="page-shell">
        <SiteHeader />
        <div className={styles.backLinkWrap}>
          <Link href="/" className={styles.backLink}>← Back to tape</Link>
        </div>
        <section className={`panel ${styles.hero}`}>
          <span className="eyebrow">Market not found</span>
          <h1 className={styles.title}>Market #{marketId} does not exist.</h1>
          <p className={styles.copy}>Return to the homepage and choose another available market.</p>
        </section>
      </main>
    );
  }

  const market = marketResult.item;
  const tradesUnavailable = tradesResult.state === "unavailable";

  const metadata = market.metadata ?? {};
  const yesOdds = Math.round(Number(metadata.yesOdds ?? (market.runtime.last_price_yes ? market.runtime.last_price_yes / 100 : 0.5)) * 100);
  const noOdds = Math.round(Number(metadata.noOdds ?? (market.runtime.last_price_no ? market.runtime.last_price_no / 100 : 0.5)) * 100);
  const matchedNotional = market.runtime.matched_notional;
  const category = String(metadata.category ?? "local");
  const coverImageUrl = String(metadata.coverImage ?? metadata.coverImageUrl ?? metadata.cover_image_url ?? "");
  const coverSourceName = String(metadata.sourceName ?? metadata.coverSourceName ?? metadata.cover_source_name ?? "");

  return (
    <main className="page-shell">
      <SiteHeader />

      <div className={styles.backLinkWrap}>
        <Link href="/" className={styles.backLink}>← Back to tape</Link>
      </div>

      <section className={styles.layout}>
        <div className={styles.main}>
          <div className={`${styles.hero} panel float-in`}>
            <div className={styles.heroTop}>
              <div>
                <span className="eyebrow">{category}</span>
                <h1 className={styles.title}>{market.title}</h1>
              </div>
              <div className={styles.heroPills}>
                <span className="pill">{market.status}</span>
                <span className="pill">{market.collateral_asset}</span>
              </div>
            </div>

            {coverImageUrl ? (
              <div className={styles.heroMedia}>
                <img className={styles.heroImage} src={coverImageUrl} alt={market.title} loading="lazy" />
                <div className={styles.heroImageScrim} />
                <div className={styles.heroImageMeta}>
                  <span>{coverSourceName || "Market cover"}</span>
                  <strong>{market.status}</strong>
                </div>
              </div>
            ) : null}

            <p className={styles.copy}>{market.description}</p>

            <div className={styles.statsGrid}>
              <div className={styles.statCard}>
                <span className={styles.label}>Yes</span>
                <strong>{yesOdds}¢</strong>
              </div>
              <div className={styles.statCard}>
                <span className={styles.label}>No</span>
                <strong>{noOdds}¢</strong>
              </div>
              <div className={styles.statCard}>
                <span className={styles.label}>Matched notional</span>
                <strong>{formatToken(matchedNotional / 100, 0)} USDT</strong>
              </div>
              <div className={styles.statCard}>
                <span className={styles.label}>Last trade</span>
                <strong>{formatTimestamp(market.runtime.last_trade_at)}</strong>
              </div>
            </div>
          </div>

          <div className={`${styles.grid} float-in float-in-delay-1`}>
            {tradesUnavailable ? (
              <section className={`panel ${styles.block}`}>
                <div className={styles.blockHeader}>
                  <div>
                    <span className="eyebrow">Live feeds</span>
                    <h2 className={styles.blockTitle}>Trade snapshot unavailable</h2>
                  </div>
                </div>
                <div className={styles.routeList}>
                  <div className={styles.routeItem}>
                    <strong>01</strong>
                    <p>{tradesResult.error?.message ?? "The detail page could not load recent trading activity."}</p>
                  </div>
                  <div className={styles.routeItem}>
                    <strong>02</strong>
                    <p>This is different from a quiet market. Reload after the data feed recovers before treating activity as empty.</p>
                  </div>
                </div>
              </section>
            ) : (
              <LiveMarketPanel market={market} trades={trades} />
            )}
            <section className={`panel ${styles.block}`}>
              <div className={styles.blockHeader}>
                <div>
                  <span className="eyebrow">What to expect</span>
                  <h2 className={styles.blockTitle}>Trading and settlement</h2>
                </div>
              </div>
              <div className={styles.routeList}>
                <div className={styles.routeItem}>
                  <strong>01</strong>
                  <p>Connect your wallet first, then enable trading once so follow-up orders stay smoother.</p>
                </div>
                <div className={styles.routeItem}>
                  <strong>02</strong>
                  <p>Orders reserve the required balance before they are submitted, so available funds stay accurate.</p>
                </div>
                <div className={styles.routeItem}>
                  <strong>03</strong>
                  <p>When the market resolves, any winning payout will appear in your portfolio and can be claimed from there.</p>
                </div>
                <div className={styles.routeItem}>
                  <strong>04</strong>
                  <p>
                    Activity summary: {market.runtime.trade_count} trades, {market.runtime.active_order_count} open orders,
                    {market.runtime.completed_payout_count}/{market.runtime.payout_count} payouts completed.
                  </p>
                </div>
              </div>
            </section>

            <section className={`panel ${styles.block}`}>
              <div className={styles.blockHeader}>
                <div>
                  <span className="eyebrow">Recent prints</span>
                  <h2 className={styles.blockTitle}>Trade flow</h2>
                </div>
              </div>
              <div className={styles.tradeTable}>
                {tradesUnavailable ? (
                  <div className={styles.tradeSub}>
                    Trade flow is temporarily unavailable. {tradesResult.error?.message ?? "Please reload before treating this market as quiet."}
                  </div>
                ) : trades.length > 0 ? (
                  trades.map((trade) => (
                    <div key={trade.trade_id} className={styles.tradeRow}>
                      <div>
                        <div className={styles.tradeHead}>{trade.outcome} · {trade.taker_side}</div>
                        <div className={styles.tradeSub}>seq {trade.sequence_no} · {formatTimestamp(trade.occurred_at)}</div>
                      </div>
                      <div className={styles.tradeNums}>
                        <strong>{trade.price}¢</strong>
                        <span>{formatToken(trade.quantity, 0)}</span>
                      </div>
                    </div>
                  ))
                ) : (
                  <div className={styles.tradeSub}>No matched prints have been recorded for this market in the current local DB.</div>
                )}
              </div>
            </section>
          </div>
        </div>

        <aside className={`${styles.sidebar} float-in float-in-delay-2`}>
          <OrderTicket market={market} />
          <section className={`panel ${styles.sidePanel}`}>
            <span className="eyebrow">Resolve window</span>
            <div className={styles.sideMetrics}>
              <div>
                <span className={styles.label}>Open</span>
                <strong>{formatTimestamp(market.open_at)}</strong>
              </div>
              <div>
                <span className={styles.label}>Close</span>
                <strong>{formatTimestamp(market.close_at)}</strong>
              </div>
              <div>
                <span className={styles.label}>Resolve</span>
                <strong>{formatTimestamp(market.resolve_at)}</strong>
              </div>
              <div>
                <span className={styles.label}>Resolved outcome</span>
                <strong>{market.resolved_outcome || "pending"}</strong>
              </div>
              <div>
                <span className={styles.label}>Payout progress</span>
                <strong>{market.runtime.completed_payout_count}/{market.runtime.payout_count} completed</strong>
              </div>
            </div>
          </section>
        </aside>
      </section>
    </main>
  );
}
