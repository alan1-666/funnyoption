"use client";

import { useEffect, useMemo, useState } from "react";

import { createWsUrl } from "@/lib/ws";
import { formatAssetAmount, formatClockTimestamp, formatTimestamp, formatToken } from "@/lib/format";
import { presentMarketDescription } from "@/lib/market-display";
import { zhMarketStatus, zhOutcome, zhSide } from "@/lib/locale";
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
    payout_asset?: string;
    user_id?: number;
  };
}

interface SideState {
  ticker: QuoteTickerEvent | null;
  depth: QuoteDepthEvent | null;
  candles: QuoteCandle[];
}

type DetailTab = "results" | "activity" | "rules";
type RangeKey = "1H" | "6H" | "1D" | "ALL";

const RANGE_OPTIONS: Array<{ key: RangeKey; label: string }> = [
  { key: "1H", label: "1H" },
  { key: "6H", label: "6H" },
  { key: "1D", label: "1D" },
  { key: "ALL", label: "ALL" }
];

const RANGE_WINDOW: Record<Exclude<RangeKey, "ALL">, number> = {
  "1H": 60 * 60 * 1000,
  "6H": 6 * 60 * 60 * 1000,
  "1D": 24 * 60 * 60 * 1000
};

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
  return candles.slice(-48);
}

function readLivePrice(market: Market, side: SideState, outcome: "YES" | "NO") {
  if (side.ticker?.last_price && side.ticker.last_price > 0) {
    return side.ticker.last_price;
  }
  const metadata = market.metadata ?? {};
  if (market.status === "RESOLVED") {
    return market.resolved_outcome === outcome ? 100 : 0;
  }
  if (outcome === "YES") {
    if (market.runtime.last_price_yes > 0) {
      return market.runtime.last_price_yes;
    }
    if (typeof metadata.yesOdds === "number" && metadata.yesOdds > 0) {
      return Math.round(metadata.yesOdds * 100);
    }
    return 50;
  }
  if (market.runtime.last_price_no > 0) {
    return market.runtime.last_price_no;
  }
  if (typeof metadata.noOdds === "number" && metadata.noOdds > 0) {
    return Math.round(metadata.noOdds * 100);
  }
  return 50;
}

function filterCandles(candles: QuoteCandle[], range: RangeKey) {
  if (candles.length === 0 || range === "ALL") {
    return candles;
  }
  const anchor = candles[candles.length - 1].bucket_end_millis;
  const threshold = anchor - RANGE_WINDOW[range];
  const filtered = candles.filter((candle) => candle.bucket_end_millis >= threshold);
  return filtered.length > 1 ? filtered : candles;
}

function buildFallbackCandles(price: number, anchorSeconds: number): QuoteCandle[] {
  const safePrice = Math.max(0, Math.min(100, price));
  const anchorMillis = anchorSeconds > 0 ? anchorSeconds * 1000 : Date.now();
  return Array.from({ length: 4 }, (_, index) => {
    const bucketStart = anchorMillis - (3 - index) * 60 * 60 * 1000;
    return {
      bucket_start_millis: bucketStart,
      bucket_end_millis: bucketStart + 60 * 60 * 1000,
      open: safePrice,
      high: safePrice,
      low: safePrice,
      close: safePrice,
      volume: 0,
      trade_count: 0
    };
  });
}

function createLinePoints(candles: QuoteCandle[], timestamps: number[], width: number, height: number, padding: { top: number; right: number; bottom: number; left: number }) {
  if (timestamps.length === 0) {
    return "";
  }
  const usableWidth = width - padding.left - padding.right;
  const usableHeight = height - padding.top - padding.bottom;
  const indexByTimestamp = new Map(candles.map((candle) => [candle.bucket_start_millis, candle.close]));
  const firstValue = candles[0]?.close ?? 50;
  let lastValue = firstValue;

  return timestamps
    .map((timestamp, index) => {
      const value = indexByTimestamp.get(timestamp) ?? lastValue;
      lastValue = value;
      const x = padding.left + (timestamps.length === 1 ? 0 : (index / (timestamps.length - 1)) * usableWidth);
      const y = padding.top + ((100 - value) / 100) * usableHeight;
      return `${index === 0 ? "M" : "L"} ${x} ${y}`;
    })
    .join(" ");
}

function buildChartModel(yesCandles: QuoteCandle[], noCandles: QuoteCandle[]) {
  const timestamps = Array.from(
    new Set([...yesCandles.map((candle) => candle.bucket_start_millis), ...noCandles.map((candle) => candle.bucket_start_millis)])
  ).sort((left, right) => left - right);

  if (timestamps.length === 0) {
    return null;
  }

  const width = 940;
  const height = 360;
  const padding = { top: 22, right: 56, bottom: 36, left: 0 };
  const yesPath = createLinePoints(yesCandles, timestamps, width, height, padding);
  const noPath = createLinePoints(noCandles, timestamps, width, height, padding);
  const levels = [0, 25, 50, 75, 100];
  const labelIndexes = Array.from(new Set([0, Math.floor((timestamps.length - 1) / 3), Math.floor(((timestamps.length - 1) * 2) / 3), timestamps.length - 1]));
  const usableWidth = width - padding.left - padding.right;
  const usableHeight = height - padding.top - padding.bottom;

  return {
    width,
    height,
    padding,
    yesPath,
    noPath,
    levels: levels.map((level) => ({
      label: `${level}%`,
      y: padding.top + ((100 - level) / 100) * usableHeight
    })),
    timeLabels: labelIndexes.map((index) => ({
      x: padding.left + (timestamps.length === 1 ? 0 : (index / (timestamps.length - 1)) * usableWidth),
      label: formatClockTimestamp(timestamps[index])
    }))
  };
}

