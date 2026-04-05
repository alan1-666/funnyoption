"use client";

import { useMemo, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import { formatAssetAmount, formatToken } from "@/lib/format";
import { zhMarketStatus, zhOutcome } from "@/lib/locale";
import type { Market } from "@/lib/types";
import styles from "@/components/order-ticket.module.css";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";
const QUANTITY_PRESETS = [1, 5, 10, 25, 100];

function readOutcomePrice(market: Market, outcome: "YES" | "NO") {
  const metadata = market.metadata ?? {};
  if (outcome === "YES") {
    return Math.round(Number(metadata.yesOdds ?? (market.runtime.last_price_yes ? market.runtime.last_price_yes / 100 : 0.5)) * 100);
  }
  return Math.round(Number(metadata.noOdds ?? (market.runtime.last_price_no ? market.runtime.last_price_no / 100 : 0.5)) * 100);
}

export function OrderTicket({ market }: { market: Market }) {
  const { wallet, session, connect, createSession, signOrder, commitOrderNonce, statusMessage } = useTradingSession();
  const [side, setSide] = useState<"BUY_YES" | "BUY_NO">("BUY_YES");
  const [price, setPrice] = useState(() => readOutcomePrice(market, "YES"));
  const [quantity, setQuantity] = useState(10);
  const [status, setStatus] = useState("");

  const outcome = side === "BUY_YES" ? "YES" : "NO";
  const yesPrice = readOutcomePrice(market, "YES");
  const noPrice = readOutcomePrice(market, "NO");
  const freeze = useMemo(() => Math.max(price, 0) * Math.max(quantity, 0), [price, quantity]);
  const normalizedMarketStatus = String(market.status).toUpperCase();
  const marketTradable = normalizedMarketStatus === "OPEN";
  const marketClosedMessage =
    normalizedMarketStatus === "WAITING_RESOLUTION"
      ? "当前市场已进入等待裁决，新的委托不会再进入撮合。"
      : "当前市场已收盘，新的委托不会再进入撮合。";
  const displayedStatus =
    status ||
    (!marketTradable
      ? marketClosedMessage
      : wallet
        ? statusMessage || "钱包已连接，等待交易授权。"
        : "连接钱包后即可开始下单。");
  const accessState = session ? "已开启" : wallet ? "等待授权" : "未连接";
  const selectedPercent = outcome === "YES" ? yesPrice : noPrice;

  async function handleSubmit() {
    try {
      if (!marketTradable) {
        setStatus(
          normalizedMarketStatus === "WAITING_RESOLUTION"
            ? "当前市场正在等待裁决，请等待后台判定结果。"
            : "当前市场已收盘，请等待结算结果。"
        );
        return;
      }

      if (!wallet) {
        setStatus("连接钱包中...");
        await connect();
        setStatus("钱包已连接，请再次点击以开启交易。");
        return;
      }

      if (!session) {
        setStatus("正在授权交易...");
        await createSession(wallet);
        setStatus("交易已开启，请确认参数后再次点击下单。");
        return;
      }

      setStatus("提交订单中...");
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
        throw new Error("Failed to sign order with trading key");
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
      setStatus("订单已提交。");
      if (typeof window !== "undefined") {
        window.dispatchEvent(
          new CustomEvent("funnyoption:order-submitted", {
            detail: {
              marketId: market.market_id,
              userId: orderPayload.userId,
              clientOrderId
            }
          })
        );
      }
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "下单失败");
    }
  }

  const primaryLabel = !marketTradable ? (normalizedMarketStatus === "WAITING_RESOLUTION" ? "等待裁决" : "市场已收盘") : session ? `买入${zhOutcome(outcome)}` : wallet ? "授权交易" : "连接钱包";

  return (
    <div className={`panel ${styles.ticket}`}>
      <div className={styles.railHeader}>
        <div>
          <span className="eyebrow">交易面板</span>
          <h2 className={styles.railTitle}>把下单动作收紧到右侧一条 rail，信息不再和主内容互相抢。</h2>
        </div>
        <span className={styles.marketState}>{zhMarketStatus(market.status)}</span>
      </div>

      <div className={styles.outcomeSwitch}>
        <button
          className={side === "BUY_YES" ? styles.outcomeActive : styles.outcomeButton}
          onClick={() => {
            setSide("BUY_YES");
            setPrice(yesPrice);
          }}
        >
          <span>{zhOutcome("YES")}</span>
          <strong>{yesPrice}¢</strong>
        </button>
        <button
          className={side === "BUY_NO" ? styles.outcomeActive : styles.outcomeButton}
          onClick={() => {
            setSide("BUY_NO");
            setPrice(noPrice);
          }}
        >
          <span>{zhOutcome("NO")}</span>
          <strong>{noPrice}¢</strong>
        </button>
      </div>

      <div className={styles.selectionBanner}>
        <div>
          <span>当前方向</span>
          <strong>{side === "BUY_YES" ? "买入 是" : "买入 否"}</strong>
        </div>
        <div>
          <span>参考价格</span>
          <strong>{selectedPercent}¢</strong>
        </div>
      </div>

      <section className={styles.controlCard}>
        <div className={styles.cardHeader}>
          <span>杠杆</span>
          <strong>1x</strong>
        </div>
        <div className={styles.leverageTrack}>
          <span className={styles.leverageBadge}>1x</span>
          <div className={styles.leverageLine}>
            <span />
            <span />
            <span />
            <span />
            <span />
            <span />
          </div>
        </div>
        <div className={styles.cardFoot}>当前仅开放 1x 预测仓位，后续再扩展更复杂的风险模式。</div>
      </section>

      <section className={styles.controlCard}>
        <div className={styles.cardHeader}>
          <span>委托数量</span>
          <strong>{formatToken(quantity, 0)}<em>份</em></strong>
        </div>

        <div className={styles.inputRow}>
          <label className={styles.inputField}>
            <span className={styles.inputLabel}>价格</span>
            <input
              className={styles.input}
              type="number"
              value={price}
              onChange={(event) => setPrice(Number(event.target.value))}
            />
          </label>
          <label className={styles.inputField}>
            <span className={styles.inputLabel}>份额</span>
            <input
              className={styles.input}
              type="number"
              value={quantity}
              onChange={(event) => setQuantity(Number(event.target.value))}
            />
          </label>
        </div>

        <div className={styles.presetRow}>
          {QUANTITY_PRESETS.map((preset) => (
            <button
              key={preset}
              className={preset === quantity ? styles.presetActive : styles.presetButton}
              onClick={() => setQuantity(preset)}
            >
              {preset}
            </button>
          ))}
          <button className={styles.presetButton} onClick={() => setQuantity(1000)}>
            Max
          </button>
        </div>

        <div className={styles.inlineSummary}>
          <div>
            <span>预计冻结</span>
            <strong>{formatAssetAmount(freeze, "USDT")} USDT</strong>
          </div>
          <div>
            <span>交易权限</span>
            <strong>{accessState}</strong>
          </div>
        </div>
      </section>

      <div className={styles.advancedCard}>
        <span>止盈 / 止损</span>
        <strong>即将开放</strong>
      </div>

      <button className={styles.primary} onClick={handleSubmit} disabled={!marketTradable}>
        {primaryLabel}
      </button>

      <div className={styles.status}>{displayedStatus}</div>
    </div>
  );
}
