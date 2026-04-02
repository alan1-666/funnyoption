"use client";

import { useEffect, useMemo, useState } from "react";

import { createWsUrl } from "@/lib/ws";
import { formatTimestamp, formatToken } from "@/lib/format";
import type { Market, Trade } from "@/lib/types";
import styles from "@/components/live-market-panel.module.css";

interface QuoteLevel {
  price: number;
  quantity: number;
}

interface QuoteDepthEvent {
  market_id: number;
  outcome: string;
  book_key: string;
  bids: QuoteLevel[];
  asks: QuoteLevel[];
  occurred_at_millis: number;
}

interface QuoteTickerEvent {
  market_id: number;
  outcome: string;
  book_key: string;
  last_price: number;
  last_quantity: number;
  best_bid: number;
  best_ask: number;
  occurred_at_millis: number;
}

interface QuoteCandle {
  bucket_start_millis: number;
  bucket_end_millis: number;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
  trade_count: number;
}

interface QuoteCandleEvent {
  market_id: number;
  outcome: string;
  book_key: string;
  interval_sec: number;
  candles: QuoteCandle[];
  occurred_at_millis: number;
}

interface MarketEnvelope {
  type: string;
  payload: {
    status?: string;
    resolved_outcome?: string;
    winning_outcome?: string;
    occurred_at_millis?: number;
    payout_amount?: number;
    user_id?: number;
  };
}

interface SideState {
  ticker: QuoteTickerEvent | null;
  depth: QuoteDepthEvent | null;
  candles: QuoteCandle[];
}

function initialTicker(market: Market, outcome: "YES" | "NO"): QuoteTickerEvent | null {
  let price = 0;
  if (market.status === "RESOLVED") {
    price = market.resolved_outcome === outcome ? 100 : 0;
  } else if (outcome === "YES" && market.runtime.last_price_yes > 0) {
    price = market.runtime.last_price_yes;
  } else if (outcome === "NO" && market.runtime.last_price_no > 0) {
    price = market.runtime.last_price_no;
  }
  if (price <= 0 && market.runtime.last_trade_at === 0) {
    return null;
  }
  return {
    market_id: market.market_id,
    outcome,
    book_key: `${market.market_id}:${outcome}`,
    last_price: price,
    last_quantity: 0,
    best_bid: 0,
    best_ask: 0,
    occurred_at_millis: market.updated_at * 1000
  };
}

function buildCandlesFromTrades(marketId: number, outcome: "YES" | "NO", trades: Trade[], intervalMillis = 60_000): QuoteCandle[] {
  const relevant = trades
    .filter((trade) => trade.market_id === marketId && String(trade.outcome).toUpperCase() === outcome)
    .sort((left, right) => left.occurred_at - right.occurred_at);

  const candles: QuoteCandle[] = [];
  for (const trade of relevant) {
    const occurredAtMillis = trade.occurred_at * 1000;
    const bucketStart = occurredAtMillis - (occurredAtMillis % intervalMillis);
    const bucketEnd = bucketStart + intervalMillis;
    const last = candles[candles.length - 1];
    if (last && last.bucket_start_millis === bucketStart) {
      last.high = Math.max(last.high, trade.price);
      last.low = Math.min(last.low, trade.price);
      last.close = trade.price;
      last.volume += trade.quantity;
      last.trade_count += 1;
      continue;
    }
    candles.push({
      bucket_start_millis: bucketStart,
      bucket_end_millis: bucketEnd,
      open: trade.price,
      high: trade.price,
      low: trade.price,
      close: trade.price,
      volume: trade.quantity,
      trade_count: 1
    });
  }
  return candles.slice(-18);
}

function renderCandlePath(candles: QuoteCandle[]) {
  if (candles.length === 0) {
    return null;
  }
  const allPrices = candles.flatMap((item) => [item.high, item.low]);
  const minPrice = Math.min(...allPrices) - 2;
  const maxPrice = Math.max(...allPrices) + 2;
  const safeRange = Math.max(maxPrice - minPrice, 1);
  const chartHeight = 124;
  const chartWidth = 360;
  const slotWidth = chartWidth / Math.max(candles.length, 1);
  const bodyWidth = Math.max(slotWidth * 0.42, 5);

  const priceToY = (price: number) => {
    const ratio = (price - minPrice) / safeRange;
    return chartHeight - ratio * (chartHeight - 8) - 4;
  };

  return {
    minPrice,
    maxPrice,
    chartHeight,
    chartWidth,
    slotWidth,
    bodyWidth,
    priceToY
  };
}

