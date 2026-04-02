"use client";

import { useEffect, useMemo, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";

import styles from "@/components/market-bootstrap.module.css";
import { useOperatorAccess } from "@/components/operator-access-provider";
import { formatAssetAmount, formatTimestamp, formatToken, shortenAddress } from "@/lib/format";
import { zhOutcome } from "@/lib/locale";
import type { Market } from "@/lib/types";

const DEFAULT_BOOTSTRAP_USER_ID = 1002;
const DEFAULT_BOOTSTRAP_PRICE = 58;
const DEFAULT_BOOTSTRAP_QUANTITY = 40;

type BootstrapForm = {
  marketId: string;
  userId: string;
  outcome: "YES" | "NO";
  price: string;
  quantity: string;
};

function defaultForm(marketId?: number): BootstrapForm {
  return {
    marketId: marketId ? String(marketId) : "",
    userId: String(DEFAULT_BOOTSTRAP_USER_ID),
    outcome: "YES",
    price: String(DEFAULT_BOOTSTRAP_PRICE),
    quantity: String(DEFAULT_BOOTSTRAP_QUANTITY)
  };
}

interface MarketBootstrapProps {
  markets: Market[];
}

export function MarketBootstrap({ markets }: MarketBootstrapProps) {
  const router = useRouter();
  const { wallet, busy: operatorBusy, signBootstrapMarket } = useOperatorAccess();
  const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState("先发首发仓位，再挂第一笔卖单。");
  const openMarkets = useMemo(
    () => markets.filter((market) => market.status === "OPEN").sort((left, right) => right.updated_at - left.updated_at),
    [markets]
  );
  const [form, setForm] = useState<BootstrapForm>(() => defaultForm(openMarkets[0]?.market_id));

  useEffect(() => {
    if (openMarkets.length === 0) {
      return;
    }
    if (openMarkets.some((market) => String(market.market_id) === form.marketId)) {
      return;
    }
    setForm((current) => ({
      ...current,
      marketId: String(openMarkets[0].market_id)
    }));
  }, [form.marketId, openMarkets]);

  const selectedMarket = openMarkets.find((market) => String(market.market_id) === form.marketId) ?? null;

  function patchForm(patch: Partial<BootstrapForm>) {
    setForm((current) => ({ ...current, ...patch }));
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const marketId = Number.parseInt(form.marketId, 10);
    const userId = Number.parseInt(form.userId, 10);
    const price = Number.parseInt(form.price, 10);
    const quantity = Number.parseInt(form.quantity, 10);

    if (!Number.isFinite(marketId) || marketId <= 0) {
      setStatus("请先选择一个有效的交易中市场。");
      return;
    }
    if (!Number.isFinite(userId) || userId <= 0) {
      setStatus("做市用户 ID 必须大于 0。");
      return;
    }
    if (!Number.isFinite(price) || price <= 0) {
      setStatus("首发价格必须大于 0。");
      return;
    }
    if (!Number.isFinite(quantity) || quantity <= 0) {
      setStatus("首发数量必须大于 0。");
      return;
    }

    setBusy(true);
    setStatus(`等待市场 #${marketId} 的首发签名...`);

    try {
      const bootstrap = {
        marketId,
        userId,
        outcome: form.outcome,
        price,
        quantity
      } as const;
      const operator = await signBootstrapMarket(bootstrap);
      if (!operator) {
        setStatus("请先连接白名单运营钱包。");
        return;
      }

      const response = await fetch(`/api/operator/markets/${marketId}/first-liquidity`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          bootstrap,
          operator
        })
      });

      const payload = (await response.json().catch(() => null)) as {
        error?: string;
        first_liquidity_id?: string;
        order_id?: string;
        operator_wallet_address?: string;
      } | null;
      if (!response.ok) {
        throw new Error(payload?.error ?? `HTTP ${response.status}`);
      }

      setStatus(
        `已发放首发仓位 ${payload?.first_liquidity_id ?? "未知"}，并创建卖单 ${payload?.order_id ?? "未知"}，签名钱包 ${shortenAddress(payload?.operator_wallet_address ?? operator.walletAddress)}。`
      );
      router.refresh();
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "首发流动性创建失败");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className={`panel ${styles.desk}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">首发流动性</span>
          <h2 className={styles.title}>发首发仓位并挂出第一笔单。</h2>
          <p className={styles.copy}>给做市账户发仓位后，后台会用同一个运营钱包挂出第一笔单，让前台可以直接进入撮合。</p>
        </div>
        <div className={styles.badges}>
          <span className="pill">交易中 {openMarkets.length}</span>
          <span className="pill">{wallet ? shortenAddress(wallet.walletAddress) : "需要连接钱包"}</span>
        </div>
      </div>

      <div className={styles.gateNote}>只有白名单运营钱包可以发放首发流动性，非白名单请求会在后台边界被直接拒绝。</div>

      <div className={styles.grid}>
        <form className={styles.form} onSubmit={handleSubmit}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="bootstrap-market">
              选择市场
            </label>
            <select
              id="bootstrap-market"
              name="market_id"
              className={styles.select}
              value={form.marketId}
              onChange={(event) => patchForm({ marketId: event.target.value })}
              disabled={openMarkets.length === 0}
            >
              {openMarkets.length === 0 ? <option value="">请先创建市场</option> : null}
              {openMarkets.map((market) => (
                <option key={market.market_id} value={market.market_id}>
                  #{market.market_id} · {market.title}
                </option>
              ))}
            </select>
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="bootstrap-user">
              做市用户 ID
            </label>
            <input
              id="bootstrap-user"
              name="user_id"
              className={styles.input}
              value={form.userId}
              onChange={(event) => patchForm({ userId: event.target.value })}
              inputMode="numeric"
              autoComplete="off"
            />
          </div>

          <div className={styles.field}>
            <span className={styles.label}>卖出方向</span>
            <div className={styles.toggleGroup}>
              {(["YES", "NO"] as const).map((outcome) => (
                <button
                  key={outcome}
                  type="button"
                  className={form.outcome === outcome ? styles.toggleActive : styles.toggle}
                  onClick={() => patchForm({ outcome })}
                >
                  {zhOutcome(outcome)}
                </button>
              ))}
            </div>
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="bootstrap-price">
              卖出价格（美分）
            </label>
            <input
              id="bootstrap-price"
              name="price"
              className={styles.input}
              value={form.price}
              onChange={(event) => patchForm({ price: event.target.value })}
              inputMode="numeric"
              autoComplete="off"
            />
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="bootstrap-quantity">
              首发数量
            </label>
            <input
              id="bootstrap-quantity"
              name="quantity"
              className={styles.input}
              value={form.quantity}
              onChange={(event) => patchForm({ quantity: event.target.value })}
              inputMode="numeric"
              autoComplete="off"
            />
          </div>

          <button
            className={styles.submit}
            type="submit"
            disabled={busy || operatorBusy === "connect" || operatorBusy === "sign" || openMarkets.length === 0}
          >
            {busy ? "提交中..." : "发放首发流动性"}
          </button>
        </form>

        <aside className={styles.side}>
          {selectedMarket ? (
            <div className={styles.selectedCard}>
              <span className="eyebrow">当前市场</span>
              <h3 className={styles.marketTitle}>#{selectedMarket.market_id} {selectedMarket.title}</h3>
              <div className={styles.marketMeta}>
                <span>停止交易 {formatTimestamp(selectedMarket.close_at)}</span>
                <span>结算时间 {formatTimestamp(selectedMarket.resolve_at)}</span>
                <span>{selectedMarket.runtime.trade_count} 笔成交</span>
                <span>成交额 {formatAssetAmount(selectedMarket.runtime.matched_notional, "USDT")} USDT</span>
              </div>
            </div>
          ) : (
            <div className={styles.empty}>还没有可用于首发流动性的市场，请先在上方创建市场。</div>
          )}

          <div className={styles.marketList}>
            {openMarkets.slice(0, 4).map((market) => (
              <button
                key={market.market_id}
                type="button"
                className={String(market.market_id) === form.marketId ? styles.marketCardActive : styles.marketCard}
                onClick={() => patchForm({ marketId: String(market.market_id) })}
              >
                <strong>#{market.market_id}</strong>
                <span className={styles.marketCardTitle}>{market.title}</span>
                <span>{market.runtime.active_order_count} 笔挂单 · {formatToken(market.runtime.matched_quantity, 0)} 已撮合</span>
              </button>
            ))}
          </div>
        </aside>
      </div>

      <div className={styles.status} aria-live="polite">
        {status}
      </div>
    </section>
  );
}
