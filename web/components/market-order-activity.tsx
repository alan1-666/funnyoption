"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import { getOrdersRead } from "@/lib/api";
import { formatTimestamp, formatToken } from "@/lib/format";
import { zhGenericStatus, zhOutcome, zhSide } from "@/lib/locale";
import type { ApiCollectionResult, Order } from "@/lib/types";
import styles from "@/components/market-order-activity.module.css";

const TERMINAL_ORDER_STATUSES = new Set(["FILLED", "CANCELLED", "FAILED", "COMPLETED", "REVOKED"]);

function emptyOrdersResult(): ApiCollectionResult<Order> {
  return {
    state: "empty",
    items: []
  };
}

function isOpenOrder(order: Order) {
  const status = String(order.status).toUpperCase();
  if (TERMINAL_ORDER_STATUSES.has(status)) {
    return false;
  }
  return order.remaining_quantity > 0 || status === "QUEUED" || status === "PARTIALLY_FILLED";
}

function sortByUpdatedAt(items: Order[]) {
  return [...items].sort((left, right) => {
    const rightStamp = right.updated_at || right.created_at;
    const leftStamp = left.updated_at || left.created_at;
    return rightStamp - leftStamp;
  });
}

function statusTone(status: string) {
  switch (String(status).toUpperCase()) {
    case "FILLED":
      return styles.statusFilled;
    case "PARTIALLY_FILLED":
      return styles.statusPartial;
    case "CANCELLED":
      return styles.statusCancelled;
    default:
      return styles.statusOpen;
  }
}

function describeCancelReason(reason: string) {
  switch (String(reason).toUpperCase()) {
    case "MARKET_CLOSED":
      return "市场收盘后自动取消";
    case "MARKET_RESOLVED":
      return "市场结算后自动取消";
    case "MARKET_NOT_TRADABLE":
      return "市场当前不可交易";
    case "IOC_NO_LIQUIDITY":
      return "立即成交失败，已自动取消";
    case "IOC_PARTIAL_FILL":
      return "未完全成交的部分已自动取消";
    case "VALIDATION_FAILED":
      return "订单校验失败";
    default:
      return reason || zhGenericStatus(reason);
  }
}

interface SubmittedOrderEventDetail {
  marketId?: number;
}

