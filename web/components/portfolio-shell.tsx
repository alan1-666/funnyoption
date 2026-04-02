"use client";

import type { Route } from "next";
import Link from "next/link";
import { useDeferredValue, useEffect, useMemo, useState } from "react";

import { ShellTopBar } from "@/components/shell-top-bar";
import { useTradingSession } from "@/components/trading-session-provider";
import { UserAvatar } from "@/components/user-avatar";
import { getProfile } from "@/lib/api";
import { formatAssetAmount, formatTimestamp, formatToken } from "@/lib/format";
import { presentMarketTitle } from "@/lib/market-display";
import { zhGenericStatus, zhOutcome, zhSide } from "@/lib/locale";
import type { Balance, Market, Order, Payout, Position, UserProfile } from "@/lib/types";
import styles from "@/components/portfolio-shell.module.css";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";
const COLLATERAL_SYMBOL = (process.env.NEXT_PUBLIC_COLLATERAL_SYMBOL ?? "USDT").toUpperCase();

type PortfolioTab = "positions" | "orders" | "history";

function isOpenOrder(order: Order) {
  const terminalStatuses = new Set(["FILLED", "CANCELLED", "FAILED", "COMPLETED", "REVOKED"]);
  if (terminalStatuses.has(String(order.status).toUpperCase())) {
    return false;
  }
  return order.remaining_quantity > 0 || String(order.status).toUpperCase() === "QUEUED";
}

function buildMarketTitleMap(markets: Market[]) {
  const entries = markets.map((market) => [market.market_id, presentMarketTitle(market)] as const);
  return new Map(entries);
}

