import { AdminReadBoard } from "@/components/admin-read-board";
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
import { formatAssetAmount, formatTimestamp, shortenAddress } from "@/lib/format";
import { zhGenericStatus } from "@/lib/locale";
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
          <span className="eyebrow">独立运营后台</span>
          <h1 className={styles.title}>市场发布、首发流动性和结算，在同一套运营工具里完成。</h1>
        </div>
        <div className={styles.topbarActions}>
          <a href={PUBLIC_WEB_BASE_URL} className={styles.linkCard}>
            前台首页
          </a>
          <a href={`${PUBLIC_WEB_BASE_URL}/portfolio`} className={styles.linkCard}>
            用户资产
          </a>
        </div>
      </section>

      <section className={styles.hero}>
        <div className={`panel ${styles.heroPrimary} float-in`}>
          <span className="eyebrow">运营概览</span>
          <p className={styles.heroCopy}>这里保留运营真正高频的动作：发市场、发首发流动性、做结算，以及快速核对账户和读面。</p>
          <div className={styles.metricGrid}>
            <div className={styles.metricCard}>
              <span className={styles.metricLabel}>交易中市场</span>
              <strong className="metric-value">{marketsResult.state === "unavailable" ? "—" : openCount}</strong>
            </div>
            <div className={styles.metricCard}>
              <span className={styles.metricLabel}>已结算市场</span>
              <strong className="metric-value">{marketsResult.state === "unavailable" ? "—" : resolvedCount}</strong>
            </div>
            <div className={styles.metricCard}>
              <span className={styles.metricLabel}>累计成交额</span>
              <strong className="metric-value">{marketsResult.state === "unavailable" ? "API" : formatAssetAmount(matchedNotional, "USDT")}</strong>
            </div>
          </div>
        </div>

        <div className={`panel ${styles.heroSide} float-in float-in-delay-1`}>
          <span className="eyebrow">验收命令</span>
          <h2 className={styles.sideTitle}>页面做操作，命令行做最终验收。</h2>
          <p className={styles.sideCopy}>需要跑完整链路时，直接执行这条 lifecycle 命令，然后回到后台核对市场、账户和终态数据。</p>
          <code className={styles.command}>{lifecycleCommand}</code>
          <p className={styles.sideMeta}>这能作为本地联调、UI 操作和共享服务结果之间的统一基线。</p>
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
          <span className="eyebrow">账户快照</span>
          <h2 className="section-title">快速核对余额、会话、订单和赔付。</h2>
        </div>
        <p className="section-copy">适合在充值、下单、结算之后，快速确认账户和读面有没有一起更新。</p>
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
                  <span className="eyebrow">用户 {snapshot.userId}</span>
                  <h3 className={styles.snapshotTitle}>账户联动</h3>
                </div>
                <span className="pill">{liveSession ? shortenAddress(liveSession.wallet_address) : "暂无会话"}</span>
              </div>

              <div className={styles.snapshotList}>
                <div className={styles.snapshotRow}>
                  <span className={styles.detailLabel}>可用 USDT</span>
                  <strong className={styles.rowValue}>{formatAssetAmount(availableUsdt, "USDT")}</strong>
                </div>
                <div className={styles.snapshotRow}>
                  <span className={styles.detailLabel}>冻结 USDT</span>
                  <strong className={styles.rowValue}>{formatAssetAmount(frozenUsdt, "USDT")}</strong>
                </div>
                <div className={styles.snapshotRow}>
                  <span className={styles.detailLabel}>持仓数</span>
                  <strong className={styles.rowValue}>{snapshot.positions.length}</strong>
                </div>
                <div className={styles.snapshotRow}>
                  <span className={styles.detailLabel}>赔付记录</span>
                  <strong className={styles.rowValue}>{snapshot.payouts.length}</strong>
                </div>
                <div className={styles.snapshotRow}>
                  <span className={styles.detailLabel}>会话</span>
                  <span className={styles.rowMeta}>{liveSession ? `${zhGenericStatus(liveSession.status)} · nonce ${liveSession.last_order_nonce}` : "本地读面里还没有活跃会话。"}</span>
                </div>
                <div className={styles.snapshotRow}>
                  <span className={styles.detailLabel}>充值</span>
                  <span className={styles.rowMeta}>{snapshot.deposits.length > 0 ? `共 ${snapshot.deposits.length} 条，最近 ${formatTimestamp(snapshot.deposits[0]?.credited_at || snapshot.deposits[0]?.created_at || 0)}` : "还没有入账记录。"}</span>
                </div>
                <div className={styles.snapshotRow}>
                  <span className={styles.detailLabel}>订单</span>
                  <span className={styles.rowMeta}>{snapshot.orders.length > 0 ? `${zhGenericStatus(snapshot.orders[0]?.status ?? "")} · 共 ${snapshot.orders.length} 条` : "还没有订单记录。"}</span>
                </div>
                <div className={styles.snapshotRow}>
                  <span className={styles.detailLabel}>提现</span>
                  <span className={styles.rowMeta}>{snapshot.withdrawals.length > 0 ? `${snapshot.withdrawals.length} 笔排队中` : "还没有提现请求。"}</span>
                </div>
              </div>
            </article>
          );
        })}
      </section>

      <section className={styles.sectionHeader}>
        <div>
          <span className="eyebrow">读面</span>
          <h2 className="section-title">成交和市场状态支持搜索与筛选。</h2>
        </div>
        <p className="section-copy">在后台直接按市场、方向、状态和关键词过滤，不用手动翻长列表。</p>
      </section>

      <AdminReadBoard markets={markets} trades={trades} />
    </main>
  );
}
