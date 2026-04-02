"use client";

import { useDeferredValue, useMemo, useState } from "react";

import { formatAssetAmount, formatTimestamp, formatToken } from "@/lib/format";
import { zhMarketStatus, zhOutcome } from "@/lib/locale";
import type { Market, Trade } from "@/lib/types";
import styles from "@/components/admin-read-board.module.css";

type TradeSideFilter = "ALL" | "BUY" | "SELL";
type TradeOutcomeFilter = "ALL" | "YES" | "NO";
type MarketStatusFilter = "ALL" | "OPEN" | "RESOLVED";

export function AdminReadBoard({ markets, trades }: { markets: Market[]; trades: Trade[] }) {
  const [tradeQuery, setTradeQuery] = useState("");
  const [tradeSide, setTradeSide] = useState<TradeSideFilter>("ALL");
  const [tradeOutcome, setTradeOutcome] = useState<TradeOutcomeFilter>("ALL");
  const [marketQuery, setMarketQuery] = useState("");
  const [marketStatus, setMarketStatus] = useState<MarketStatusFilter>("ALL");

  const deferredTradeQuery = useDeferredValue(tradeQuery.trim().toLowerCase());
  const deferredMarketQuery = useDeferredValue(marketQuery.trim().toLowerCase());

  const filteredTrades = useMemo(() => {
    return trades.filter((trade) => {
      if (tradeSide !== "ALL" && String(trade.taker_side).toUpperCase() !== tradeSide) {
        return false;
      }
      if (tradeOutcome !== "ALL" && String(trade.outcome).toUpperCase() !== tradeOutcome) {
        return false;
      }
      if (!deferredTradeQuery) {
        return true;
      }
      const haystack = [
        trade.trade_id,
        String(trade.market_id),
        String(trade.outcome),
        String(trade.taker_side),
        String(trade.maker_side),
        String(trade.sequence_no)
      ]
        .join(" ")
        .toLowerCase();
      return haystack.includes(deferredTradeQuery);
    });
  }, [deferredTradeQuery, tradeOutcome, tradeSide, trades]);

  const filteredMarkets = useMemo(() => {
    return markets.filter((market) => {
      if (marketStatus !== "ALL" && String(market.status).toUpperCase() !== marketStatus) {
        return false;
      }
      if (!deferredMarketQuery) {
        return true;
      }
      const haystack = [
        String(market.market_id),
        market.title,
        market.collateral_asset,
        market.resolved_outcome,
        market.category?.display_name ?? market.metadata?.category ?? ""
      ]
        .join(" ")
        .toLowerCase();
      return haystack.includes(deferredMarketQuery);
    });
  }, [deferredMarketQuery, marketStatus, markets]);

  return (
    <section className={styles.grid}>
      <div className={`panel ${styles.panel}`}>
        <div className={styles.header}>
          <div>
            <span className="eyebrow">成交记录</span>
            <h2 className={styles.title}>按关键词筛成交。</h2>
          </div>
          <span className="pill">结果 {filteredTrades.length}</span>
        </div>

        <div className={styles.filters}>
          <label className={styles.searchWrap}>
            <span className={styles.searchTag}>搜索</span>
            <input
              className={styles.search}
              value={tradeQuery}
              onChange={(event) => setTradeQuery(event.target.value)}
              placeholder="输入 trade_id、市场 ID、方向"
            />
          </label>
          <select className={styles.select} value={tradeSide} onChange={(event) => setTradeSide(event.target.value as TradeSideFilter)}>
            <option value="ALL">全部方向</option>
            <option value="BUY">买入</option>
            <option value="SELL">卖出</option>
          </select>
          <select className={styles.select} value={tradeOutcome} onChange={(event) => setTradeOutcome(event.target.value as TradeOutcomeFilter)}>
            <option value="ALL">全部结果</option>
            <option value="YES">是</option>
            <option value="NO">否</option>
          </select>
        </div>

        <div className={styles.list}>
          {filteredTrades.length > 0 ? (
            filteredTrades.map((trade) => (
              <div key={trade.trade_id} className={styles.row}>
                <div className={styles.primary}>
                  <strong>#{trade.market_id} · {zhOutcome(trade.outcome)}</strong>
                  <span>{trade.taker_side === "BUY" ? "买入" : "卖出"} · 序号 {trade.sequence_no}</span>
                </div>
                <div className={styles.secondary}>
                  <span>{formatToken(trade.quantity, 0)} 份</span>
                  <span>{trade.price}¢</span>
                  <span>{formatTimestamp(trade.occurred_at)}</span>
                </div>
              </div>
            ))
          ) : (
            <div className={styles.empty}>没有符合条件的成交记录。</div>
          )}
        </div>
      </div>

      <div className={`panel ${styles.panel}`}>
        <div className={styles.header}>
          <div>
            <span className="eyebrow">市场</span>
            <h2 className={styles.title}>按状态查看市场。</h2>
          </div>
          <span className="pill">结果 {filteredMarkets.length}</span>
        </div>

        <div className={styles.filters}>
          <label className={styles.searchWrap}>
            <span className={styles.searchTag}>搜索</span>
            <input
              className={styles.search}
              value={marketQuery}
              onChange={(event) => setMarketQuery(event.target.value)}
              placeholder="输入市场 ID、标题、分类"
            />
          </label>
          <select className={styles.select} value={marketStatus} onChange={(event) => setMarketStatus(event.target.value as MarketStatusFilter)}>
            <option value="ALL">全部状态</option>
            <option value="OPEN">交易中</option>
            <option value="RESOLVED">已结算</option>
          </select>
        </div>

        <div className={styles.list}>
          {filteredMarkets.length > 0 ? (
            filteredMarkets.map((market) => (
              <div key={market.market_id} className={styles.row}>
                <div className={styles.primary}>
                  <strong>#{market.market_id} · {market.title}</strong>
                  <span>
                    {market.category?.display_name ?? market.metadata?.category ?? "未分类"}
                    {" · "}
                    {zhMarketStatus(market.status)}
                    {market.resolved_outcome ? ` · ${zhOutcome(market.resolved_outcome)}` : ""}
                  </span>
                </div>
                <div className={styles.secondary}>
                  <span>{market.runtime.trade_count} 笔成交</span>
                  <span>{formatAssetAmount(market.runtime.matched_notional, "USDT")} USDT</span>
                  <span>{formatTimestamp(market.updated_at)}</span>
                </div>
              </div>
            ))
          ) : (
            <div className={styles.empty}>没有符合条件的市场。</div>
          )}
        </div>
      </div>
    </section>
  );
}
