"use client";

import type { Route } from "next";
import Link from "next/link";
import { useDeferredValue, useEffect, useMemo, useState } from "react";

import { ShellTopBar } from "@/components/shell-top-bar";
import { useTradingSession } from "@/components/trading-session-provider";
import { UserAvatar } from "@/components/user-avatar";
import {
  authenticatedFetch,
  getBalancesRead,
  getOrdersRead,
  getPayoutsRead,
  getPositionsRead,
  getProfileRead,
  getUserProposalsRead,
  setApiSessionId
} from "@/lib/api";
import { formatAssetAmount, formatTimestamp, formatToken } from "@/lib/format";
import { presentMarketTitle } from "@/lib/market-display";
import { zhGenericStatus, zhOutcome, zhSide } from "@/lib/locale";
import type {
  ApiCollectionResult,
  ApiItemResult,
  Balance,
  Market,
  Order,
  Payout,
  Position,
  UserProfile
} from "@/lib/types";
import styles from "@/components/portfolio-shell.module.css";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";
const COLLATERAL_SYMBOL = (process.env.NEXT_PUBLIC_COLLATERAL_SYMBOL ?? "USDT").toUpperCase();

type PortfolioTab = "positions" | "orders" | "history" | "proposals";

const TERMINAL_ORDER_STATUSES = new Set(["FILLED", "CANCELLED", "FAILED", "COMPLETED", "REVOKED"]);

function isOpenOrder(order: Order) {
  if (TERMINAL_ORDER_STATUSES.has(String(order.status).toUpperCase())) {
    return false;
  }
  return order.remaining_quantity > 0 || String(order.status).toUpperCase() === "QUEUED";
}

function buildMarketTitleMap(markets: Market[]) {
  const entries = markets.map((market) => [market.market_id, presentMarketTitle(market)] as const);
  return new Map(entries);
}

function toCollectionResult<T>(items: T[]): ApiCollectionResult<T> {
  return {
    state: items.length > 0 ? "ok" : "empty",
    items
  };
}

function toProfileResult(profile: UserProfile | null): ApiItemResult<UserProfile> {
  if (!profile) {
    return {
      state: "not-found",
      item: null
    };
  }

  return {
    state: "ok",
    item: profile
  };
}

