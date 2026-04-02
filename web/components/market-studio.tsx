"use client";

import { useMemo, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";

import type { Market } from "@/lib/types";
import styles from "@/components/market-studio.module.css";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";
const DEFAULT_USER_ID = Number(process.env.NEXT_PUBLIC_DEFAULT_USER_ID ?? "1001");

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
  createdBy: string;
  status: string;
  collateralAsset: string;
  openAt: string;
  closeAt: string;
  resolveAt: string;
};

interface MarketStudioProps {
  existingMarkets: Market[];
  stayOnPage?: boolean;
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
    createdBy: String(DEFAULT_USER_ID),
    status: "OPEN",
    collateralAsset: "USDT",
    openAt: toDateTimeLocal(now),
    closeAt: toDateTimeLocal(now + 7 * 24 * 60 * 60),
    resolveAt: toDateTimeLocal(now + 14 * 24 * 60 * 60)
  };
}

export function MarketStudio({ existingMarkets, stayOnPage = false }: MarketStudioProps) {
  const router = useRouter();
  const [form, setForm] = useState<StudioForm>(defaultForm);
  const [status, setStatus] = useState("Ready to import a Polymarket question");
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
    setStatus("Importing from Polymarket…");
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
    setStatus("Creating market…");
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/markets`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          title: form.title,
          description: form.description,
          collateral_asset: form.collateralAsset,
          status: form.status,
          open_at: toEpochSeconds(form.openAt),
          close_at: toEpochSeconds(form.closeAt),
          resolve_at: toEpochSeconds(form.resolveAt),
          created_by: Number(form.createdBy || DEFAULT_USER_ID),
          cover_image_url: form.coverImage,
          cover_source_url: form.sourceUrl,
          cover_source_name: form.sourceName,
          metadata: {
            category: form.category,
            coverImage: form.coverImage,
            sourceUrl: form.sourceUrl,
            sourceSlug: form.sourceSlug,
            sourceName: form.sourceName,
            sourceKind: imported?.kind ?? "manual",
            yesOdds: 0.5,
            noOdds: 0.5
          }
        })
      });

      const payload = (await response.json().catch(() => null)) as { market_id?: number; error?: string } | null;
      if (!response.ok || !payload?.market_id) {
        throw new Error(payload?.error ?? `HTTP ${response.status}`);
      }

      setStatus(`Created market #${payload.market_id}`);
      if (!stayOnPage) {
        router.push(`/markets/${payload.market_id}`);
      }
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
            Pull a Polymarket slug or URL, preserve the public brief, then publish it into the local market catalog without hand-copying the admin payload.
          </p>
        </div>
        <div className={styles.badges}>
          <span className="pill">{featuredCount} open</span>
          <span className="pill">{existingMarkets.length} tracked</span>
        </div>
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
        <button className={styles.primary} type="submit" disabled={busy}>
          {busy ? "Importing…" : "Import From Polymarket"}
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
            <label className={styles.label} htmlFor="market-created-by">
              Created by
            </label>
            <input
              id="market-created-by"
              name="created_by"
              type="number"
              className={styles.input}
              value={form.createdBy}
              onChange={(event) => patchForm({ createdBy: event.target.value })}
              inputMode="numeric"
            />
          </div>
        </div>

        <div className={styles.grid2}>
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
        </div>

        <div className={styles.grid2}>
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
        </div>

        <button className={styles.publish} type="submit" disabled={busy}>
          {busy ? "Publishing…" : "Publish Market"}
        </button>
      </form>

      <div className={styles.status} aria-live="polite">
        {status}
      </div>
    </section>
  );
}