export function LiveMarketPanel({ market, trades }: { market: Market; trades: Trade[] }) {
  const [activeTab, setActiveTab] = useState<DetailTab>("results");
  const [activeRange, setActiveRange] = useState<RangeKey>("1D");
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
    market.status === "RESOLVED"
      ? "市场已结算，等待终态事件"
      : market.status === "WAITING_RESOLUTION"
        ? "市场等待裁决，实时行情已停止更新"
        : market.status === "CLOSED"
          ? "市场已收盘，等待结算结果"
          : "等待实时行情"
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
      socket.onopen = () => setStatus("实时连接中");
      socket.onmessage = (event) => {
        try {
          const payload = JSON.parse(event.data) as QuoteDepthEvent | QuoteTickerEvent | QuoteCandleEvent;
          const outcome = String(payload.outcome).toUpperCase() as "YES" | "NO";
          setSides((current) => ({
            ...current,
            [outcome]:
              stream.type === "ticker"
                ? { ...current[outcome], ticker: payload as QuoteTickerEvent }
                : stream.type === "depth"
                  ? { ...current[outcome], depth: payload as QuoteDepthEvent }
                  : { ...current[outcome], candles: (payload as QuoteCandleEvent).candles ?? current[outcome].candles }
          }));
        } catch {
          setStatus("行情流解析失败");
        }
      };
      socket.onerror = () => setStatus("行情流异常");
      socket.onclose = () => setStatus("行情流空闲");
      return socket;
    });

    const marketSocket = new WebSocket(createWsUrl(`/ws?stream=market&market_id=${market.market_id}`));
    marketSocket.onopen = () => setStatus("实时连接中");
    marketSocket.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data) as MarketEnvelope;
        setEvents((current) => [payload, ...current].slice(0, 8));
      } catch {
        setStatus("市场事件解析失败");
      }
    };
    marketSocket.onerror = () => setStatus("市场事件流异常");
    marketSocket.onclose = () => setStatus("市场事件流空闲");

    return () => {
      sockets.forEach((socket) => socket.close());
      marketSocket.close();
    };
  }, [market.market_id, streams]);

  const yesPrice = readLivePrice(market, sides.YES, "YES");
  const noPrice = readLivePrice(market, sides.NO, "NO");
  const yesCandles = filterCandles(sides.YES.candles, activeRange);
  const noCandles = filterCandles(sides.NO.candles, activeRange);
  const renderedYesCandles = yesCandles.length > 0 ? yesCandles : buildFallbackCandles(yesPrice, market.updated_at);
  const renderedNoCandles = noCandles.length > 0 ? noCandles : buildFallbackCandles(noPrice, market.updated_at);
  const chartModel = buildChartModel(renderedYesCandles, renderedNoCandles);
  const options = (market.options ?? [])
    .filter((option) => option.is_active !== false)
    .sort((left, right) => left.sort_order - right.sort_order || left.key.localeCompare(right.key));
  const recentTrades = [...trades].sort((left, right) => right.occurred_at - left.occurred_at).slice(0, 6);

  return (
    <section className={`panel ${styles.panel}`}>
      <div className={styles.header}>
        <div className={styles.headerIntro}>
          <span className="eyebrow">市场走势</span>
          <p className={styles.copy}>用一张大图表直接看 yes / no 的走势，下方再切结果、活动和规则，不再拆成很多说明块。</p>
        </div>
        <span className="pill">{status}</span>
      </div>

      <section className={styles.chartCard}>
        <div className={styles.marketStrip}>
          <div className={styles.legend}>
            <button className={`${styles.legendChip} ${styles.legendChipYes}`}>
              <span>{zhOutcome("YES")}</span>
              <strong>{yesPrice}%</strong>
            </button>
            <button className={styles.legendChip}>
              <span>{zhOutcome("NO")}</span>
              <strong>{noPrice}%</strong>
            </button>
          </div>
          <div className={styles.marketMeta}>
            <span>{formatAssetAmount(market.runtime.matched_notional, "USDT")} USDT</span>
            <span>{market.runtime.trade_count} 笔成交</span>
            <span>{zhMarketStatus(market.status)}</span>
          </div>
        </div>

        <div className={styles.chartFrame}>
          {chartModel ? (
            <>
              <svg viewBox={`0 0 ${chartModel.width} ${chartModel.height}`} className={styles.chart} role="img" aria-label="market chart">
                {chartModel.levels.map((level) => (
                  <g key={level.label}>
                    <line
                      x1={chartModel.padding.left}
                      x2={chartModel.width - chartModel.padding.right}
                      y1={level.y}
                      y2={level.y}
                      className={styles.gridLine}
                    />
                    <text x={chartModel.width - chartModel.padding.right + 12} y={level.y + 4} className={styles.axisLabel}>
                      {level.label}
                    </text>
                  </g>
                ))}
                <path d={chartModel.yesPath} className={styles.pathYes} />
                <path d={chartModel.noPath} className={styles.pathNo} />
              </svg>

              <div className={styles.timeAxis}>
                {chartModel.timeLabels.map((item) => (
                  <span key={`${item.label}-${item.x}`} style={{ left: `${(item.x / chartModel.width) * 100}%` }}>
                    {item.label}
                  </span>
                ))}
              </div>
            </>
          ) : (
            <div className={styles.emptyState}>当前市场还没有足够的数据来绘制价格走势。</div>
          )}
        </div>

        <div className={styles.rangeBar}>
          {RANGE_OPTIONS.map((range) => (
            <button
              key={range.key}
              className={activeRange === range.key ? styles.rangeActive : styles.rangeButton}
              onClick={() => setActiveRange(range.key)}
            >
              {range.label}
            </button>
          ))}
        </div>
      </section>

      <section className={styles.detailPanel}>
        <div className={styles.tabBar}>
          <button className={activeTab === "results" ? styles.tabActive : styles.tabButton} onClick={() => setActiveTab("results")}>
            结果
          </button>
          <button className={activeTab === "activity" ? styles.tabActive : styles.tabButton} onClick={() => setActiveTab("activity")}>
            活动
          </button>
          <button className={activeTab === "rules" ? styles.tabActive : styles.tabButton} onClick={() => setActiveTab("rules")}>
            规则
          </button>
        </div>

        {activeTab === "results" ? (
          <div className={styles.resultsList}>
            {options.length > 0 ? (
              options.map((option) => {
                const optionKey = option.key.toUpperCase();
                const livePrice = optionKey === "YES" ? yesPrice : optionKey === "NO" ? noPrice : null;
                const isResolved = market.status === "RESOLVED" && market.resolved_outcome?.toUpperCase() === optionKey;
                return (
                  <div key={option.key} className={styles.resultRow}>
                    <div>
                      <strong>{option.label}</strong>
                      <span>{isResolved ? "已命中结算结果" : "当前有效选项"}</span>
                    </div>
                    <div className={styles.resultMeta}>
                      <span>{livePrice !== null ? `${livePrice}%` : "待开放"}</span>
                      <strong>{isResolved ? "Resolved" : "Live"}</strong>
                    </div>
                  </div>
                );
              })
            ) : (
              <div className={styles.emptyState}>这个市场还没有配置选项。</div>
            )}
          </div>
        ) : null}

        {activeTab === "activity" ? (
          <div className={styles.activityList}>
            {recentTrades.map((trade) => (
              <div key={trade.trade_id} className={styles.activityRow}>
                <div>
                  <strong>{zhOutcome(trade.outcome)} · {zhSide(trade.taker_side)}</strong>
                  <span>{formatTimestamp(trade.occurred_at)}</span>
                </div>
                <div className={styles.activityMeta}>
                  <span>{trade.price}¢</span>
                  <strong>{formatToken(trade.quantity, 0)} 份</strong>
                </div>
              </div>
            ))}
            {events.map((event, index) => (
              <div key={`${event.type}-${index}`} className={styles.activityRow}>
                <div>
                  <strong>{event.type}</strong>
                  <span>{event.payload.status ?? event.payload.winning_outcome ?? event.payload.resolved_outcome ?? "状态更新"}</span>
                </div>
                <div className={styles.activityMeta}>
                  {event.payload.payout_amount ? (
                    <span>{formatAssetAmount(event.payload.payout_amount, String(event.payload.payout_asset ?? "USDT"))}</span>
                  ) : null}
                  <strong>{formatTimestamp(Math.floor((event.payload.occurred_at_millis ?? 0) / 1000))}</strong>
                </div>
              </div>
            ))}
            {recentTrades.length === 0 && events.length === 0 ? (
              <div className={styles.emptyState}>当前浏览器会话里还没有收到活动数据。</div>
            ) : null}
          </div>
        ) : null}

        {activeTab === "rules" ? (
          <div className={styles.rulesList}>
            <div className={styles.ruleCard}>
              <strong>开始交易</strong>
              <span>{formatTimestamp(market.open_at)}</span>
            </div>
            <div className={styles.ruleCard}>
              <strong>停止交易</strong>
              <span>{formatTimestamp(market.close_at)}</span>
            </div>
            <div className={styles.ruleCard}>
              <strong>结算时间</strong>
              <span>{formatTimestamp(market.resolve_at)}</span>
            </div>
            <div className={styles.ruleCard}>
              <strong>说明</strong>
              <span>{presentMarketDescription(market)}</span>
            </div>
          </div>
        ) : null}
      </section>
    </section>
  );
}