export function PortfolioShell({
  balances,
  positions,
  orders,
  payouts,
  markets,
  profile
}: {
  balances: Balance[];
  positions: Position[];
  orders: Order[];
  payouts: Payout[];
  markets: Market[];
  profile: UserProfile | null;
}) {
  const { wallet, session, busy, connect, createSession } = useTradingSession();
  const [activeTab, setActiveTab] = useState<PortfolioTab>("positions");
  const [query, setQuery] = useState("");
  const [pendingClaim, setPendingClaim] = useState<Record<string, boolean>>({});
  const [claimStatus, setClaimStatus] = useState<Record<string, string>>({});
  const [profileState, setProfileState] = useState<UserProfile | null>(profile);
  const [copiedWallet, setCopiedWallet] = useState(false);
  const [showWalletQr, setShowWalletQr] = useState(false);
  const deferredQuery = useDeferredValue(query.trim().toLowerCase());

  const titleMap = useMemo(() => buildMarketTitleMap(markets), [markets]);
  const usdt = balances.find((item) => item.asset.toUpperCase() === COLLATERAL_SYMBOL);
  const openOrders = orders.filter(isOpenOrder);
  const currentWallet = wallet?.walletAddress ?? session?.walletAddress ?? "";
  const profileWallet = currentWallet || profileState?.wallet_address || "";
  const activePayoutWallet = currentWallet || "";
  const currentProfile = profileState;
  const qrImageSrc = profileWallet
    ? `https://api.qrserver.com/v1/create-qr-code/?size=480x480&data=${encodeURIComponent(profileWallet)}`
    : "";
  const filteredPositions = positions.filter((position) => {
    if (!deferredQuery) return true;
    const title = (titleMap.get(position.market_id) ?? "").toLowerCase();
    return title.includes(deferredQuery) || zhOutcome(position.outcome).includes(deferredQuery);
  });
  const filteredOpenOrders = openOrders.filter((order) => {
    if (!deferredQuery) return true;
    const title = (titleMap.get(order.market_id) ?? "").toLowerCase();
    return title.includes(deferredQuery) || zhOutcome(order.outcome).includes(deferredQuery) || zhSide(order.side).includes(deferredQuery);
  });
  const filteredPayouts = payouts.filter((payout) => {
    if (!deferredQuery) return true;
    const title = (titleMap.get(payout.market_id) ?? "").toLowerCase();
    return title.includes(deferredQuery) || zhOutcome(payout.winning_outcome).includes(deferredQuery);
  });

  useEffect(() => {
    setProfileState(profile);
  }, [profile]);

  useEffect(() => {
    if (!session?.userId) {
      return;
    }

    let cancelled = false;
    getProfile(session.userId)
      .then((nextProfile) => {
        if (cancelled || !nextProfile) return;
        setProfileState(nextProfile);
      })
      .catch(() => {
        if (cancelled) return;
      });

    return () => {
      cancelled = true;
    };
  }, [session?.userId]);

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
    if (!activePayoutWallet) {
      setClaimStatus((current) => ({ ...current, [eventId]: "请先连接钱包。" }));
      return;
    }

    setPendingClaim((current) => ({ ...current, [eventId]: true }));
    setClaimStatus((current) => ({ ...current, [eventId]: "正在提交领取请求…" }));

    try {
      const payout = payouts.find((item) => item.event_id === eventId);
      const response = await fetch(`${API_BASE_URL}/api/v1/payouts/${eventId}/claim`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          user_id: session?.userId ?? payout?.user_id ?? 1001,
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
          ? session
            ? "交易已开启"
            : "授权交易"
          : "连接钱包";

  return (
    <section className={`${styles.shell} float-in`}>
      <ShellTopBar
        query={query}
        onQueryChange={setQuery}
        balanceDisplay={`${formatAssetAmount(usdt?.available ?? 0, COLLATERAL_SYMBOL)} ${COLLATERAL_SYMBOL}`}
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
            <div className={styles.cashValue}>{formatAssetAmount(usdt?.available ?? 0, COLLATERAL_SYMBOL)}</div>
            <div className={styles.cashAsset}>{COLLATERAL_SYMBOL}</div>

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
                  className={styles.copyButton}
                  onClick={handleCopyWallet}
                  disabled={!profileWallet}
                  aria-label="复制钱包地址"
                  title={copiedWallet ? "已复制" : "复制钱包地址"}
                >
                  <svg viewBox="0 0 24 24" className={styles.iconSvg} aria-hidden="true">
                    <rect x="9" y="9" width="10" height="10" rx="2" />
                    <path d="M7 15H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h7a2 2 0 0 1 2 2v1" />
                  </svg>
                </button>
              </div>

              <button
                type="button"
                className={styles.primaryButton}
                onClick={handlePrimaryAction}
                disabled={busy !== null || Boolean(session)}
              >
                {primaryActionLabel}
              </button>
            </div>
          </div>
        </section>

        <aside className={styles.metricsCard}>
          <div className={styles.sideTopline}>账户概况</div>
          <div className={styles.metricRow}>
            <div className={styles.metricBlock}>
              <span className={styles.metricLabel}>持仓</span>
              <strong className={styles.metricValue}>{positions.length}</strong>
            </div>
            <div className={styles.metricBlock}>
              <span className={styles.metricLabel}>开的订单</span>
              <strong className={styles.metricValue}>{openOrders.length}</strong>
            </div>
            <div className={styles.metricBlock}>
              <span className={styles.metricLabel}>历史结算</span>
              <strong className={styles.metricValue}>{payouts.length}</strong>
            </div>
          </div>
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
        </div>

        <div className={styles.content}>
          {activeTab === "positions" ? (
            filteredPositions.length > 0 ? (
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
            filteredOpenOrders.length > 0 ? (
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
            filteredPayouts.length > 0 ? (
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
                <span className={styles.qrAddress} title={profileWallet}>
                  {profileWallet}
                </span>
                <button
                  type="button"
                  className={styles.copyButton}
                  onClick={handleCopyWallet}
                  aria-label="复制钱包地址"
                  title={copiedWallet ? "已复制" : "复制钱包地址"}
                >
                  <svg viewBox="0 0 24 24" className={styles.iconSvg} aria-hidden="true">
                    <rect x="9" y="9" width="10" height="10" rx="2" />
                    <path d="M7 15H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h7a2 2 0 0 1 2 2v1" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  );
}
