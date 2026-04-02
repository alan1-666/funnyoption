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
  const { wallet, busy: operatorBusy, signResolveMarket } = useOperatorAccess();
  const [status, setStatus] = useState("结算通道待命");
  const [busyMarketId, setBusyMarketId] = useState<number | null>(null);
  const [outcomes, setOutcomes] = useState<Record<number, "YES" | "NO">>({});

  const openMarkets = useMemo(
    () => markets.filter((market) => market.status === "OPEN").sort((left, right) => right.updated_at - left.updated_at),
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
    setStatus(`等待市场 #${marketId} 的结算签名...`);
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
      setStatus(`市场 #${marketId} 已提交结算，结果 ${zhOutcome(payload?.resolved_outcome ?? selectedOutcome(marketId))}，签名钱包 ${shortenAddress(payload?.operator_wallet_address ?? operator.walletAddress)}。`);
      startTransition(() => router.refresh());
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "结算失败");
    } finally {
      setBusyMarketId(null);
    }
  }

  return (
    <section className={`panel ${styles.panel}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">市场结算</span>
          <h2 className={styles.title}>在后台结算交易中的市场。</h2>
          <p className={styles.copy}>提交结算后，清簿、赔付生成和账务入账仍由共享服务继续完成，后台只负责发起动作和核对结果。</p>
        </div>
        <div className={styles.badges}>
          <span className="pill">交易中 {openMarkets.length}</span>
          <span className="pill">{wallet ? shortenAddress(wallet.walletAddress) : "需要连接钱包"}</span>
        </div>
      </div>

      <div className={styles.gateNote}>只有白名单运营钱包可以提交结算；非白名单钱包即使已连接，也只能查看读面。</div>

      <div className={styles.grid}>
        <div className={styles.list}>
          {openMarkets.length > 0 ? (
            openMarkets.map((market) => (
              <article key={market.market_id} className={styles.marketRow}>
                <div className={styles.marketMain}>
                  <div className={styles.marketTitleRow}>
                    <strong className={styles.marketTitle}>#{market.market_id} {market.title}</strong>
                    <span className="pill">{zhMarketStatus(market.status)}</span>
                  </div>
                  <div className={styles.marketMeta}>
                    <span>{market.category?.display_name ?? market.metadata?.category ?? "未分类"}</span>
                    <span>停止交易 {formatTimestamp(market.close_at)}</span>
                    <span>结算时间 {formatTimestamp(market.resolve_at)}</span>
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
                    {busyMarketId === market.market_id ? "提交中..." : "执行结算"}
                  </button>
                </div>
              </article>
            ))
          ) : (
            <div className={styles.empty}>当前没有可结算的交易中市场。</div>
          )}
        </div>

        <aside className={styles.side}>
          <div className={styles.sideCard}>
            <span className="eyebrow">说明</span>
            <p className={styles.sideCopy}>这里负责发起结算动作。后续的清簿、赔付和余额入账仍由共享服务自动完成。</p>
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
    </section>
  );
}
