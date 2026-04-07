"use client";

import { useMemo, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import { authenticatedFetch } from "@/lib/api";
import { formatAssetAmount, formatTimestamp, formatToken, shortenAddress } from "@/lib/format";
import { zhGenericStatus, zhOutcome } from "@/lib/locale";
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
      setStatusMap((current) => ({ ...current, [eventId]: "当前没有可用的钱包地址。" }));
      return;
    }

    setPending((current) => ({ ...current, [eventId]: true }));
    setStatusMap((current) => ({ ...current, [eventId]: "正在提交领取请求..." }));

    try {
      const response = await authenticatedFetch(`${API_BASE_URL}/api/v1/payouts/${eventId}/claim`, {
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
      setStatusMap((current) => ({ ...current, [eventId]: `领取请求已提交${payload.tx_id ? ` #${payload.tx_id}` : ""}` }));
    } catch (error) {
      setStatusMap((current) => ({
        ...current,
        [eventId]: error instanceof Error ? error.message : "提交领取失败"
      }));
    } finally {
      setPending((current) => ({ ...current, [eventId]: false }));
    }
  }

  return (
    <section className={`panel ${styles.console}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">赔付领取</span>
          <p className={styles.copy}>当赔付可领取时，可以把结算金额领回当前连接的钱包。</p>
        </div>
        <span className="pill">{primaryWallet ? shortenAddress(primaryWallet) : "钱包待连接"}</span>
      </div>

      <div className={styles.items}>
        {payouts.map((item) => {
          const completed = item.status === "COMPLETED";
          return (
            <article key={item.event_id} className={styles.item}>
              <div>
                <span className={styles.label}>Settlement Event</span>
                <h3 className={styles.title}>市场 #{item.market_id} · {zhOutcome(item.winning_outcome)}</h3>
                <div className={styles.meta}>
                  <span>{item.position_asset}</span>
                  <span>{formatToken(item.settled_quantity, 0)} 份</span>
                  <span>{formatTimestamp(item.updated_at || item.created_at)}</span>
                  <span>{zhGenericStatus(item.status)}</span>
                </div>
              </div>

              <div className={styles.amount}>
                <strong className={styles.value}>{formatAssetAmount(item.payout_amount, item.payout_asset)} {item.payout_asset}</strong>
                <button
                  className={styles.button}
                  disabled={completed || pending[item.event_id]}
                  onClick={() => handleClaim(item.event_id)}
                >
                  {completed ? "已领取" : pending[item.event_id] ? "提交中..." : "领取赔付"}
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
