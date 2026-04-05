import { LiveMarketPanel } from "@/components/live-market-panel";
import { OrderTicket } from "@/components/order-ticket";
import { ShellTopBar } from "@/components/shell-top-bar";
import { formatAssetAmount, formatTimestamp } from "@/lib/format";
import { getMarketRead, getTradesRead } from "@/lib/api";
import { presentMarketCategory, presentMarketDescription, presentMarketTitle } from "@/lib/market-display";
import { zhMarketStatus, zhOutcome } from "@/lib/locale";
import styles from "@/app/markets/[marketId]/page.module.css";

function readMarketPricePercent(market: Awaited<ReturnType<typeof getMarketRead>> extends { item: infer T | null } ? T : never, outcome: "YES" | "NO") {
  const metadata = market.metadata ?? {};
  if (market.status === "RESOLVED") {
    return market.resolved_outcome === outcome ? 100 : 0;
  }
  if (outcome === "YES") {
    if (market.runtime.last_price_yes > 0) {
      return market.runtime.last_price_yes;
    }
    if (typeof metadata.yesOdds === "number" && metadata.yesOdds > 0) {
      return Math.round(metadata.yesOdds * 100);
    }
    return 50;
  }
  if (market.runtime.last_price_no > 0) {
    return market.runtime.last_price_no;
  }
  if (typeof metadata.noOdds === "number" && metadata.noOdds > 0) {
    return Math.round(metadata.noOdds * 100);
  }
  return 50;
}

