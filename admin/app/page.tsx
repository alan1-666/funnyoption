import { AdminMarketOps } from "@/components/admin-market-ops";
import { MarketBootstrap } from "@/components/market-bootstrap";
import { MarketStudio } from "@/components/market-studio";
import { OperatorAccessCard } from "@/components/operator-access-card";
import {
  getBalances,
  getDeposits,
  getMarketsRead,
  getOrders,
  getPayouts,
  getPositions,
  getSessions,
  getTradesRead,
  getWithdrawals
} from "@/lib/api";
import { formatTimestamp, formatToken, shortenAddress } from "@/lib/format";
import styles from "@/app/page.module.css";

const DEMO_USER_IDS = [1001, 1002, 1003] as const;
const PUBLIC_WEB_BASE_URL = process.env.NEXT_PUBLIC_PUBLIC_WEB_BASE_URL ?? "http://127.0.0.1:3000";

export default async function AdminPage() {
  const [marketsResult, tradesResult, userSnapshots] = await Promise.all([
    getMarketsRead(),
    getTradesRead(),
    Promise.all(
      DEMO_USER_IDS.map(async (userId) => {
        const [balances, positions, deposits, withdrawals, payouts, sessions, orders] = await Promise.all([
          getBalances(userId),
          getPositions(userId),
          getDeposits(userId),
          getWithdrawals(userId),
          getPayouts(userId),
          getSessions(userId),
          getOrders(userId)
        ]);

        return {
          userId,
          balances,
          positions,
          deposits,
          withdrawals,
          payouts,
          sessions,
          orders
        };
      })
    )
  ]);

  const markets = marketsResult.items;
  const trades = tradesResult.items;
  const openCount = markets.filter((market) => market.status === "OPEN").length;
  const resolvedCount = markets.filter((market) => market.status === "RESOLVED").length;
  const matchedNotional = markets.reduce((sum, market) => sum + market.runtime.matched_notional, 0);
  const lifecycleCommand = "set -a; source /Users/zhangza/code/funnyoption/.env.local; set +a; go run ./cmd/local-lifecycle";

  return (
    <main className={`page-shell ${styles.pageShell}`}>
      <section className={styles.topbar}>
        <div className={styles.brandBlock}>
          <span className="eyebrow">Dedicated admin service</span>
          <h1 className={styles.title}>Operate markets from a wallet-gated runtime that is no longer embedded in the public web shell.</h1>
        </div>
        <div className={styles.topbarActions}>
          <a href={PUBLIC_WEB_BASE_URL} className={styles.linkCard}>
            Public Tape
          </a>
          <a href={`${PUBLIC_WEB_BASE_URL}/portfolio`} className={styles.linkCard}>
            User Portfolio
          </a>
        </div>
      </section>

      <section className={styles.hero}>
        <div className={`panel ${styles.heroPrimary} float-in`}>
          <span className="eyebrow">Operator lane</span>
          <p className={styles.heroCopy}>The operator surface now lives in its own service boundary. Market creation, first-liquidity bootstrap, and resolution now run through admin-owned API routes so the public app no longer carries privileged controls.</p>
          <div className={styles.metricGrid}>
            <div className={styles.metricCard}>
              <span className={styles.metricLabel}>Open markets</span>
              <strong className="metric-value">{marketsResult.state === "unavailable" ? "—" : openCount}</strong>
            </div>
            <div className={styles.metricCard}>
              <span className={styles.metricLabel}>Resolved markets</span>
              <strong className="metric-value">{marketsResult.state === "unavailable" ? "—" : resolvedCount}</strong>
            </div>
            <div className={styles.metricCard}>
              <span className={styles.metricLabel}>Matched notional</span>
              <strong className="metric-value">{marketsResult.state === "unavailable" ? "API" : `${formatToken(matchedNotional / 100, 0)}`}</strong>
            </div>
          </div>
        </div>

        <div className={`panel ${styles.heroSide} float-in float-in-delay-1`}>
          <span className="eyebrow">Lifecycle proof</span>
          <h2 className={styles.sideTitle}>The local lifecycle runner still anchors the read-side proof.</h2>
          <p className={styles.sideCopy}>This admin service now owns the full operator lane for create, bootstrap, and resolve. Deposit truthfulness and lifecycle settlement verification still line up with the shared local runner below.</p>
          <code className={styles.command}>{lifecycleCommand}</code>
          <p className={styles.sideMeta}>Use the cards below to confirm market state, sessions, balances, and terminal reads after any local run.</p>
        </div>
      </section>

      <section className={styles.adminGrid}>
        <OperatorAccessCard />
        <MarketStudio existingMarkets={markets} />
        <MarketBootstrap markets={markets} />
        <AdminMarketOps markets={markets} />
      </section>

      <section className={styles.sectionHeader}>
        <div>
          <span className="eyebrow">User snapshots</span>
          <h2 className="section-title">Balances, sessions, orders, and payouts in one glance.</h2>
        </div>
        <p className="section-copy">These demo-user cards are still the fastest way to check whether lifecycle actions moved balances, positions, deposits, and settlement outputs in the shared backend.</p>
      </section>

      <section className={styles.snapshotGrid}>
        {userSnapshots.map((snapshot) => {
          const availableUsdt = snapshot.balances.find((item) => item.asset === "USDT")?.available ?? 0;
          const frozenUsdt = snapshot.balances.find((item) => item.asset === "USDT")?.frozen ?? 0;
          const liveSession = snapshot.sessions.find((item) => item.status === "ACTIVE");
          return (
            <article key={snapshot.userId} className={`panel ${styles.snapshotCard}`}>
              <div className={styles.snapshotHeader}>
                <div>
                  <span className="eyebrow">User {snapshot.userId}</span>
                  <h3 className={styles.snapshotTitle}>Lifecycle footprint</h3>
                </div>
                <span className="pill">{liveSession ? shortenAddress(liveSession.wallet_address) : "No active session"}</span>
              </div>

              <div className={styles.statGrid}>
                <div className={styles.statBlock}>
                  <span>Available USDT</span>
                  <strong>{formatToken(availableUsdt, 0)}</strong>
                </div>
                <div className={styles.statBlock}>
                  <span>Frozen USDT</span>
                  <strong>{formatToken(frozenUsdt, 0)}</strong>
                </div>
                <div className={styles.statBlock}>
                  <span>Open positions</span>
                  <strong>{snapshot.positions.length}</strong>
                </div>
                <div className={styles.statBlock}>
                  <span>Payout rows</span>
                  <strong>{snapshot.payouts.length}</strong>
                </div>
              </div>

              <div className={styles.detailGrid}>
                <div className={styles.detailBlock}>
                  <span className={styles.detailLabel}>Sessions</span>
                  <p>{liveSession ? `${liveSession.status} · nonce ${liveSession.last_order_nonce}` : "No active session in local reads."}</p>
                </div>
                <div className={styles.detailBlock}>
                  <span className={styles.detailLabel}>Deposits</span>
                  <p>{snapshot.deposits.length > 0 ? `${snapshot.deposits.length} row(s), latest ${formatTimestamp(snapshot.deposits[0]?.credited_at || snapshot.deposits[0]?.created_at || 0)}` : "No credited deposits yet."}</p>
                </div>
                <div className={styles.detailBlock}>
                  <span className={styles.detailLabel}>Orders</span>
                  <p>{snapshot.orders.length > 0 ? `${snapshot.orders[0]?.status} · ${snapshot.orders.length} row(s)` : "No orders yet."}</p>
                </div>
                <div className={styles.detailBlock}>
                  <span className={styles.detailLabel}>Withdrawals</span>
                  <p>{snapshot.withdrawals.length > 0 ? `${snapshot.withdrawals.length} queued` : "No withdrawals queued."}</p>
                </div>
              </div>
            </article>
          );
        })}
      </section>

      <section className={styles.sectionHeader}>
        <div>
          <span className="eyebrow">Recent tape</span>
          <h2 className="section-title">Latest trades and market terminals.</h2>
        </div>
        <p className="section-copy">This feed stays wired to the shared backend reads so operators can check matching and terminal market state without dropping into SQL.</p>
      </section>

      <section className={styles.feedGrid}>
        <div className={`panel ${styles.feedCard}`}>
          <div className={styles.feedHeader}>
            <span className="eyebrow">Trades</span>
            <span className="pill">{trades.length} recent</span>
          </div>
          <div className={styles.feedList}>
            {trades.length > 0 ? (
              trades.slice(0, 8).map((trade) => (
                <div key={trade.trade_id} className={styles.feedRow}>
                  <strong>#{trade.market_id} {trade.outcome}</strong>
                  <span>{trade.taker_side} {formatToken(trade.quantity, 0)} @ {trade.price}c</span>
                  <span>{formatTimestamp(trade.occurred_at)}</span>
                </div>
              ))
            ) : (
              <div className={styles.emptyState}>No trades are visible yet.</div>
            )}
          </div>
        </div>

        <div className={`panel ${styles.feedCard}`}>
          <div className={styles.feedHeader}>
            <span className="eyebrow">Markets</span>
            <span className="pill">{markets.length} tracked</span>
          </div>
          <div className={styles.feedList}>
            {markets.length > 0 ? (
              markets.slice(0, 8).map((market) => (
                <div key={market.market_id} className={styles.feedRow}>
                  <strong>#{market.market_id}</strong>
                  <span>{market.status}{market.resolved_outcome ? ` · ${market.resolved_outcome}` : ""}</span>
                  <span>{formatTimestamp(market.updated_at)}</span>
                </div>
              ))
            ) : (
              <div className={styles.emptyState}>No markets available in the local API.</div>
            )}
          </div>
        </div>
      </section>
    </main>
  );
}
