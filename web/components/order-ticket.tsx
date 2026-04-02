"use client";

import { useMemo, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import { formatToken } from "@/lib/format";
import type { Market } from "@/lib/types";
import styles from "@/components/order-ticket.module.css";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";

export function OrderTicket({ market }: { market: Market }) {
  const metadata = market.metadata ?? {};
  const defaultPrice = Math.round(Number(metadata.yesOdds ?? 0.5) * 100);
  const { wallet, session, connect, createSession, signOrder, commitOrderNonce, statusMessage } = useTradingSession();
  const [side, setSide] = useState<"BUY_YES" | "BUY_NO">("BUY_YES");
  const [price, setPrice] = useState(defaultPrice);
  const [quantity, setQuantity] = useState(100);
  const [status, setStatus] = useState<string>("");

  const freeze = useMemo(() => Math.max(price, 0) * Math.max(quantity, 0), [price, quantity]);
  const outcome = side === "BUY_YES" ? "YES" : "NO";

  async function handleSubmit() {
    try {
      if (!wallet) {
        setStatus("Connecting wallet...");
        await connect();
        setStatus("Wallet connected. Click again to enable trading.");
        return;
      }

      if (!session) {
        setStatus("Authorizing trading...");
        await createSession(wallet);
        setStatus("Trading is enabled. Review the ticket and click again to place the order.");
        return;
      }

      setStatus("Placing order...");
      const clientOrderId = `web_${Date.now()}`;
      const orderPayload = await signOrder(
        {
          marketId: market.market_id,
          outcome: outcome.toLowerCase(),
          side: "buy",
          orderType: "limit",
          timeInForce: "gtc",
          price,
          quantity,
          clientOrderId
        },
        session
      );

      if (!orderPayload) {
        throw new Error("Failed to sign order with session key");
      }

      const response = await fetch(`${API_BASE_URL}/api/v1/orders`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          user_id: orderPayload.userId,
          market_id: market.market_id,
          outcome: outcome.toLowerCase(),
          side: "buy",
          type: "limit",
          time_in_force: "gtc",
          price,
          quantity,
          client_order_id: clientOrderId,
          session_id: orderPayload.sessionId,
          wallet_address: orderPayload.walletAddress,
          session_signature: orderPayload.sessionSignature,
          order_nonce: orderPayload.orderNonce,
          requested_at: orderPayload.requestedAt
        })
      });

      if (!response.ok) {
        const payload = (await response.json().catch(() => null)) as { error?: string } | null;
        throw new Error(payload?.error ?? `HTTP ${response.status}`);
      }

      await response.json();
      commitOrderNonce(orderPayload.orderNonce);
      setStatus("Order submitted.");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Failed to place order");
    }
  }

  return (
    <div className={`panel ${styles.ticket}`}>
      <div>
        <span className="eyebrow">Trade ticket</span>
        <h3 className="section-title" style={{ fontSize: "2.4rem", marginTop: 12 }}>
          {outcome} bias
        </h3>
      </div>

      <div className={styles.tabs}>
        <button className={side === "BUY_YES" ? styles.tabActive : styles.tab} onClick={() => setSide("BUY_YES")}>
          Buy Yes
        </button>
        <button className={side === "BUY_NO" ? styles.tabActive : styles.tab} onClick={() => setSide("BUY_NO")}>
          Buy No
        </button>
      </div>

      <div className={styles.grid}>
        <label className={styles.field}>
          <span className={styles.label}>Price (cents)</span>
          <input className={styles.input} type="number" value={price} onChange={(event) => setPrice(Number(event.target.value))} />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Quantity</span>
          <input className={styles.input} type="number" value={quantity} onChange={(event) => setQuantity(Number(event.target.value))} />
        </label>
      </div>

      <div className={styles.summary}>
        <div className={styles.row}>
          <span>Collateral freeze</span>
          <strong>{formatToken(freeze, 0)} USDT</strong>
        </div>
        <div className={styles.row}>
          <span>Wallet</span>
          <strong>{wallet ? "Connected" : "Not connected"}</strong>
        </div>
        <div className={styles.row}>
          <span>Trading access</span>
          <strong>{session ? "Enabled" : wallet ? "Waiting for approval" : "Connect wallet first"}</strong>
        </div>
        <div className={styles.row}>
          <span>Signing</span>
          <strong>{session ? "Session approval active" : wallet ? "Wallet approval required" : "Wallet required"}</strong>
        </div>
      </div>

      <button className={styles.primary} onClick={handleSubmit}>
        {session ? `Place ${outcome}` : wallet ? "Enable Trading" : "Connect Wallet"}
      </button>
      <div className={styles.status}>{status || statusMessage}</div>
    </div>
  );
}
