"use client";

import { startTransition, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import styles from "@/components/admin-market-ops.module.css";
import { useOperatorAccess } from "@/components/operator-access-provider";
import { formatTimestamp, formatToken, shortenAddress } from "@/lib/format";
import type { Market } from "@/lib/types";

interface AdminMarketOpsProps {
  markets: Market[];
}

export function AdminMarketOps({ markets }: AdminMarketOpsProps) {
  const router = useRouter();
  const { wallet, busy: operatorBusy, signResolveMarket } = useOperatorAccess();
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
    setStatus(`Requesting operator signature for market #${marketId}...`);
    try {
      const market = {
        marketId,
        outcome: selectedOutcome(marketId)
      } as const;
      const operator = await signResolveMarket(market);
      if (!operator) {
        setStatus("Connect an allowlisted wallet before resolving a market.");
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
      setStatus(`Market #${marketId} queued for resolution as ${payload?.resolved_outcome ?? selectedOutcome(marketId)} by ${shortenAddress(payload?.operator_wallet_address ?? operator.walletAddress)}`);
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
          <h2 className={styles.title}>Resolve live markets from the dedicated admin runtime.</h2>
          <p className={styles.copy}>Resolution now runs through an admin-owned API lane that requires an allowlisted wallet signature before it proxies the resolve event to the shared backend.</p>
        </div>
        <div className={styles.badges}>
          <span className="pill">{openMarkets.length} open</span>
          <span className="pill">{recentResolved.length} recent terminals</span>
          <span className="pill">{wallet ? shortenAddress(wallet.walletAddress) : "Wallet required"}</span>
        </div>
      </div>

      <div className={styles.gateNote}>
        Wallet-gated operator access is enforced at the admin service boundary. A connected but non-allowlisted wallet can inspect reads here, but resolution requests are denied before they hit the shared market event API.
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
                  <button
                    className={styles.resolve}
                    type="button"
                    disabled={busyMarketId === market.market_id || operatorBusy === "connect" || operatorBusy === "sign"}
                    onClick={() => handleResolve(market.market_id)}
                  >
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
            <p className={styles.sideCopy}>The admin service only gates and forwards the resolve action. Matching cleanup, payout creation, settlement credits, and terminal reads still happen in the existing backend services.</p>
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
