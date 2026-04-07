"use client";

import { startTransition, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import styles from "@/components/admin-market-ops.module.css";
import { useOperatorAccess } from "@/components/operator-access-provider";
import { formatAssetAmount, formatTimestamp, shortenAddress } from "@/lib/format";
import { zhMarketStatus, zhOutcome } from "@/lib/locale";
import type { Market } from "@/lib/types";

interface AdminMarketOpsProps {
  markets: Market[];
}

export function AdminMarketOps({ markets }: AdminMarketOpsProps) {
  const router = useRouter();
  const { wallet, busy: operatorBusy, signResolveMarket, signGenericMessage } = useOperatorAccess();
  const [status, setStatus] = useState("裁决通道待命");
  const [busyMarketId, setBusyMarketId] = useState<number | null>(null);
  const [outcomes, setOutcomes] = useState<Record<number, "YES" | "NO">>({});
  const [rejectReasons, setRejectReasons] = useState<Record<number, string>>({});

  const waitingResolutionMarkets = useMemo(
    () => markets.filter((market) => market.status === "WAITING_RESOLUTION").sort((left, right) => right.updated_at - left.updated_at),
    [markets]
  );
  const pendingReviewMarkets = useMemo(
    () => markets.filter((market) => market.status === "PENDING_REVIEW").sort((left, right) => right.created_at - left.created_at),
    [markets]
  );
  const recentResolved = useMemo(
    () => markets.filter((market) => market.status === "RESOLVED").sort((left, right) => right.updated_at - left.updated_at).slice(0, 4),
    [markets]
  );

  function selectedOutcome(marketId: number) {
    return outcomes[marketId] ?? "YES";
  }

  async function handleResolve(marketId: number) {
    setBusyMarketId(marketId);
    setStatus(`等待市场 #${marketId} 的裁决签名...`);
    try {
      const market = {
        marketId,
        outcome: selectedOutcome(marketId)
      } as const;
      const operator = await signResolveMarket(market);
      if (!operator) {
        setStatus("请先连接白名单运营钱包。");
        return;
      }

      const response = await fetch(`/api/operator/markets/${marketId}/resolve`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          market,
          operator
        })
      });

      const payload = (await response.json().catch(() => null)) as { error?: string; resolved_outcome?: string; operator_wallet_address?: string } | null;
      if (!response.ok) {
        throw new Error(payload?.error ?? `HTTP ${response.status}`);
      }
      setStatus(`市场 #${marketId} 已提交裁决，结果 ${zhOutcome(payload?.resolved_outcome ?? selectedOutcome(marketId))}，签名钱包 ${shortenAddress(payload?.operator_wallet_address ?? operator.walletAddress)}。`);
      startTransition(() => router.refresh());
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "裁决失败");
    } finally {
      setBusyMarketId(null);
    }
  }

  async function handleApprove(marketId: number) {
    setBusyMarketId(marketId);
    setStatus(`等待审核 #${marketId} 的批准签名...`);
    try {
      const requestedAt = Date.now();
      const message = `approve_market:${marketId}:${requestedAt}`;
      const operator = await signGenericMessage(message);
      if (!operator) {
        setStatus("请先连接白名单运营钱包。");
        return;
      }

      const response = await fetch(`/api/operator/markets/${marketId}/approve`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ operator })
      });

      const payload = (await response.json().catch(() => null)) as { error?: string } | null;
      if (!response.ok) {
        throw new Error(payload?.error ?? `HTTP ${response.status}`);
      }
      setStatus(`市场 #${marketId} 已批准上线。`);
      startTransition(() => router.refresh());
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "批准失败");
    } finally {
      setBusyMarketId(null);
    }
  }

  async function handleReject(marketId: number) {
    setBusyMarketId(marketId);
    setStatus(`等待审核 #${marketId} 的拒绝签名...`);
    try {
      const requestedAt = Date.now();
      const message = `reject_market:${marketId}:${requestedAt}`;
      const operator = await signGenericMessage(message);
      if (!operator) {
        setStatus("请先连接白名单运营钱包。");
        return;
      }

      const response = await fetch(`/api/operator/markets/${marketId}/reject`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          reason: rejectReasons[marketId] ?? "",
          operator
        })
      });

      const payload = (await response.json().catch(() => null)) as { error?: string } | null;
      if (!response.ok) {
        throw new Error(payload?.error ?? `HTTP ${response.status}`);
      }
      setStatus(`市场 #${marketId} 提案已拒绝。`);
      startTransition(() => router.refresh());
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "拒绝失败");
    } finally {
      setBusyMarketId(null);
    }
  }

  return (
    <section className={`panel ${styles.panel}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">市场裁决</span>
          <h2 className={styles.title}>只处理已经进入等待裁决的市场。</h2>
          <p className={styles.copy}>后台只负责对等待裁决的市场给出最终结果。提交后，清簿、赔付生成和账务入账仍由共享服务继续完成。</p>
        </div>
        <div className={styles.badges}>
          <span className="pill">等待裁决 {waitingResolutionMarkets.length}</span>
          <span className="pill">{wallet ? shortenAddress(wallet.walletAddress) : "需要连接钱包"}</span>
        </div>
      </div>

      <div className={styles.gateNote}>只有白名单运营钱包可以提交裁决；非白名单钱包即使已连接，也只能查看读面。交易中的市场和 oracle 市场都不会出现在这里。</div>

      <div className={styles.grid}>
        <div className={styles.list}>
          {waitingResolutionMarkets.length > 0 ? (
            waitingResolutionMarkets.map((market) => (
              <article key={market.market_id} className={styles.marketRow}>
                <div className={styles.marketMain}>
                  <div className={styles.marketTitleRow}>
                    <strong className={styles.marketTitle}>#{market.market_id} {market.title}</strong>
                    <span className="pill">{zhMarketStatus(market.status)}</span>
                  </div>
                  <div className={styles.marketMeta}>
                    <span>{market.category?.display_name ?? market.metadata?.category ?? "未分类"}</span>
                    <span>停止交易 {formatTimestamp(market.close_at)}</span>
                    <span>裁决时间 {formatTimestamp(market.resolve_at)}</span>
                    <span>{market.runtime.trade_count} 笔成交</span>
                    <span>成交额 {formatAssetAmount(market.runtime.matched_notional, "USDT")} USDT</span>
                  </div>
                </div>

                <div className={styles.actions}>
                  <div className={styles.toggleGroup}>
                    {(["YES", "NO"] as const).map((outcome) => (
                      <button
                        key={outcome}
                        type="button"
                        className={selectedOutcome(market.market_id) === outcome ? styles.toggleActive : styles.toggle}
                        onClick={() => setOutcomes((current) => ({ ...current, [market.market_id]: outcome }))}
                      >
                        {zhOutcome(outcome)}
                      </button>
                    ))}
                  </div>
                  <button
                    className={styles.resolve}
                    type="button"
                    disabled={busyMarketId === market.market_id || operatorBusy === "connect" || operatorBusy === "sign"}
                    onClick={() => handleResolve(market.market_id)}
                  >
                    {busyMarketId === market.market_id ? "提交中..." : "提交裁决"}
                  </button>
                </div>
              </article>
            ))
          ) : (
            <div className={styles.empty}>当前没有进入等待裁决的市场。</div>
          )}
        </div>

        <aside className={styles.side}>
          <div className={styles.sideCard}>
            <span className="eyebrow">说明</span>
            <p className={styles.sideCopy}>这里负责发起最终裁决。只有等待裁决的人工市场会进入这个队列；oracle 市场仍由预言机自动判定。</p>
          </div>
          {recentResolved.length > 0 ? (
            <div className={styles.sideCard}>
              <span className="eyebrow">最近已结算</span>
              <div className={styles.resolvedList}>
                {recentResolved.map((market) => (
                  <div key={market.market_id} className={styles.resolvedRow}>
                    <strong>#{market.market_id}</strong>
                    <span>{market.category?.display_name ?? market.metadata?.category ?? "未分类"}</span>
                    <span>{market.resolved_outcome ? zhOutcome(market.resolved_outcome) : "—"}</span>
                    <span>{formatTimestamp(market.updated_at)}</span>
                  </div>
                ))}
              </div>
            </div>
          ) : null}
        </aside>
      </div>

      <div className={styles.status} aria-live="polite">
        {status}
      </div>

      <div className={styles.header} style={{ marginTop: 24 }}>
        <div>
          <span className="eyebrow">市场提案审核</span>
          <h2 className={styles.title}>用户提交的市场提案待审核。</h2>
        </div>
        <div className={styles.badges}>
          <span className="pill">待审核 {pendingReviewMarkets.length}</span>
        </div>
      </div>

      <div className={styles.list}>
        {pendingReviewMarkets.length > 0 ? (
          pendingReviewMarkets.map((market) => (
            <article key={market.market_id} className={styles.marketRow}>
              <div className={styles.marketMain}>
                <div className={styles.marketTitleRow}>
                  <strong className={styles.marketTitle}>#{market.market_id} {market.title}</strong>
                  <span className="pill">待审核</span>
                </div>
                <p className={styles.marketCopy}>{market.description || "无描述"}</p>
                {market.options && market.options.length > 0 && (
                  <div className={styles.marketMeta} style={{ gap: 6 }}>
                    <span style={{ color: "var(--text-secondary)", fontSize: 12 }}>选项：</span>
                    {market.options.map((opt) => (
                      <span key={opt.key} className="pill" style={{ fontSize: 12 }}>
                        {opt.label}{opt.short_label && opt.short_label !== opt.label ? ` (${opt.short_label})` : ""}
                      </span>
                    ))}
                  </div>
                )}
                <div className={styles.marketMeta}>
                  <span>{market.category?.display_name ?? "未分类"}</span>
                  <span>提交于 {formatTimestamp(market.created_at)}</span>
                  {market.close_at > 0 && <span>截止 {formatTimestamp(market.close_at)}</span>}
                  <span>提交者 #{market.created_by}</span>
                </div>
              </div>

              <div className={styles.actions}>
                <button
                  className={styles.resolve}
                  type="button"
                  disabled={busyMarketId === market.market_id || operatorBusy === "connect" || operatorBusy === "sign"}
                  onClick={() => handleApprove(market.market_id)}
                >
                  {busyMarketId === market.market_id ? "处理中..." : "批准上线"}
                </button>
                <input
                  type="text"
                  placeholder="拒绝理由（可选）"
                  value={rejectReasons[market.market_id] ?? ""}
                  onChange={(e) => setRejectReasons((cur) => ({ ...cur, [market.market_id]: e.target.value }))}
                  style={{
                    padding: "8px 12px",
                    borderRadius: 10,
                    border: "1px solid var(--line)",
                    background: "transparent",
                    color: "var(--text)",
                    fontSize: 13
                  }}
                />
                <button
                  className={styles.toggle}
                  type="button"
                  disabled={busyMarketId === market.market_id || operatorBusy === "connect" || operatorBusy === "sign"}
                  onClick={() => handleReject(market.market_id)}
                  style={{ color: "#ef4444" }}
                >
                  拒绝
                </button>
              </div>
            </article>
          ))
        ) : (
          <div className={styles.empty}>当前没有待审核的市场提案。</div>
        )}
      </div>
    </section>
  );
}
