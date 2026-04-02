"use client";

import { useMemo, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";

import styles from "@/components/market-studio.module.css";
import { useOperatorAccess } from "@/components/operator-access-provider";
import { shortenAddress } from "@/lib/format";
import type { Market } from "@/lib/types";

type ImportedMarket = {
  kind: "event" | "market";
  slug: string;
  title: string;
  description: string;
  category: string;
  coverImage: string;
  sourceUrl: string;
  sourceName: string;
};

type StudioForm = {
  title: string;
  description: string;
  category: string;
  coverImage: string;
  sourceUrl: string;
  sourceSlug: string;
  sourceName: string;
  sourceKind: string;
  status: string;
  collateralAsset: string;
  openAt: string;
  closeAt: string;
  resolveAt: string;
};

interface MarketStudioProps {
  existingMarkets: Market[];
}

function toDateTimeLocal(secondsFromEpoch: number) {
  return new Date(secondsFromEpoch * 1000).toISOString().slice(0, 16);
}

function toEpochSeconds(value: string) {
  if (!value.trim()) return 0;
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? 0 : Math.floor(parsed / 1000);
}

function defaultForm(): StudioForm {
  const now = Math.floor(Date.now() / 1000);
  return {
    title: "",
    description: "",
    category: "Polymarket",
    coverImage: "",
    sourceUrl: "",
    sourceSlug: "",
    sourceName: "Polymarket",
    sourceKind: "manual",
    status: "OPEN",
    collateralAsset: "USDT",
    openAt: toDateTimeLocal(now),
    closeAt: toDateTimeLocal(now + 7 * 24 * 60 * 60),
    resolveAt: toDateTimeLocal(now + 14 * 24 * 60 * 60)
  };
}

export function MarketStudio({ existingMarkets }: MarketStudioProps) {
  const router = useRouter();
  const { wallet, busy: operatorBusy, signCreateMarket } = useOperatorAccess();
  const [form, setForm] = useState<StudioForm>(defaultForm);
  const [status, setStatus] = useState("Import from Polymarket or draft a market manually");
  const [busy, setBusy] = useState(false);
  const [importInput, setImportInput] = useState("");
  const [imported, setImported] = useState<ImportedMarket | null>(null);

  const featuredCount = useMemo(() => existingMarkets.filter((market) => market.status === "OPEN").length, [existingMarkets]);

  function patchForm(patch: Partial<StudioForm>) {
    setForm((current) => ({ ...current, ...patch }));
  }

  async function handleImport(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!importInput.trim()) {
      setStatus("Paste a Polymarket URL or slug first.");
      return;
    }

    setBusy(true);
    setStatus("Importing from Polymarket...");
    try {
      const response = await fetch(`/api/polymarket?input=${encodeURIComponent(importInput.trim())}`, {
        cache: "no-store"
      });
      const payload = (await response.json().catch(() => null)) as (ImportedMarket & { error?: string }) | null;

      if (!response.ok || !payload || !payload.title) {
        throw new Error(payload?.error ?? `HTTP ${response.status}`);
      }

      setImported(payload);
      patchForm({
        title: payload.title,
        description: payload.description,
        category: payload.category,
        coverImage: payload.coverImage,
        sourceUrl: payload.sourceUrl,
        sourceSlug: payload.slug,
        sourceName: payload.sourceName,
        sourceKind: payload.kind,
        status: "OPEN"
      });
      setStatus(`Imported ${payload.kind} "${payload.title}"`);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Failed to import market");
    } finally {
      setBusy(false);
    }
  }

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!form.title.trim()) {
      setStatus("Title is required.");
      return;
    }

    setBusy(true);
    setStatus("Requesting operator wallet signature...");
    try {
      const market = {
        title: form.title,
        description: form.description,
        category: form.category,
        coverImage: form.coverImage,
        sourceUrl: form.sourceUrl,
        sourceSlug: form.sourceSlug,
        sourceName: form.sourceName,
        sourceKind: imported?.kind ?? form.sourceKind,
        status: form.status,
        collateralAsset: form.collateralAsset,
        openAt: toEpochSeconds(form.openAt),
        closeAt: toEpochSeconds(form.closeAt),
        resolveAt: toEpochSeconds(form.resolveAt)
      };
      const operator = await signCreateMarket(market);
      if (!operator) {
        setStatus("Connect an allowlisted wallet before publishing a market.");
        return;
      }

      const response = await fetch("/api/operator/markets", {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          market,
          operator
        })
      });

      const payload = (await response.json().catch(() => null)) as { market_id?: number; error?: string; operator_wallet_address?: string } | null;
      if (!response.ok || !payload?.market_id) {
        throw new Error(payload?.error ?? `HTTP ${response.status}`);
      }

      setStatus(`Created market #${payload.market_id} as ${shortenAddress(payload.operator_wallet_address ?? operator.walletAddress)}`);
      router.refresh();
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Failed to create market");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className={`panel ${styles.studio}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">Market intake</span>
          <h2 className={styles.title}>Import, review, and publish an operator-managed market.</h2>
          <p className={styles.copy}>
            This intake lane now belongs to the dedicated admin service. Publishing requires an allowlisted wallet signature instead of a public-web-only form submit.
          </p>
        </div>
        <div className={styles.badges}>
          <span className="pill">{featuredCount} open</span>
          <span className="pill">{existingMarkets.length} tracked</span>
          <span className="pill">{wallet ? shortenAddress(wallet.walletAddress) : "Wallet required"}</span>
        </div>
      </div>

      <div className={styles.accessNote}>
        Publishing is wallet-gated. The admin service signs the create intent with the connected operator wallet before proxying the request to the shared API.
      </div>

      <form className={styles.importBar} onSubmit={handleImport}>
        <div className={styles.field}>
          <label className={styles.label} htmlFor="polymarket-input">
            Polymarket URL or slug
          </label>
          <input
            id="polymarket-input"
            name="polymarket_input"
            className={styles.input}
            value={importInput}
            onChange={(event) => setImportInput(event.target.value)}
            placeholder="https://polymarket.com/event/..."
            autoComplete="off"
          />
        </div>
        <button className={styles.primary} type="submit" disabled={busy || operatorBusy === "sign"}>
          {busy ? "Importing..." : "Import From Polymarket"}
        </button>
      </form>

      {imported ? (
        <section className={styles.preview}>
          <div
            className={styles.previewImage}
            style={imported.coverImage ? { backgroundImage: `linear-gradient(180deg, rgba(9, 9, 11, 0.12), rgba(9, 9, 11, 0.82)), url(${encodeURI(imported.coverImage)})` } : undefined}
          >
            <div className={styles.previewPills}>
              <span className="pill">{imported.kind}</span>
              <span className="pill">{imported.category}</span>
            </div>
          </div>
          <div className={styles.previewCopy}>
            <h3>{imported.title}</h3>
            <p>{imported.description}</p>
            <div className={styles.previewMeta}>
              <span>{imported.sourceName}</span>
              <span>{imported.sourceUrl}</span>
            </div>
          </div>
        </section>
      ) : null}

      <form className={styles.form} onSubmit={handleCreate}>
        <div className={styles.field}>
          <label className={styles.label} htmlFor="market-title">
            Title
          </label>
          <input
            id="market-title"
            name="title"
            className={styles.input}
            value={form.title}
            onChange={(event) => patchForm({ title: event.target.value })}
            placeholder="Will BSC keep the rally going?"
            autoComplete="off"
          />
        </div>

        <div className={styles.field}>
          <label className={styles.label} htmlFor="market-description">
            Description
          </label>
          <textarea
            id="market-description"
            name="description"
            className={styles.textarea}
            value={form.description}
            onChange={(event) => patchForm({ description: event.target.value })}
            placeholder="Explain the settlement rule in one short paragraph."
            rows={4}
          />
        </div>

        <div className={styles.grid2}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="market-category">
              Category
            </label>
            <input
              id="market-category"
              name="category"
              className={styles.input}
              value={form.category}
              onChange={(event) => patchForm({ category: event.target.value })}
              placeholder="Macro / Crypto / Sports"
              autoComplete="off"
            />
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="market-cover">
              Cover image URL
            </label>
            <input
              id="market-cover"
              name="cover_image"
              className={styles.input}
              value={form.coverImage}
              onChange={(event) => patchForm({ coverImage: event.target.value })}
              placeholder="https://..."
              autoComplete="off"
              inputMode="url"
            />
          </div>
        </div>

        <div className={styles.grid2}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="market-open">
              Open at
            </label>
            <input
              id="market-open"
              name="open_at"
              type="datetime-local"
              className={styles.input}
              value={form.openAt}
              onChange={(event) => patchForm({ openAt: event.target.value })}
            />
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="market-close">
              Close at
            </label>
            <input
              id="market-close"
              name="close_at"
              type="datetime-local"
              className={styles.input}
              value={form.closeAt}
              onChange={(event) => patchForm({ closeAt: event.target.value })}
            />
          </div>
        </div>

        <div className={styles.grid2}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="market-resolve">
              Resolve at
            </label>
            <input
              id="market-resolve"
              name="resolve_at"
              type="datetime-local"
              className={styles.input}
              value={form.resolveAt}
              onChange={(event) => patchForm({ resolveAt: event.target.value })}
            />
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="market-status">
              Status
            </label>
            <input
              id="market-status"
              name="status"
              className={styles.input}
              value={form.status}
              onChange={(event) => patchForm({ status: event.target.value })}
              autoComplete="off"
            />
          </div>
        </div>

        <div className={styles.grid2}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="market-collateral">
              Collateral asset
            </label>
            <input
              id="market-collateral"
              name="collateral_asset"
              className={styles.input}
              value={form.collateralAsset}
              onChange={(event) => patchForm({ collateralAsset: event.target.value })}
              autoComplete="off"
            />
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="market-source-url">
              Source URL
            </label>
            <input
              id="market-source-url"
              name="source_url"
              className={styles.input}
              value={form.sourceUrl}
              onChange={(event) => patchForm({ sourceUrl: event.target.value })}
              autoComplete="off"
              inputMode="url"
            />
          </div>
        </div>

        <div className={styles.grid2}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="market-source-slug">
              Source slug
            </label>
            <input
              id="market-source-slug"
              name="source_slug"
              className={styles.input}
              value={form.sourceSlug}
              onChange={(event) => patchForm({ sourceSlug: event.target.value })}
              autoComplete="off"
            />
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="market-source-name">
              Source name
            </label>
            <input
              id="market-source-name"
              name="source_name"
              className={styles.input}
              value={form.sourceName}
              onChange={(event) => patchForm({ sourceName: event.target.value })}
              autoComplete="off"
            />
          </div>
        </div>

        <button className={styles.publish} type="submit" disabled={busy || operatorBusy === "connect" || operatorBusy === "sign"}>
          {busy ? "Publishing..." : "Publish Market"}
        </button>
      </form>

      <div className={styles.status} aria-live="polite">
        {status}
      </div>
    </section>
  );
}