export default async function MarketDetailPage({ params }: { params: Promise<{ marketId: string }> }) {
  const { marketId } = await params;
  const numericMarketId = Number(marketId);
  if (!Number.isInteger(numericMarketId) || numericMarketId <= 0) {
    return (
      <main className="page-shell">
        <ShellTopBar />
        <section className={`panel ${styles.hero}`}>
          <span className="eyebrow">市场不存在</span>
          <h1 className={styles.title}>市场 #{marketId} 不是有效的本地市场编号。</h1>
          <p className={styles.copy}>请返回首页，从市场列表中重新选择。</p>
        </section>
      </main>
    );
  }

  const [marketResult, tradesResult] = await Promise.all([getMarketRead(numericMarketId), getTradesRead(numericMarketId)]);
  const trades = tradesResult.items;

  if (marketResult.state === "unavailable") {
    return (
      <main className="page-shell">
        <ShellTopBar />
        <section className={`panel ${styles.hero}`}>
          <span className="eyebrow">接口不可用</span>
          <h1 className={styles.title}>当前无法加载这个市场详情。</h1>
          <p className={styles.copy}>
            {marketResult.error?.message ?? "市场数据暂时不可用。"} 请稍后刷新。
          </p>
        </section>
      </main>
    );
  }

  if (marketResult.state === "not-found" || !marketResult.item) {
    return (
      <main className="page-shell">
        <ShellTopBar />
        <section className={`panel ${styles.hero}`}>
          <span className="eyebrow">市场不存在</span>
          <h1 className={styles.title}>市场 #{marketId} 不存在。</h1>
          <p className={styles.copy}>请返回首页选择其他市场。</p>
        </section>
      </main>
    );
  }

  const market = marketResult.item;
  const tradesUnavailable = tradesResult.state === "unavailable";

  const metadata = market.metadata ?? {};
  const category = presentMarketCategory(market);
  const displayTitle = presentMarketTitle(market);
  const coverImageUrl = String(metadata.coverImage ?? metadata.coverImageUrl ?? metadata.cover_image_url ?? "");
  const coverSourceName = String(metadata.sourceName ?? metadata.coverSourceName ?? metadata.cover_source_name ?? "");
  const sourceLabel = coverSourceName || category;
  const yesPrice = readMarketPricePercent(market, "YES");
  const noPrice = readMarketPricePercent(market, "NO");
  const statusLabel = zhMarketStatus(market.status);
  const marketIdLabel = `#${market.market_id}`;

  return (
    <main className="page-shell">
      <ShellTopBar />

      <section className={styles.layout}>
        <div className={styles.main}>
          <div className={`${styles.hero} panel float-in`}>
            <div className={styles.heroBackdrop}>
              {coverImageUrl ? (
                <img className={styles.heroImage} src={coverImageUrl} alt={displayTitle} loading="lazy" />
              ) : (
                <div className={styles.heroFallback}>
                  <span>{category}</span>
                  <strong>{market.collateral_asset}</strong>
                </div>
              )}
              <div className={styles.heroImageScrim} />
            </div>

            <div className={styles.heroContent}>
              <div className={styles.heroPills}>
                <span className="eyebrow">{category}</span>
                <span className="pill">{statusLabel}</span>
                <span className="pill">{market.collateral_asset}</span>
                <span className="pill">{marketIdLabel}</span>
                {sourceLabel !== category ? <span className="pill">{sourceLabel}</span> : null}
              </div>

              <div className={styles.heroGrid}>
                <div className={styles.heroStory}>
                  <div className={styles.heroHeading}>
                    <h1 className={styles.title}>{displayTitle}</h1>
                    <p className={styles.copy}>{presentMarketDescription(market)}</p>
                  </div>

                  <div className={styles.metricStrip}>
                    <article className={styles.metricCard}>
                      <span className={styles.metricLabel}>累计成交额</span>
                      <strong>{formatAssetAmount(market.runtime.matched_notional, market.collateral_asset)} {market.collateral_asset}</strong>
                    </article>
                    <article className={styles.metricCard}>
                      <span className={styles.metricLabel}>成交笔数</span>
                      <strong>{market.runtime.trade_count}</strong>
                    </article>
                    <article className={styles.metricCard}>
                      <span className={styles.metricLabel}>挂单数量</span>
                      <strong>{market.runtime.active_order_count}</strong>
                    </article>
                  </div>
                </div>

                <section className={styles.contextCard}>
                  <div className={styles.contextHeader}>
                    <div>
                      <span className={styles.contextLabel}>市场合约</span>
                      <h2 className={styles.contextTitle}>像 Worm 一样把事件、赔率和时间线收成一个主舞台。</h2>
                    </div>
                    <span className={styles.contextStatus}>{statusLabel}</span>
                  </div>

                  <div className={styles.outcomeArena}>
                    <article className={styles.outcomeCard}>
                      <span className={styles.outcomeRole}>结果一</span>
                      <strong>{zhOutcome("YES")}</strong>
                      <div className={styles.outcomePrice}>{yesPrice}%</div>
                    </article>

                    <div className={styles.outcomeDivider}>
                      <span className={styles.outcomeDividerBadge}>VS</span>
                      <div className={styles.timelineList}>
                        <div>
                          <span className={styles.timelineLabel}>停止交易</span>
                          <strong>{formatTimestamp(market.close_at)}</strong>
                        </div>
                        <div>
                          <span className={styles.timelineLabel}>裁决 / 结算</span>
                          <strong>{formatTimestamp(market.resolve_at)}</strong>
                        </div>
                        <div>
                          <span className={styles.timelineLabel}>结果</span>
                          <strong>{market.resolved_outcome ? zhOutcome(market.resolved_outcome) : "待定"}</strong>
                        </div>
                      </div>
                    </div>

                    <article className={styles.outcomeCard}>
                      <span className={styles.outcomeRole}>结果二</span>
                      <strong>{zhOutcome("NO")}</strong>
                      <div className={styles.outcomePrice}>{noPrice}%</div>
                    </article>
                  </div>
                </section>
              </div>
            </div>
          </div>

          <div className={styles.liveWrap}>
            {tradesUnavailable ? (
              <section className={`panel ${styles.block}`}>
                <div className={styles.blockHeader}>
                  <div>
                    <span className="eyebrow">行情暂不可用</span>
                    <h2 className={styles.blockTitle}>当前无法加载市场走势</h2>
                  </div>
                </div>
                <p className={styles.blockCopy}>
                  {tradesResult.error?.message ?? "详情页暂时无法读取最近成交数据。"}
                </p>
              </section>
            ) : (
              <LiveMarketPanel market={market} trades={trades} />
            )}
          </div>
        </div>

        <aside className={`${styles.sidebar} float-in float-in-delay-2`}>
          <OrderTicket market={market} />
        </aside>
      </section>
    </main>
  );
}