export function PortfolioShell({
  balances,
  positions,
  orders,
  payouts,
  markets,
  marketsUnavailable,
  marketsError,
  profile
}: {
  balances: Balance[];
  positions: Position[];
  orders: Order[];
  payouts: Payout[];
  markets: Market[];
  marketsUnavailable?: boolean;
  marketsError?: string;
  profile: UserProfile | null;
}) {
  const { wallet, session, busy, connect, createSession } = useTradingSession();
  const [activeTab, setActiveTab] = useState<PortfolioTab>("positions");
  const [query, setQuery] = useState("");
  const [pendingClaim, setPendingClaim] = useState<Record<string, boolean>>({});
  const [claimStatus, setClaimStatus] = useState<Record<string, string>>({});
  const [balancesResult, setBalancesResult] = useState<ApiCollectionResult<Balance>>(() => toCollectionResult(balances));
  const [positionsResult, setPositionsResult] = useState<ApiCollectionResult<Position>>(() => toCollectionResult(positions));
  const [ordersResult, setOrdersResult] = useState<ApiCollectionResult<Order>>(() => toCollectionResult(orders));
  const [payoutsResult, setPayoutsResult] = useState<ApiCollectionResult<Payout>>(() => toCollectionResult(payouts));
  const [profileResult, setProfileResult] = useState<ApiItemResult<UserProfile>>(() => toProfileResult(profile));
  const [proposalsResult, setProposalsResult] = useState<ApiCollectionResult<Market>>(() => toCollectionResult([]));
  const [portfolioSyncing, setPortfolioSyncing] = useState(false);
  const [copiedWallet, setCopiedWallet] = useState(false);
  const [showWalletQr, setShowWalletQr] = useState(false);
  const deferredQuery = useDeferredValue(query.trim().toLowerCase());

  const sessionUserId = session?.userId && session.userId > 0 ? session.userId : null;
  const titleMap = useMemo(() => buildMarketTitleMap(markets), [markets]);
  const accountBalances = balancesResult.items;
  const accountPositions = positionsResult.items;
  const accountOrders = ordersResult.items;
  const accountPayouts = payoutsResult.items;
  const usdt = accountBalances.find((item) => item.asset.toUpperCase() === COLLATERAL_SYMBOL);
  const openOrders = accountOrders.filter(isOpenOrder);
  const currentWallet = wallet?.walletAddress ?? session?.walletAddress ?? "";
  const currentProfile = profileResult.item;
  const profileWallet = currentWallet || currentProfile?.wallet_address || "";
  const activePayoutWallet = currentWallet || "";
  const copyFeedbackText = copiedWallet ? "钱包地址已复制到剪贴板" : "";
  const balanceDisplay =
    sessionUserId && balancesResult.state !== "unavailable" && !portfolioSyncing
      ? `${formatAssetAmount(usdt?.available ?? 0, COLLATERAL_SYMBOL)} ${COLLATERAL_SYMBOL}`
      : sessionUserId
        ? "余额同步中"
        : wallet
          ? "待授权"
          : "未连接";
  const cashValueDisplay =
    sessionUserId && balancesResult.state !== "unavailable" && !portfolioSyncing
      ? formatAssetAmount(usdt?.available ?? 0, COLLATERAL_SYMBOL)
      : "—";
  const cashAssetLabel =
    sessionUserId && balancesResult.state !== "unavailable" && !portfolioSyncing
      ? COLLATERAL_SYMBOL
      : portfolioSyncing
        ? "同步中"
        : wallet
          ? "待授权"
          : "未连接";
  const overviewStatusCopy = sessionUserId
    ? portfolioSyncing
      ? `正在同步 user #${sessionUserId} 的余额、持仓、订单和结算记录...`
      : [
          balancesResult.state === "unavailable"
            ? balancesResult.error?.message ?? "余额接口暂不可用，请稍后刷新。"
            : "",
          profileResult.state === "unavailable"
            ? profileResult.error?.message ?? "账户资料暂不可用，请稍后刷新。"
            : "",
          marketsUnavailable ? marketsError ?? "市场元数据暂不可用，列表标题将降级为市场编号。" : ""
        ]
          .filter(Boolean)
          .join(" ")
    : wallet
      ? "钱包已连接，请先授权交易密钥后查看当前账户资产。"
      : "未连接钱包时不会读取任何用户账户集合，请先连接钱包。";
  const qrImageSrc = profileWallet
    ? `https://api.qrserver.com/v1/create-qr-code/?size=480x480&data=${encodeURIComponent(profileWallet)}`
    : "";
  const filteredPositions = accountPositions.filter((position) => {
    if (!deferredQuery) return true;
    const title = (titleMap.get(position.market_id) ?? "").toLowerCase();
    return title.includes(deferredQuery) || zhOutcome(position.outcome).includes(deferredQuery);
  });
  const filteredOpenOrders = openOrders.filter((order) => {
    if (!deferredQuery) return true;
    const title = (titleMap.get(order.market_id) ?? "").toLowerCase();
    return title.includes(deferredQuery) || zhOutcome(order.outcome).includes(deferredQuery) || zhSide(order.side).includes(deferredQuery);
  });
  const filteredPayouts = accountPayouts.filter((payout) => {
    if (!deferredQuery) return true;
    const title = (titleMap.get(payout.market_id) ?? "").toLowerCase();
    return title.includes(deferredQuery) || zhOutcome(payout.winning_outcome).includes(deferredQuery);
  });

  useEffect(() => {
    if (!sessionUserId || !session?.sessionId) {
      setPortfolioSyncing(false);
      setBalancesResult(toCollectionResult([]));
      setPositionsResult(toCollectionResult([]));
      setOrdersResult(toCollectionResult([]));
      setPayoutsResult(toCollectionResult([]));
      setProposalsResult(toCollectionResult([]));
      setProfileResult(toProfileResult(null));
      setPendingClaim({});
      setClaimStatus({});
      return;
    }

    let cancelled = false;
    setPortfolioSyncing(true);
    setBalancesResult(toCollectionResult([]));
    setPositionsResult(toCollectionResult([]));
    setOrdersResult(toCollectionResult([]));
    setPayoutsResult(toCollectionResult([]));
    setProposalsResult(toCollectionResult([]));
    setProfileResult(toProfileResult(null));
    setPendingClaim({});
    setClaimStatus({});

    // Ensure the module-level session ID is set before fetching,
    // because parent useEffect (setApiSessionId) runs after child effects.
    setApiSessionId(session.sessionId);

    void Promise.all([
      getBalancesRead(sessionUserId, { ensureAsset: COLLATERAL_SYMBOL }),
      getPositionsRead(sessionUserId),
      getOrdersRead(sessionUserId),
      getPayoutsRead(sessionUserId),
      getProfileRead(sessionUserId),
      getUserProposalsRead(sessionUserId)
    ])
      .then(([nextBalances, nextPositions, nextOrders, nextPayouts, nextProfile, nextProposals]) => {
        if (cancelled) return;
        setBalancesResult(nextBalances);
        setPositionsResult(nextPositions);
        setOrdersResult(nextOrders);
        setPayoutsResult(nextPayouts);
        setProfileResult(nextProfile);
        setProposalsResult(nextProposals);
      })
      .finally(() => {
        if (cancelled) return;
        setPortfolioSyncing(false);
      });

    return () => {
      cancelled = true;
    };
  }, [sessionUserId, session?.sessionId]);

  useEffect(() => {
    if (!copiedWallet) {
      return;
    }

    const timer = window.setTimeout(() => setCopiedWallet(false), 1600);
    return () => window.clearTimeout(timer);
  }, [copiedWallet]);

  async function handlePrimaryAction() {
    try {
      if (!wallet) {
        await connect();
        return;
      }
      if (!session) {
        await createSession(wallet);
      }
    } catch {
      // Provider updates status message.
    }
  }

  async function handleCopyWallet() {
    if (!profileWallet || typeof navigator === "undefined" || !navigator.clipboard?.writeText) {
      return;
    }

    try {
      await navigator.clipboard.writeText(profileWallet);
      setCopiedWallet(true);
    } catch {
      // Ignore clipboard failures silently.
    }
  }

  async function handleClaim(eventId: string) {
    if (!sessionUserId || !activePayoutWallet) {
      setClaimStatus((current) => ({ ...current, [eventId]: "请先连接钱包并授权交易密钥。" }));
      return;
    }

    setPendingClaim((current) => ({ ...current, [eventId]: true }));
    setClaimStatus((current) => ({ ...current, [eventId]: "正在提交领取请求…" }));

    try {
      const response = await authenticatedFetch(`${API_BASE_URL}/api/v1/payouts/${eventId}/claim`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          user_id: sessionUserId,
          wallet_address: activePayoutWallet,
          recipient_address: activePayoutWallet
        })
      });

      if (!response.ok) {
        const payload = (await response.json().catch(() => null)) as { error?: string } | null;
        throw new Error(payload?.error ?? `HTTP ${response.status}`);
      }

      setClaimStatus((current) => ({ ...current, [eventId]: "领取请求已提交。" }));
    } catch (error) {
      setClaimStatus((current) => ({
        ...current,
        [eventId]: error instanceof Error ? error.message : "领取失败"
      }));
    } finally {
      setPendingClaim((current) => ({ ...current, [eventId]: false }));
    }
  }

  const primaryActionLabel =
    busy === "connect"
      ? "连接中..."
      : busy === "session"
        ? "授权中..."
        : wallet
          ? "授权交易"
          : "连接钱包";

  return (
    <section className={`${styles.shell} float-in`}>
      <ShellTopBar
        query={query}
        onQueryChange={setQuery}
        balanceDisplay={balanceDisplay}
        profile={currentProfile}
      />

      <div className={styles.summaryGrid}>
        <section className={styles.cashCard}>
          <button
            type="button"
            className={styles.cornerAction}
            onClick={() => setShowWalletQr(true)}
            disabled={!profileWallet}
            aria-label="分享钱包地址"
            title="分享钱包地址"
          >
            <svg viewBox="0 0 24 24" className={styles.iconSvg} aria-hidden="true">
              <path d="M8 16 16 8" />
              <path d="M10 8h6v6" />
              <path d="M8 8H7a3 3 0 0 0-3 3v6a3 3 0 0 0 3 3h6a3 3 0 0 0 3-3v-1" />
            </svg>
          </button>

          <div className={styles.avatarPanel}>
            <UserAvatar
              profile={currentProfile}
              walletAddress={profileWallet}
              size="xl"
              shape="panel"
              className={styles.cashAvatar}
            />
          </div>

          <div className={styles.cashBody}>
            <div className={styles.cashTopline}>我的余额</div>
            <div className={styles.cashValue}>{cashValueDisplay}</div>
            <div className={styles.cashAsset}>{cashAssetLabel}</div>

            <div className={styles.walletRail}>
              <button
                type="button"
                className={styles.iconAction}
                onClick={() => setShowWalletQr(true)}
                disabled={!profileWallet}
                aria-label="显示钱包二维码"
                title="显示钱包二维码"
              >
                <svg viewBox="0 0 24 24" className={styles.iconSvg} aria-hidden="true">
                  <path d="M4 4h6v6H4z" />
                  <path d="M14 4h6v6h-6z" />
                  <path d="M4 14h6v6H4z" />
                  <path d="M15 15h1" />
                  <path d="M18 15h2v2" />
                  <path d="M14 18h2" />
                  <path d="M17 18h1" />
                  <path d="M20 18v2h-2" />
                </svg>
              </button>

              <div className={styles.addressBar}>
                <span className={styles.addressValue} title={profileWallet || "钱包未连接"}>
                  {profileWallet || "钱包未连接"}
                </span>
                <button
                  type="button"
                  className={`${styles.copyButton} ${copiedWallet ? styles.copyButtonSuccess : ""}`}
                  onClick={handleCopyWallet}
                  disabled={!profileWallet}
                  aria-label="复制钱包地址"
                  title={copiedWallet ? "已复制" : "复制钱包地址"}
                >
                  {copiedWallet ? (
                    <svg viewBox="0 0 24 24" className={styles.iconSvg} aria-hidden="true">
                      <path d="M5 13.2 9.2 17 19 7.5" />
                    </svg>
                  ) : (
                    <svg viewBox="0 0 24 24" className={styles.iconSvg} aria-hidden="true">
                      <rect x="9" y="9" width="10" height="10" rx="2" />
                      <path d="M7 15H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h7a2 2 0 0 1 2 2v1" />
                    </svg>
                  )}
                </button>
              </div>

              <div className={styles.copyFeedbackRow} aria-live="polite">
                <span className={`${styles.copyFeedback} ${copiedWallet ? styles.copyFeedbackVisible : ""}`}>
                  {copyFeedbackText}
                </span>
              </div>

              {!session || busy !== null ? (
                <button
                  type="button"
                  className={styles.primaryButton}
                  onClick={handlePrimaryAction}
                  disabled={busy !== null || Boolean(session)}
                >
                  {primaryActionLabel}
                </button>
              ) : null}
            </div>
          </div>
        </section>

        <aside className={styles.metricsCard}>
          <div className={styles.sideTopline}>账户概况</div>
          <div className={styles.metricRow}>
            <div className={styles.metricBlock}>
              <span className={styles.metricLabel}>持仓</span>
              <strong className={styles.metricValue}>
                {sessionUserId && !portfolioSyncing ? accountPositions.length : "—"}
              </strong>
            </div>
            <div className={styles.metricBlock}>
              <span className={styles.metricLabel}>开的订单</span>
              <strong className={styles.metricValue}>
                {sessionUserId && !portfolioSyncing ? openOrders.length : "—"}
              </strong>
            </div>
            <div className={styles.metricBlock}>
              <span className={styles.metricLabel}>历史结算</span>
              <strong className={styles.metricValue}>
                {sessionUserId && !portfolioSyncing ? accountPayouts.length : "—"}
              </strong>
            </div>
          </div>
          {overviewStatusCopy ? <p className={styles.sideFoot}>{overviewStatusCopy}</p> : null}
        </aside>
      </div>

      <section className={styles.panel}>
        <div className={styles.tabBar}>
          <button
            className={`${styles.tabButton} ${activeTab === "positions" ? styles.tabButtonActive : ""}`}
            onClick={() => setActiveTab("positions")}
          >
            仓位
          </button>
          <button
            className={`${styles.tabButton} ${activeTab === "orders" ? styles.tabButtonActive : ""}`}
            onClick={() => setActiveTab("orders")}
          >
            开的订单
          </button>
          <button
            className={`${styles.tabButton} ${activeTab === "history" ? styles.tabButtonActive : ""}`}
            onClick={() => setActiveTab("history")}
          >
            历史结算
          </button>
          <button
            className={`${styles.tabButton} ${activeTab === "proposals" ? styles.tabButtonActive : ""}`}
            onClick={() => setActiveTab("proposals")}
          >
            我的提案
          </button>
        </div>

        <div className={styles.content}>
          {activeTab === "positions" ? (
            !sessionUserId ? (
              <div className={styles.empty}>
                {wallet ? "钱包已连接，请先授权交易密钥后查看持仓。" : "请先连接钱包并授权交易密钥后查看持仓。"}
              </div>
            ) : portfolioSyncing ? (
              <div className={styles.empty}>正在同步当前账户持仓...</div>
            ) : positionsResult.state === "unavailable" ? (
              <div className={styles.empty}>
                {positionsResult.error?.message ?? "持仓数据暂不可用，请稍后刷新。"}
              </div>
            ) : filteredPositions.length > 0 ? (
              <div className={styles.list}>
                {filteredPositions.map((position) => (
                  <Link
                    key={`${position.market_id}-${position.outcome}`}
                    href={`/markets/${position.market_id}` as Route}
                    className={styles.row}
                  >
                    <div className={styles.rowMain}>
                      <strong className={styles.rowTitle}>
                        {titleMap.get(position.market_id) ?? `市场 ${position.market_id}`}
                      </strong>
                      <span className={styles.rowSub}>
                        {zhOutcome(position.outcome)} · 当前持仓
                      </span>
                    </div>
                    <div className={styles.rowSide}>
                      <strong>{formatToken(position.quantity, 0)} 份</strong>
                      <span>已结算 {formatToken(position.settled_quantity, 0)}</span>
                    </div>
                  </Link>
                ))}
              </div>
            ) : (
              <div className={styles.empty}>{deferredQuery ? "没有匹配的仓位。" : "当前还没有持仓。"}</div>
            )
          ) : null}

          {activeTab === "orders" ? (
            !sessionUserId ? (
              <div className={styles.empty}>
                {wallet ? "钱包已连接，请先授权交易密钥后查看挂单。" : "请先连接钱包并授权交易密钥后查看挂单。"}
              </div>
            ) : portfolioSyncing ? (
              <div className={styles.empty}>正在同步当前账户订单...</div>
            ) : ordersResult.state === "unavailable" ? (
              <div className={styles.empty}>
                {ordersResult.error?.message ?? "订单数据暂不可用，请稍后刷新。"}
              </div>
            ) : filteredOpenOrders.length > 0 ? (
              <div className={styles.list}>
                {filteredOpenOrders.map((order) => (
                  <Link
                    key={order.order_id}
                    href={`/markets/${order.market_id}` as Route}
                    className={styles.row}
                  >
                    <div className={styles.rowMain}>
                      <strong className={styles.rowTitle}>
                        {titleMap.get(order.market_id) ?? `市场 ${order.market_id}`}
                      </strong>
                      <span className={styles.rowSub}>
                        {zhSide(order.side)} · {zhOutcome(order.outcome)} · {zhGenericStatus(order.status)}
                      </span>
                    </div>
                    <div className={styles.rowSide}>
                      <strong>{order.price}¢ × {formatToken(order.quantity, 0)}</strong>
                      <span>剩余 {formatToken(order.remaining_quantity, 0)}</span>
                    </div>
                  </Link>
                ))}
              </div>
            ) : (
              <div className={styles.empty}>{deferredQuery ? "没有匹配的订单。" : "当前没有挂单中的订单。"}</div>
            )
          ) : null}

          {activeTab === "history" ? (
            !sessionUserId ? (
              <div className={styles.empty}>
                {wallet ? "钱包已连接，请先授权交易密钥后查看历史结算。" : "请先连接钱包并授权交易密钥后查看历史结算。"}
              </div>
            ) : portfolioSyncing ? (
              <div className={styles.empty}>正在同步当前账户结算记录...</div>
            ) : payoutsResult.state === "unavailable" ? (
              <div className={styles.empty}>
                {payoutsResult.error?.message ?? "历史结算数据暂不可用，请稍后刷新。"}
              </div>
            ) : filteredPayouts.length > 0 ? (
              <div className={styles.list}>
                {filteredPayouts.map((payout) => {
                  const completed = String(payout.status).toUpperCase() === "COMPLETED";
                  return (
                    <div key={payout.event_id} className={styles.row}>
                      <div className={styles.rowMain}>
                        <strong className={styles.rowTitle}>
                          {titleMap.get(payout.market_id) ?? `市场 ${payout.market_id}`}
                        </strong>
                        <span className={styles.rowSub}>
                          {zhOutcome(payout.winning_outcome)} · {zhGenericStatus(payout.status)} · {formatTimestamp(payout.updated_at || payout.created_at)}
                        </span>
                        {claimStatus[payout.event_id] ? (
                          <span className={styles.rowHint}>{claimStatus[payout.event_id]}</span>
                        ) : null}
                      </div>
                      <div className={styles.rowSide}>
                        <strong>{formatAssetAmount(payout.payout_amount, payout.payout_asset)} {payout.payout_asset}</strong>
                        <button
                          className={styles.secondaryButton}
                          disabled={completed || pendingClaim[payout.event_id]}
                          onClick={() => handleClaim(payout.event_id)}
                        >
                          {completed ? "已领取" : pendingClaim[payout.event_id] ? "提交中..." : "领取"}
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className={styles.empty}>{deferredQuery ? "没有匹配的结算记录。" : "还没有历史结算记录。"}</div>
            )
          ) : null}

          {activeTab === "proposals" ? (
            !sessionUserId ? (
              <div className={styles.empty}>
                {wallet ? "钱包已连接，请先授权交易密钥后查看提案。" : "请先连接钱包并授权交易密钥后查看提案。"}
              </div>
            ) : portfolioSyncing ? (
              <div className={styles.empty}>正在同步您的市场提案...</div>
            ) : proposalsResult.state === "unavailable" ? (
              <div className={styles.empty}>
                {proposalsResult.error?.message ?? "提案数据暂不可用，请稍后刷新。"}
              </div>
            ) : proposalsResult.items.length > 0 ? (
              <div className={styles.list}>
                {proposalsResult.items.map((proposal) => (
                  <div key={proposal.market_id} className={styles.row}>
                    <div className={styles.rowMain}>
                      <strong className={styles.rowTitle}>{proposal.title}</strong>
                      <span className={styles.rowSub}>
                        {formatTimestamp(proposal.created_at)}
                      </span>
                    </div>
                    <div className={styles.rowSide}>
                      <strong style={{
                        color: proposal.status === "OPEN" ? "#4caf50"
                          : proposal.status === "REJECTED" ? "#ff6b6b"
                          : "rgba(255,255,255,0.5)",
                        fontSize: "0.82rem"
                      }}>
                        {proposal.status === "PENDING_REVIEW" ? "审核中"
                          : proposal.status === "REJECTED" ? "已拒绝"
                          : proposal.status === "OPEN" ? "已通过"
                          : zhGenericStatus(proposal.status)}
                      </strong>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className={styles.empty}>还没有提交过市场提案。点击顶部 + 按钮创建一个！</div>
            )
          ) : null}
        </div>
      </section>

      {showWalletQr && profileWallet ? (
        <div
          className={styles.qrOverlay}
          role="presentation"
          onClick={() => setShowWalletQr(false)}
        >
          <div
            className={styles.qrDialog}
            role="dialog"
            aria-modal="true"
            aria-label="钱包二维码"
            onClick={(event) => event.stopPropagation()}
          >
            <button
              type="button"
              className={styles.qrClose}
              onClick={() => setShowWalletQr(false)}
              aria-label="关闭钱包二维码"
            >
              <svg viewBox="0 0 24 24" className={styles.iconSvg} aria-hidden="true">
                <path d="M6 6 18 18" />
                <path d="M18 6 6 18" />
              </svg>
            </button>

            <div className={styles.qrCard}>
              <div className={styles.qrImageWrap}>
                <img
                  src={qrImageSrc}
                  alt="钱包二维码"
                  className={styles.qrImage}
                />
                <div className={styles.qrBrand}>w</div>
              </div>

              <div className={styles.qrFooter}>
                <div className={styles.qrFooterMeta}>
                  <span className={styles.qrAddress} title={profileWallet}>
                    {profileWallet}
                  </span>
                  <span
                    className={`${styles.copyFeedback} ${styles.qrCopyFeedback} ${copiedWallet ? styles.copyFeedbackVisible : ""}`}
                    aria-live="polite"
                  >
                    {copyFeedbackText}
                  </span>
                </div>
                <button
                  type="button"
                  className={`${styles.copyButton} ${copiedWallet ? styles.copyButtonSuccess : ""}`}
                  onClick={handleCopyWallet}
                  aria-label="复制钱包地址"
                  title={copiedWallet ? "已复制" : "复制钱包地址"}
                >
                  {copiedWallet ? (
                    <svg viewBox="0 0 24 24" className={styles.iconSvg} aria-hidden="true">
                      <path d="M5 13.2 9.2 17 19 7.5" />
                    </svg>
                  ) : (
                    <svg viewBox="0 0 24 24" className={styles.iconSvg} aria-hidden="true">
                      <rect x="9" y="9" width="10" height="10" rx="2" />
                      <path d="M7 15H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h7a2 2 0 0 1 2 2v1" />
                    </svg>
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  );
}
