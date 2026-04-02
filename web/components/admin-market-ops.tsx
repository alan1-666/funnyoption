"use client";

import { startTransition, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import type { Market } from "@/lib/types";
import { formatTimestamp, formatToken } from "@/lib/format";
import styles from "@/components/admin-market-ops.module.css";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";

interface AdminMarketOpsProps {
  markets: Market[];
}

export function AdminMarketOps({ markets }: AdminMarketOpsProps) {
  const router = useRouter();
  const [status, setStatus] = useState("Resolution lane idle");
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
    setStatus(`Resolving market #${marketId}...`);
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/markets/${marketId}/resolve`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          outcome: selectedOutcome(marketId)
        })
      });
      const payload = (await response.json().catch(() => null)) as { error?: string; resolved_outcome?: string } | null;
      if (!response.ok) {
        throw new Error(payload?.error ?? `HTTP ${response.status}`);
      }
      setStatus(`Market #${marketId} queued for resolution as ${payload?.resolved_outcome ?? selectedOutcome(marketId)}`);
      startTransition(() => router.refresh());
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Failed to resolve market");
    } finally {
      setBusyMarketId(null);
    }
  }

  return (
    <section className={`panel ${styles.panel}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">Resolution desk</span>
          <h2 className={styles.title}>Resolve live markets without dropping back into the public app.</h2>
          <p className={styles.copy}>This is the operator close-out lane. Choose the terminal outcome, publish the resolve event, and let settlement update market, order, payout, and balance reads behind the scenes.</p>
        </div>
        <div className={styles.badges}>
          <span className="pill">{openMarkets.length} open</span>
          <span className="pill">{recentResolved.length} recent terminals</span>
        </div>
      </div>

      <div className={styles.grid}>
        <div className={styles.list}>
          {openMarkets.length > 0 ? (
            openMarkets.map((market) => (
              <article key={market.market_id} className={styles.marketRow}>
                <div className={styles.marketMain}>
                  <div className={styles.marketTitleRow}>
                    <strong className={styles.marketTitle}>#{market.market_id} {market.title}</strong>
                    <span className="pill">{market.status}</span>
                  </div>
                  <p className={styles.marketCopy}>{market.description || "No market description provided."}</p>
                  <div className={styles.marketMeta}>
                    <span>Close {formatTimestamp(market.close_at)}</span>
                    <span>Resolve {formatTimestamp(market.resolve_at)}</span>
                    <span>{market.runtime.trade_count} trades</span>
                    <span>{formatToken(market.runtime.matched_notional / 100, 0)} USDT matched</span>
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
                        {outcome}
                      </button>
                    ))}
                  </div>
                  <button className={styles.resolve} type="button" disabled={busyMarketId === market.market_id} onClick={() => handleResolve(market.market_id)}>
                    {busyMarketId === market.market_id ? "Resolving..." : "Resolve Market"}
                  </button>
                </div>
              </article>
            ))
          ) : (
            <div className={styles.empty}>No open markets are available to resolve.</div>
          )}
        </div>

        <aside className={styles.side}>
          <div className={styles.sideCard}>
            <span className="eyebrow">Settlement note</span>
            <p className={styles.sideCopy}>The button above only emits the resolve event. Matching cleanup, payout creation, settlement credits, and terminal reads still happen in the existing backend services.</p>
          </div>
          <div className={styles.sideCard}>
            <span className="eyebrow">Recent terminals</span>
            <div className={styles.resolvedList}>
              {recentResolved.length > 0 ? (
                recentResolved.map((market) => (
                  <div key={market.market_id} className={styles.resolvedRow}>
                    <strong>#{market.market_id}</strong>
                    <span>{market.resolved_outcome || "—"}</span>
                    <span>{formatTimestamp(market.updated_at)}</span>
                  </div>
                ))
              ) : (
                <div className={styles.emptyCompact}>No resolved markets yet.</div>
              )}
            </div>
          </div>
        </aside>
      </div>

      <div className={styles.status} aria-live="polite">
        {status}
      </div>
    </section>
  );
}
