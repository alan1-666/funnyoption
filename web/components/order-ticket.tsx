"use client";

import { useMemo, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import { formatAssetAmount, formatHeadlineTimestamp, formatTimestamp, formatToken } from "@/lib/format";
import { presentMarketTitle } from "@/lib/market-display";
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

function readCoverImage(market: Market) {
  const metadata = market.metadata ?? {};
  const raw = metadata.coverImage ?? metadata.coverImageUrl ?? metadata.cover_image_url;
  return typeof raw === "string" ? raw : "";
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
  const coverImage = readCoverImage(market);
  const displayTitle = presentMarketTitle(market);
  const freeze = useMemo(() => Math.max(price, 0) * Math.max(quantity, 0), [price, quantity]);
  const displayedStatus = status || (wallet ? statusMessage || "钱包已连接，等待交易授权。" : "连接钱包后即可开始下单。");
  const accessState = session ? "已开启" : wallet ? "等待授权" : "未连接";

  async function handleSubmit() {
    try {
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
      setStatus("订单已提交。");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "下单失败");
    }
  }

  const primaryLabel = session ? `买入${zhOutcome(outcome)}` : wallet ? "授权交易" : "连接钱包";

  return (
    <div className={`panel ${styles.ticket}`}>
      <div className={styles.marketHead}>
        {coverImage ? (
          <img className={styles.marketThumb} src={coverImage} alt={displayTitle} loading="lazy" />
        ) : (
          <div className={styles.marketThumbFallback}>{market.collateral_asset}</div>
        )}
        <div className={styles.marketMeta}>
          <div className={styles.marketTopline}>
            <span className={styles.marketDate}>{formatHeadlineTimestamp(market.close_at)}</span>
            <span className={styles.marketState}>{zhMarketStatus(market.status)}</span>
          </div>
          <strong className={styles.marketTitle}>{displayTitle}</strong>
        </div>
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

      <section className={styles.summaryCard}>
        <div className={styles.summaryRow}>
          <span>当前方向</span>
          <strong>{side === "BUY_YES" ? "买入 是" : "买入 否"}</strong>
        </div>
        <div className={styles.summaryRow}>
          <span>成交价格</span>
          <strong>{price}¢</strong>
        </div>
        <div className={styles.summaryRow}>
          <span>成交份额</span>
          <strong>{formatToken(quantity, 0)} 份</strong>
        </div>
        <div className={styles.summaryRow}>
          <span>截止时间</span>
          <strong>{formatTimestamp(market.close_at)}</strong>
        </div>
      </section>

      <div className={styles.advancedCard}>
        <span>止盈 / 止损</span>
        <strong>即将开放</strong>
      </div>

      <button className={styles.primary} onClick={handleSubmit}>
        {primaryLabel}
      </button>

      <div className={styles.status}>{displayedStatus}</div>
    </div>
  );
}