export function MarketOrderActivity({ marketId, embedded = false }: { marketId: number; embedded?: boolean }) {
  const { wallet, session } = useTradingSession();
  const sessionUserId = session?.userId && session.userId > 0 ? session.userId : null;
  const [ordersResult, setOrdersResult] = useState<ApiCollectionResult<Order>>(() => emptyOrdersResult());
  const [syncing, setSyncing] = useState(false);
  const [lastSyncedAt, setLastSyncedAt] = useState(0);

  useEffect(() => {
    let active = true;

    async function refreshOrders() {
      if (!sessionUserId) {
        if (!active) return;
        setOrdersResult(emptyOrdersResult());
        setSyncing(false);
        setLastSyncedAt(0);
        return;
      }

      if (active) {
        setSyncing(true);
      }
      const nextOrders = await getOrdersRead(sessionUserId, marketId);
      if (!active) {
        return;
      }
      setOrdersResult(nextOrders);
      setLastSyncedAt(Date.now());
      setSyncing(false);
    }

    void refreshOrders();
    const poll = window.setInterval(() => {
      void refreshOrders();
    }, 4_000);

    const handleSubmitted = (event: Event) => {
      const detail = (event as CustomEvent<SubmittedOrderEventDetail>).detail;
      if (detail?.marketId !== marketId) {
        return;
      }
      void refreshOrders();
    };

    window.addEventListener("funnyoption:order-submitted", handleSubmitted);
    return () => {
      active = false;
      window.clearInterval(poll);
      window.removeEventListener("funnyoption:order-submitted", handleSubmitted);
    };
  }, [marketId, sessionUserId]);

  const sortedOrders = useMemo(() => sortByUpdatedAt(ordersResult.items), [ordersResult.items]);
  const openOrders = useMemo(() => sortedOrders.filter(isOpenOrder), [sortedOrders]);
  const settledOrders = useMemo(() => sortedOrders.filter((order) => !isOpenOrder(order)).slice(0, 6), [sortedOrders]);

  if (!wallet) {
    return (
      <section className={embedded ? styles.panelEmbedded : `panel ${styles.panel}`}>
        <div className={styles.header}>
          <div>
            <span className="eyebrow">我的订单</span>
            <h2 className={styles.title}>连接钱包后查看订单</h2>
          </div>
        </div>
      </section>
    );
  }

  if (!sessionUserId) {
    return (
      <section className={embedded ? styles.panelEmbedded : `panel ${styles.panel}`}>
        <div className={styles.header}>
          <div>
            <span className="eyebrow">我的订单</span>
            <h2 className={styles.title}>完成授权后查看订单</h2>
          </div>
        </div>
      </section>
    );
  }

  return (
    <section className={embedded ? styles.panelEmbedded : `panel ${styles.panel}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">我的订单</span>
          <h2 className={styles.title}>订单状态</h2>
        </div>
        <div className={styles.headerMeta}>
          <span>{syncing ? "同步中" : `${openOrders.length} 笔挂单`}</span>
          <strong>{lastSyncedAt > 0 ? formatTimestamp(Math.floor(lastSyncedAt / 1000)) : "等待同步"}</strong>
        </div>
      </div>

      {ordersResult.state === "unavailable" ? (
        <div className={styles.emptyState}>{ordersResult.error?.message ?? "当前无法读取你的订单状态，请稍后刷新。"}</div>
      ) : (
        <>
          <div className={styles.section}>
            <div className={styles.sectionHeader}>
              <span>当前挂单</span>
              <strong>{openOrders.length}</strong>
            </div>
            {openOrders.length > 0 ? (
              <div className={styles.list}>
                {openOrders.map((order) => (
                  <article key={order.order_id} className={styles.row}>
                    <div className={styles.rowMain}>
                      <div className={styles.rowTitle}>
                        <strong>{zhSide(order.side)} {zhOutcome(order.outcome)}</strong>
                        <span>{order.price}¢ · {formatToken(order.quantity, 0)} 份</span>
                      </div>
                      <div className={styles.rowMeta}>
                        <span>已成交 {formatToken(order.filled_quantity, 0)} / {formatToken(order.quantity, 0)}</span>
                        <span>剩余 {formatToken(order.remaining_quantity, 0)} 份</span>
                      </div>
                    </div>
                    <div className={styles.rowAside}>
                      <span className={`${styles.statusPill} ${statusTone(order.status)}`}>{zhGenericStatus(order.status)}</span>
                      <small>{formatTimestamp(order.updated_at || order.created_at)}</small>
                    </div>
                  </article>
                ))}
              </div>
            ) : (
              <div className={styles.emptyState}>暂无活动挂单</div>
            )}
          </div>

          <div className={styles.section}>
            <div className={styles.sectionHeader}>
              <span>最近结果</span>
              <Link className={styles.link} href="/portfolio">
                查看全部
              </Link>
            </div>
            {settledOrders.length > 0 ? (
              <div className={styles.list}>
                {settledOrders.map((order) => (
                  <article key={order.order_id} className={styles.row}>
                    <div className={styles.rowMain}>
                      <div className={styles.rowTitle}>
                        <strong>{zhSide(order.side)} {zhOutcome(order.outcome)}</strong>
                        <span>{order.price}¢ · {formatToken(order.quantity, 0)} 份</span>
                      </div>
                      <div className={styles.rowMeta}>
                        <span>已成交 {formatToken(order.filled_quantity, 0)} / {formatToken(order.quantity, 0)}</span>
                        {order.cancel_reason ? <span>{describeCancelReason(order.cancel_reason)}</span> : <span>{zhGenericStatus(order.status)}</span>}
                      </div>
                    </div>
                    <div className={styles.rowAside}>
                      <span className={`${styles.statusPill} ${statusTone(order.status)}`}>{zhGenericStatus(order.status)}</span>
                      <small>{formatTimestamp(order.updated_at || order.created_at)}</small>
                    </div>
                  </article>
                ))}
              </div>
            ) : (
              <div className={styles.emptyState}>暂无历史结果</div>
            )}
          </div>
        </>
      )}
    </section>
  );
}
