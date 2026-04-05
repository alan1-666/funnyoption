import { LiveMarketPanel } from "@/components/live-market-panel";
import { MarketOrderActivity } from "@/components/market-order-activity";
import { OrderTicket } from "@/components/order-ticket";
import { ShellTopBar } from "@/components/shell-top-bar";
import { formatTimestamp } from "@/lib/format";
import { getMarketRead, getTradesRead } from "@/lib/api";
import { presentMarketCategory, presentMarketDescription, presentMarketTitle } from "@/lib/market-display";
import { zhMarketStatus, zhOutcome } from "@/lib/locale";
import styles from "@/app/markets/[marketId]/page.module.css";

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

  return (
    <main className="page-shell">
      <ShellTopBar />

      <section className={styles.layout}>
        <div className={styles.main}>
          <div className={`${styles.hero} panel float-in`}>
            <div className={styles.heroCopy}>
              <div className={styles.heroPills}>
                <span className="eyebrow">{category}</span>
                <span className="pill">{zhMarketStatus(market.status)}</span>
                <span className="pill">{market.collateral_asset}</span>
                {sourceLabel !== category ? <span className="pill">{sourceLabel}</span> : null}
              </div>

              <div className={styles.heroHeading}>
                <h1 className={styles.title}>{displayTitle}</h1>
                <p className={styles.copy}>{presentMarketDescription(market)}</p>
              </div>
            </div>

            <div className={styles.heroMedia}>
              {coverImageUrl ? (
                <img className={styles.heroImage} src={coverImageUrl} alt={displayTitle} loading="lazy" />
              ) : (
                <div className={styles.heroFallback}>
                  <span>{category}</span>
                  <strong>{market.collateral_asset}</strong>
                </div>
              )}
              <div className={styles.heroImageScrim} />
              <div className={styles.heroMediaMeta}>
                <div>
                  <span className={styles.mediaLabel}>结算时间</span>
                  <strong>{formatTimestamp(market.resolve_at)}</strong>
                </div>
                <div>
                  <span className={styles.mediaLabel}>赔付进度</span>
                  <strong>{market.runtime.completed_payout_count}/{market.runtime.payout_count}</strong>
                </div>
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
          <MarketOrderActivity marketId={market.market_id} />
          <section className={`panel ${styles.sidePanel}`}>
            <div className={styles.sidePanelHeader}>
              <span className="eyebrow">时间与状态</span>
              <span className="pill">{zhMarketStatus(market.status)}</span>
            </div>
            <div className={styles.sideMetrics}>
              <div>
                <span className={styles.label}>开始交易</span>
                <strong>{formatTimestamp(market.open_at)}</strong>
              </div>
              <div>
                <span className={styles.label}>停止交易</span>
                <strong>{formatTimestamp(market.close_at)}</strong>
              </div>
              <div>
                <span className={styles.label}>结算时间</span>
                <strong>{formatTimestamp(market.resolve_at)}</strong>
              </div>
              <div>
                <span className={styles.label}>结算结果</span>
                <strong>{market.resolved_outcome ? zhOutcome(market.resolved_outcome) : "待结算"}</strong>
              </div>
              <div>
                <span className={styles.label}>来源</span>
                <strong>{sourceLabel}</strong>
              </div>
              <div>
                <span className={styles.label}>赔付进度</span>
                <strong>{market.runtime.completed_payout_count}/{market.runtime.payout_count}</strong>
              </div>
            </div>
          </section>
        </aside>
      </section>
    </main>
  );
}
