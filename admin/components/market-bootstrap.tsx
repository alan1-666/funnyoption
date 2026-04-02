"use client";

import { useEffect, useMemo, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";

import styles from "@/components/market-bootstrap.module.css";
import { useOperatorAccess } from "@/components/operator-access-provider";
import { formatTimestamp, formatToken, shortenAddress } from "@/lib/format";
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
  const [status, setStatus] = useState("Issue paired YES/NO inventory, then queue the first resting sell order from the same wallet-gated admin lane.");
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
      setStatus("Select a valid open market before bootstrapping liquidity.");
      return;
    }
    if (!Number.isFinite(userId) || userId <= 0) {
      setStatus("Bootstrap user id must be positive.");
      return;
    }
    if (!Number.isFinite(price) || price <= 0) {
      setStatus("Bootstrap price must be positive.");
      return;
    }
    if (!Number.isFinite(quantity) || quantity <= 0) {
      setStatus("Bootstrap quantity must be positive.");
      return;
    }

    setBusy(true);
    setStatus(`Requesting operator wallet signature for market #${marketId} bootstrap...`);

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
        setStatus("Connect an allowlisted wallet before issuing first-liquidity.");
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
        `Issued paired inventory ${payload?.first_liquidity_id ?? "unknown"} and queued sell order ${payload?.order_id ?? "unknown"} for market #${marketId} as ${shortenAddress(payload?.operator_wallet_address ?? operator.walletAddress)}.`
      );
      router.refresh();
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Failed to bootstrap first-liquidity");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className={`panel ${styles.desk}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">Bootstrap lane</span>
          <h2 className={styles.title}>Issue explicit first-liquidity from the same operator wallet lane.</h2>
          <p className={styles.copy}>
            This replaces the old ungated Go/template bootstrap path. The dedicated admin runtime now signs the paired-inventory issuance and first sell-order submit behind the same allowlist enforced for create and resolve.
          </p>
        </div>
        <div className={styles.badges}>
          <span className="pill">{openMarkets.length} open</span>
          <span className="pill">{wallet ? shortenAddress(wallet.walletAddress) : "Wallet required"}</span>
        </div>
      </div>

      <div className={styles.gateNote}>
        First-liquidity is wallet-gated at the admin boundary. Requests from non-allowlisted wallets are denied before paired inventory or bootstrap orders reach the shared API.
      </div>

      <div className={styles.grid}>
        <form className={styles.form} onSubmit={handleSubmit}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="bootstrap-market">
              Open market
            </label>
            <select
              id="bootstrap-market"
              name="market_id"
              className={styles.select}
              value={form.marketId}
              onChange={(event) => patchForm({ marketId: event.target.value })}
              disabled={openMarkets.length === 0}
            >
              {openMarkets.length === 0 ? <option value="">Create a market first</option> : null}
              {openMarkets.map((market) => (
                <option key={market.market_id} value={market.market_id}>
                  #{market.market_id} {market.title}
                </option>
              ))}
            </select>
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="bootstrap-user">
              Bootstrap user id
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
            <span className={styles.label}>Sell outcome</span>
            <div className={styles.toggleGroup}>
              {(["YES", "NO"] as const).map((outcome) => (
                <button
                  key={outcome}
                  type="button"
                  className={form.outcome === outcome ? styles.toggleActive : styles.toggle}
                  onClick={() => patchForm({ outcome })}
                >
                  {outcome}
                </button>
              ))}
            </div>
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="bootstrap-price">
              Sell price (cents)
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
              Paired quantity
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
            {busy ? "Bootstrapping..." : "Issue First-Liquidity"}
          </button>
        </form>

        <aside className={styles.side}>
          {selectedMarket ? (
            <div className={styles.selectedCard}>
              <span className="eyebrow">Selected market</span>
              <h3 className={styles.marketTitle}>#{selectedMarket.market_id} {selectedMarket.title}</h3>
              <p className={styles.marketCopy}>{selectedMarket.description || "No market description provided."}</p>
              <div className={styles.marketMeta}>
                <span>Close {formatTimestamp(selectedMarket.close_at)}</span>
                <span>Resolve {formatTimestamp(selectedMarket.resolve_at)}</span>
                <span>{selectedMarket.runtime.trade_count} trades</span>
                <span>{formatToken(selectedMarket.runtime.matched_notional / 100, 0)} USDT matched</span>
              </div>
            </div>
          ) : (
            <div className={styles.empty}>No open market is ready for bootstrap yet. Publish one in the market-intake lane first.</div>
          )}

          <div className={styles.marketList}>
            {openMarkets.slice(0, 4).map((market) => (
              <button
                key={market.market_id}
                type="button"
                className={String(market.market_id) === form.marketId ? styles.marketCardActive : styles.marketCard}
                onClick={() => patchForm({ marketId: String(market.market_id) })}
              >
                <strong>#{market.market_id} {market.title}</strong>
                <span>{market.runtime.active_order_count} active order(s)</span>
                <span>Updated {formatTimestamp(market.updated_at)}</span>
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