export function LiveMarketPanel({ market, trades }: { market: Market; trades: Trade[] }) {
  const [sides, setSides] = useState<Record<"YES" | "NO", SideState>>(() => ({
    YES: {
      ticker: initialTicker(market, "YES"),
      depth: null,
      candles: buildCandlesFromTrades(market.market_id, "YES", trades)
    },
    NO: {
      ticker: initialTicker(market, "NO"),
      depth: null,
      candles: buildCandlesFromTrades(market.market_id, "NO", trades)
    }
  }));
  const [events, setEvents] = useState<MarketEnvelope[]>([]);
  const [status, setStatus] = useState(
    market.status === "RESOLVED" ? "Market resolved; waiting for terminal stream updates" : "Waiting for live quote stream"
  );

  const streams = useMemo(
    () => [
      { type: "ticker", key: `${market.market_id}:YES` },
      { type: "ticker", key: `${market.market_id}:NO` },
      { type: "depth", key: `${market.market_id}:YES` },
      { type: "depth", key: `${market.market_id}:NO` },
      { type: "candle", key: `${market.market_id}:YES` },
      { type: "candle", key: `${market.market_id}:NO` }
    ],
    [market.market_id]
  );

  useEffect(() => {
    const sockets = streams.map((stream) => {
      const url = createWsUrl(`/ws?stream=${stream.type}&book_key=${stream.key}`);
      const socket = new WebSocket(url);
      socket.onopen = () => setStatus("Streaming over WebSocket");
      socket.onmessage = (event) => {
        try {
          const payload = JSON.parse(event.data) as QuoteDepthEvent | QuoteTickerEvent | QuoteCandleEvent;
          const outcome = String(payload.outcome).toUpperCase() as "YES" | "NO";
          setSides((current) => ({
            ...current,
            [outcome]: stream.type === "ticker"
              ? { ...current[outcome], ticker: payload as QuoteTickerEvent }
              : stream.type === "depth"
                ? { ...current[outcome], depth: payload as QuoteDepthEvent }
                : { ...current[outcome], candles: (payload as QuoteCandleEvent).candles ?? current[outcome].candles }
          }));
        } catch {
          setStatus("Failed to decode quote stream");
        }
      };
      socket.onerror = () => setStatus("Quote stream degraded");
      socket.onclose = () => setStatus("Quote stream idle");
      return socket;
    });

    const marketSocket = new WebSocket(createWsUrl(`/ws?stream=market&market_id=${market.market_id}`));
    marketSocket.onopen = () => setStatus("Streaming over WebSocket");
    marketSocket.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data) as MarketEnvelope;
        setEvents((current) => [payload, ...current].slice(0, 8));
      } catch {
        setStatus("Failed to decode market stream");
      }
    };
    marketSocket.onerror = () => setStatus("Market event stream degraded");
    marketSocket.onclose = () => setStatus("Market event stream idle");

    return () => {
      sockets.forEach((socket) => socket.close());
      marketSocket.close();
    };
  }, [market.market_id, streams]);

  return (
    <section className={`panel ${styles.panel}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">Live feeds</span>
          <p className={styles.copy}>This panel only shows what the local `ws` service has actually produced. If there is no quote or depth yet, the surface stays empty on purpose.</p>
        </div>
        <span className="pill">{status}</span>
      </div>

      <div className={styles.grid}>
        {(["YES", "NO"] as const).map((outcome) => {
          const side = sides[outcome];
          return (
            <article key={outcome} className={styles.card}>
              <div className={styles.title}>
                <h3>{outcome} book</h3>
                <span className="pill">{side.ticker?.book_key ?? `${market.market_id}:${outcome}`}</span>
              </div>
              <div className={styles.stats}>
                <div className={styles.stat}>
                  <span className={styles.label}>Last</span>
                  <strong>{side.ticker ? `${side.ticker.last_price}¢` : "—"}</strong>
                </div>
                <div className={styles.stat}>
                  <span className={styles.label}>Bid</span>
                  <strong>{side.ticker?.best_bid ? `${side.ticker.best_bid}¢` : "—"}</strong>
                </div>
                <div className={styles.stat}>
                  <span className={styles.label}>Ask</span>
                  <strong>{side.ticker?.best_ask ? `${side.ticker.best_ask}¢` : "—"}</strong>
                </div>
              </div>

              <div className={styles.candleWrap}>
                <div className={styles.candleHead}>
                  <span className={styles.label}>1m candle tape</span>
                  {side.candles.length > 0 ? (
                    <span className={styles.candleMeta}>
                      O {side.candles[side.candles.length - 1].open} · H {side.candles[side.candles.length - 1].high} · L {side.candles[side.candles.length - 1].low} · C {side.candles[side.candles.length - 1].close}
                    </span>
                  ) : (
                    <span className={styles.candleMeta}>waiting for prints</span>
                  )}
                </div>
                {(() => {
                  const geometry = renderCandlePath(side.candles);
                  if (!geometry) {
                    return <div className={styles.candleEmpty}>No trades yet on this side.</div>;
                  }
                  return (
                    <svg viewBox={`0 0 ${geometry.chartWidth} ${geometry.chartHeight}`} className={styles.candleChart} role="img" aria-label={`${outcome} candle chart`}>
                      {side.candles.map((candle, index) => {
                        const centerX = geometry.slotWidth * index + geometry.slotWidth / 2;
                        const wickTop = geometry.priceToY(candle.high);
                        const wickBottom = geometry.priceToY(candle.low);
                        const openY = geometry.priceToY(candle.open);
                        const closeY = geometry.priceToY(candle.close);
                        const bodyY = Math.min(openY, closeY);
                        const bodyHeight = Math.max(Math.abs(closeY - openY), 2);
                        const rising = candle.close >= candle.open;
                        const className = rising ? styles.candleUp : styles.candleDown;

                        return (
                          <g key={`${outcome}-${candle.bucket_start_millis}`} className={className}>
                            <line x1={centerX} x2={centerX} y1={wickTop} y2={wickBottom} className={styles.wick} />
                            <rect
                              x={centerX - geometry.bodyWidth / 2}
                              y={bodyY}
                              width={geometry.bodyWidth}
                              height={bodyHeight}
                              rx={2}
                              className={styles.body}
                            />
                          </g>
                        );
                      })}
                    </svg>
                  );
                })()}
              </div>

              <div className={styles.ladder}>
                {side.depth && [...side.depth.bids, ...side.depth.asks].length > 0 ? (
                  [...side.depth.bids, ...side.depth.asks].slice(0, 6).map((level, index) => (
                    <div key={`${outcome}-${level.price}-${index}`} className={styles.row}>
                      <span>{level.price}¢</span>
                      <strong>{formatToken(level.quantity, 0)}</strong>
                    </div>
                  ))
                ) : (
                  <div className={styles.candleEmpty}>No live depth snapshot yet for this side.</div>
                )}
              </div>

              <div className={styles.meta}>
                <span>updated {formatTimestamp(Math.floor((side.ticker?.occurred_at_millis ?? 0) / 1000))}</span>
                <span>qty {formatToken(side.ticker?.last_quantity ?? 0, 0)}</span>
              </div>
            </article>
          );
        })}
      </div>

      <div className={styles.events}>
        {events.length > 0 ? (
          events.map((event, index) => (
            <div key={`${event.type}-${index}`} className={styles.event}>
              <div className={styles.eventHead}>{event.type}</div>
              <div className={styles.meta}>
                <span>{event.payload.status ?? event.payload.winning_outcome ?? event.payload.resolved_outcome ?? "state update"}</span>
                {event.payload.payout_amount ? <span>payout {formatToken(event.payload.payout_amount, 0)}</span> : null}
                <span>{formatTimestamp(Math.floor((event.payload.occurred_at_millis ?? 0) / 1000))}</span>
              </div>
            </div>
          ))
        ) : (
          <div className={styles.candleEmpty}>No market-event messages have been observed for this market in the current browser session.</div>
        )}
      </div>
    </section>
  );
}
