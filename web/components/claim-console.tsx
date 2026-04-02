"use client";

import { useMemo, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import { formatTimestamp, formatToken, shortenAddress } from "@/lib/format";
import type { Deposit, Payout } from "@/lib/types";
import styles from "@/components/claim-console.module.css";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";

interface ClaimConsoleProps {
  payouts: Payout[];
  deposits: Deposit[];
}

export function ClaimConsole({ payouts, deposits }: ClaimConsoleProps) {
  const { wallet, session } = useTradingSession();
  const primaryWallet = useMemo(() => deposits[0]?.wallet_address ?? wallet?.walletAddress ?? "", [deposits, wallet]);
  const [pending, setPending] = useState<Record<string, boolean>>({});
  const [statusMap, setStatusMap] = useState<Record<string, string>>({});

  async function handleClaim(eventId: string) {
    if (!primaryWallet) {
      setStatusMap((current) => ({ ...current, [eventId]: "No connected payout wallet found yet." }));
      return;
    }

    setPending((current) => ({ ...current, [eventId]: true }));
    setStatusMap((current) => ({ ...current, [eventId]: "Submitting payout claim..." }));

    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/payouts/${eventId}/claim`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          user_id: session?.userId ?? payouts.find((item) => item.event_id === eventId)?.user_id ?? 1001,
          wallet_address: primaryWallet,
          recipient_address: primaryWallet
        })
      });

      if (!response.ok) {
        const payload = (await response.json().catch(() => null)) as { error?: string } | null;
        throw new Error(payload?.error ?? `HTTP ${response.status}`);
      }

      const payload = (await response.json()) as { tx_id?: number };
      setStatusMap((current) => ({ ...current, [eventId]: `Claim submitted${payload.tx_id ? ` #${payload.tx_id}` : ""}` }));
    } catch (error) {
      setStatusMap((current) => ({
        ...current,
        [eventId]: error instanceof Error ? error.message : "Failed to submit claim"
      }));
    } finally {
      setPending((current) => ({ ...current, [eventId]: false }));
    }
  }

  return (
    <section className={`panel ${styles.console}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">Payout claims</span>
          <p className={styles.copy}>Claim any resolved payouts to your connected wallet when they become available.</p>
        </div>
        <span className="pill">{primaryWallet ? shortenAddress(primaryWallet) : "Wallet pending"}</span>
      </div>

      <div className={styles.items}>
        {payouts.map((item) => {
          const completed = item.status === "COMPLETED";
          return (
            <article key={item.event_id} className={styles.item}>
              <div>
                <span className={styles.label}>Settlement Event</span>
                <h3 className={styles.title}>Market #{item.market_id} {item.winning_outcome}</h3>
                <div className={styles.meta}>
                  <span>{item.position_asset}</span>
                  <span>{formatToken(item.settled_quantity, 0)} shares</span>
                  <span>{formatTimestamp(item.updated_at || item.created_at)}</span>
                  <span>{item.status}</span>
                </div>
              </div>

              <div className={styles.amount}>
                <strong className={styles.value}>{formatToken(item.payout_amount, 0)} {item.payout_asset}</strong>
                <button
                  className={styles.button}
                  disabled={completed || pending[item.event_id]}
                  onClick={() => handleClaim(item.event_id)}
                >
                  {completed ? "Claimed" : pending[item.event_id] ? "Submitting..." : "Claim Payout"}
                </button>
                <div className={styles.status}>{statusMap[item.event_id] ?? " "}</div>
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}
