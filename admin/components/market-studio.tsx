"use client";

import { useMemo, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";

import styles from "@/components/market-studio.module.css";
import { useOperatorAccess } from "@/components/operator-access-provider";
import { shortenAddress } from "@/lib/format";
import { defaultBinaryOptions, normalizeCategoryKey, normalizeMarketOptions, type MarketOptionDraft } from "@/lib/operator-auth";
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

type OptionRow = {
  id: string;
  key: string;
  label: string;
  shortLabel: string;
  isActive: boolean;
};

type StudioForm = {
  title: string;
  description: string;
  categoryKey: string;
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
  options: OptionRow[];
};

interface MarketStudioProps {
  existingMarkets: Market[];
}

const CATEGORY_OPTIONS = [
  { key: "CRYPTO", label: "加密" },
  { key: "SPORTS", label: "体育" }
] as const;

const SPORTS_TRIPLE_OPTIONS: MarketOptionDraft[] = [
  { key: "HOME_WIN", label: "主胜", shortLabel: "主胜", sortOrder: 10, isActive: true },
  { key: "DRAW", label: "平", shortLabel: "平", sortOrder: 20, isActive: true },
  { key: "AWAY_WIN", label: "客胜", shortLabel: "客胜", sortOrder: 30, isActive: true }
];

function zhImportKind(kind: ImportedMarket["kind"]) {
  return kind === "event" ? "事件" : "市场";
}

function toDateTimeLocal(secondsFromEpoch: number) {
  return new Date(secondsFromEpoch * 1000).toISOString().slice(0, 16);
}

function toEpochSeconds(value: string) {
  if (!value.trim()) return 0;
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? 0 : Math.floor(parsed / 1000);
}

function createOptionRow(seed?: Partial<OptionRow>, index = 0): OptionRow {
  return {
    id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    key: seed?.key ?? `OPTION_${index + 1}`,
    label: seed?.label ?? "",
    shortLabel: seed?.shortLabel ?? "",
    isActive: seed?.isActive ?? true
  };
}

function rowsFromOptions(options: MarketOptionDraft[]) {
  return options.map((option, index) =>
    createOptionRow(
      {
        key: option.key,
        label: option.label,
        shortLabel: option.shortLabel ?? option.label,
        isActive: option.isActive
      },
      index
    )
  );
}

function buildDraftOptions(rows: OptionRow[]) {
  const compactRows = rows
    .map((row, index) => ({
      key: row.key.trim() || `OPTION_${index + 1}`,
      label: row.label.trim(),
      shortLabel: row.shortLabel.trim() || row.label.trim(),
      sortOrder: (index + 1) * 10,
      isActive: row.isActive
    }))
    .filter((row) => row.label);

  return normalizeMarketOptions(compactRows);
}

function optionsMatchPreset(left: MarketOptionDraft[], right: MarketOptionDraft[]) {
  const normalizedLeft = normalizeMarketOptions(left);
  const normalizedRight = normalizeMarketOptions(right);
  if (normalizedLeft.length !== normalizedRight.length) {
    return false;
  }
  return normalizedLeft.every((option, index) => {
    const candidate = normalizedRight[index];
    return (
      option.key === candidate.key &&
      option.label === candidate.label &&
      (option.shortLabel ?? option.label) === (candidate.shortLabel ?? candidate.label) &&
      option.isActive === candidate.isActive
    );
  });
}

function isBinaryOptions(options: MarketOptionDraft[]) {
  if (options.length !== 2) {
    return false;
  }
  const keys = options.map((option) => option.key).sort();
  return keys[0] === "NO" && keys[1] === "YES";
}

function defaultForm(): StudioForm {
  const now = Math.floor(Date.now() / 1000);
  return {
    title: "",
    description: "",
    categoryKey: "CRYPTO",
    coverImage: "",
    sourceUrl: "",
    sourceSlug: "",
    sourceName: "Polymarket",
    sourceKind: "manual",
    status: "OPEN",
    collateralAsset: "USDT",
    openAt: toDateTimeLocal(now),
    closeAt: toDateTimeLocal(now + 7 * 24 * 60 * 60),
    resolveAt: toDateTimeLocal(now + 14 * 24 * 60 * 60),
    options: rowsFromOptions(defaultBinaryOptions())
  };
}

export function MarketStudio({ existingMarkets }: MarketStudioProps) {
  const router = useRouter();
  const { wallet, busy: operatorBusy, signCreateMarket } = useOperatorAccess();
  const [form, setForm] = useState<StudioForm>(defaultForm);
  const [status, setStatus] = useState("支持导入 Polymarket，也可以直接用可视化表单配置分类和选项。");
  const [busy, setBusy] = useState(false);
  const [importInput, setImportInput] = useState("");
  const [imported, setImported] = useState<ImportedMarket | null>(null);

  const featuredCount = useMemo(() => existingMarkets.filter((market) => market.status === "OPEN").length, [existingMarkets]);
  const preparedOptions = useMemo(() => buildDraftOptions(form.options), [form.options]);
  const activeOptionCount = preparedOptions.filter((option) => option.isActive).length;
  const binaryReady = isBinaryOptions(preparedOptions);
  const categoryLabel = form.categoryKey === "SPORTS" ? "体育" : "加密";

  function patchForm(patch: Partial<StudioForm>) {
    setForm((current) => ({ ...current, ...patch }));
  }

  function handleCategoryChange(rawValue: string) {
    const nextCategory = normalizeCategoryKey(rawValue);
    const currentOptions = buildDraftOptions(form.options);
    let nextOptions = form.options;
    let nextStatus = form.status;
    let nextMessage = `已切换到${nextCategory === "SPORTS" ? "体育" : "加密"}分类。`;

    if (nextCategory === "SPORTS" && isBinaryOptions(currentOptions)) {
      nextOptions = rowsFromOptions(SPORTS_TRIPLE_OPTIONS);
      nextStatus = "DRAFT";
      nextMessage = "已切到体育分类，并自动套用三选模板；当前会先保留为 DRAFT。";
    } else if (nextCategory === "CRYPTO" && optionsMatchPreset(currentOptions, SPORTS_TRIPLE_OPTIONS)) {
      nextOptions = rowsFromOptions(defaultBinaryOptions());
      nextMessage = "已切到加密分类，并自动切回二元模板。";
    }

    setForm((current) => ({
      ...current,
      categoryKey: nextCategory,
      status: nextStatus,
      options: nextOptions
    }));
    setStatus(nextMessage);
  }

  function patchOption(id: string, patch: Partial<OptionRow>) {
    setForm((current) => ({
      ...current,
      options: current.options.map((option) => {
        if (option.id !== id) {
          return option;
        }
        const next = { ...option, ...patch };
        if (patch.label !== undefined && (!option.shortLabel.trim() || option.shortLabel.trim() === option.label.trim())) {
          next.shortLabel = patch.label;
        }
        return next;
      })
    }));
  }

  function moveOption(id: string, direction: -1 | 1) {
    setForm((current) => {
      const index = current.options.findIndex((option) => option.id === id);
      const target = index + direction;
      if (index < 0 || target < 0 || target >= current.options.length) {
        return current;
      }
      const nextOptions = [...current.options];
      const [item] = nextOptions.splice(index, 1);
      nextOptions.splice(target, 0, item);
      return {
        ...current,
        options: nextOptions
      };
    });
  }

  function addOption() {
    setForm((current) => ({
      ...current,
      options: [...current.options, createOptionRow(undefined, current.options.length)]
    }));
  }

  function removeOption(id: string) {
    setForm((current) => ({
      ...current,
      options: current.options.filter((option) => option.id !== id)
    }));
  }

  function applyPreset(options: MarketOptionDraft[], nextStatus?: string, nextMessage?: string) {
    patchForm({
      options: rowsFromOptions(options),
      ...(nextStatus ? { status: nextStatus } : {})
    });
    if (nextMessage) {
      setStatus(nextMessage);
    }
  }

  async function handleImport(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!importInput.trim()) {
      setStatus("请先输入 Polymarket 链接或 slug。");
      return;
    }

    setBusy(true);
    setStatus("正在导入 Polymarket 数据...");
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
        categoryKey: normalizeCategoryKey(payload.category),
        coverImage: payload.coverImage,
        sourceUrl: payload.sourceUrl,
        sourceSlug: payload.slug,
        sourceName: payload.sourceName,
        sourceKind: payload.kind,
        status: "OPEN",
        options: rowsFromOptions(defaultBinaryOptions())
      });
      setStatus(`已导入${zhImportKind(payload.kind)}「${payload.title}」，选项已切回二元模板。`);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "导入失败");
    } finally {
      setBusy(false);
    }
  }

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!form.title.trim()) {
      setStatus("请填写市场标题。");
      return;
    }
    if (preparedOptions.length < 2 || activeOptionCount < 2) {
      setStatus("至少需要两个启用中的选项。");
      return;
    }
    if (form.options.some((option) => option.label.trim() === "")) {
      setStatus("请补齐每个选项的名称，或者删除空白选项。");
      return;
    }
    if (form.status === "OPEN" && !binaryReady) {
      setStatus("当前交易引擎只支持二元市场直接进入 OPEN。多选项市场请先保存为 DRAFT。");
      return;
    }

    setBusy(true);
    setStatus("等待运营钱包签名...");
    try {
      const market = {
        title: form.title,
        description: form.description,
        categoryKey: form.categoryKey,
        coverImage: form.coverImage,
        sourceUrl: form.sourceUrl,
        sourceSlug: form.sourceSlug,
        sourceName: form.sourceName,
        sourceKind: imported?.kind ?? form.sourceKind,
        status: form.status,
        collateralAsset: form.collateralAsset,
        openAt: toEpochSeconds(form.openAt),
        closeAt: toEpochSeconds(form.closeAt),
        resolveAt: toEpochSeconds(form.resolveAt),
        options: preparedOptions
      };

      const operator = await signCreateMarket(market);
      if (!operator) {
        setStatus("请先连接白名单运营钱包。");
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

      setStatus(`市场 #${payload.market_id} 已创建，签名钱包 ${shortenAddress(payload.operator_wallet_address ?? operator.walletAddress)}`);
      router.refresh();
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "创建市场失败");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className={`panel ${styles.studio}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">市场创建</span>
          <h2 className={styles.title}>导入并发布新市场。</h2>
          <p className={styles.copy}>市场会写入正式分类表和选项表。日常发市场不需要再手填 JSON，直接在下面配置即可。</p>
        </div>
        <div className={styles.badges}>
          <span className="pill">交易中 {featuredCount}</span>
          <span className="pill">共 {existingMarkets.length} 个市场</span>
          <span className="pill">{wallet ? shortenAddress(wallet.walletAddress) : "需要连接钱包"}</span>
        </div>
      </div>

      <div className={styles.accessNote}>发布动作会先完成运营钱包签名，再写入共享 API。二元市场可以直接 OPEN；多选项市场当前请先保留在 DRAFT。</div>

      <form className={styles.importBar} onSubmit={handleImport}>
        <div className={styles.field}>
          <label className={styles.label} htmlFor="polymarket-input">
            Polymarket 链接或 slug
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
          {busy ? "导入中..." : "导入"}
        </button>
      </form>

      {imported ? (
        <section className={styles.preview}>
          <div
            className={styles.previewThumb}
            style={imported.coverImage ? { backgroundImage: `url(${encodeURI(imported.coverImage)})` } : undefined}
          />
          <div className={styles.previewBody}>
            <div className={styles.previewPills}>
              <span className="pill">{zhImportKind(imported.kind)}</span>
              <span className="pill">{categoryLabel}</span>
            </div>
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
        <div className={`${styles.field} ${styles.fieldWide}`}>
          <label className={styles.label} htmlFor="market-title">
            标题
          </label>
          <input
            id="market-title"
            name="title"
            className={styles.input}
            value={form.title}
            onChange={(event) => patchForm({ title: event.target.value })}
            placeholder="例如：BTC 今天会站上 100000 吗？"
            autoComplete="off"
          />
        </div>

        <div className={`${styles.field} ${styles.fieldWide}`}>
          <label className={styles.label} htmlFor="market-description">
            描述
          </label>
          <textarea
            id="market-description"
            name="description"
            className={styles.textarea}
            value={form.description}
            onChange={(event) => patchForm({ description: event.target.value })}
            placeholder="用一段短文说明结算规则。"
            rows={4}
          />
        </div>

        <div className={styles.field}>
          <label className={styles.label} htmlFor="market-category">
            分类
          </label>
          <select
            id="market-category"
            name="category_key"
            className={styles.input}
            value={form.categoryKey}
            onChange={(event) => handleCategoryChange(event.target.value)}
          >
            {CATEGORY_OPTIONS.map((category) => (
              <option key={category.key} value={category.key}>
                {category.label}
              </option>
            ))}
          </select>
        </div>

        <div className={styles.field}>
          <label className={styles.label} htmlFor="market-status">
            状态
          </label>
          <select
            id="market-status"
            name="status"
            className={styles.input}
            value={form.status}
            onChange={(event) => patchForm({ status: event.target.value })}
          >
            <option value="DRAFT">DRAFT</option>
            <option value="OPEN">OPEN</option>
          </select>
        </div>

        <div className={styles.field}>
          <label className={styles.label} htmlFor="market-cover">
            封面图链接
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

        <div className={styles.field}>
          <label className={styles.label} htmlFor="market-collateral">
            保证金币种
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
          <label className={styles.label} htmlFor="market-open">
            开始交易
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
            停止交易
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

        <div className={styles.field}>
          <label className={styles.label} htmlFor="market-resolve">
            结算时间
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
          <label className={styles.label} htmlFor="market-source-url">
            来源链接
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
            来源 slug
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
            来源名称
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

        <section className={`${styles.field} ${styles.optionField}`}>
          <div className={styles.optionHeader}>
            <div>
              <label className={styles.label}>选项配置</label>
              <p className={styles.optionCopy}>直接编辑选项列表即可。日常二元市场用模板一键填好，多选项市场再切到 DRAFT。</p>
            </div>
            <div className={styles.optionActions}>
              <button type="button" className={styles.secondary} onClick={() => applyPreset(defaultBinaryOptions(), undefined, "已切回二元模板，可直接作为 OPEN 市场发布。")}>
                二元模板
              </button>
              <button type="button" className={styles.secondary} onClick={() => applyPreset(SPORTS_TRIPLE_OPTIONS, "DRAFT", "已切换为体育三选模板，并自动改为 DRAFT。")}>
                体育三选
              </button>
              <button type="button" className={styles.secondary} onClick={addOption}>
                新增选项
              </button>
            </div>
          </div>

          <div className={styles.optionSummary}>
            <span className="pill">已配置 {preparedOptions.length} 个选项</span>
            <span className="pill">启用中 {activeOptionCount} 个</span>
            <span className={`pill ${form.status === "OPEN" && !binaryReady ? styles.warningPill : ""}`}>
              {binaryReady ? "当前可直接 OPEN" : "多选项请先 DRAFT"}
            </span>
          </div>

          <div className={styles.optionList}>
            {form.options.map((option, index) => (
              <article key={option.id} className={styles.optionRow}>
                <div className={styles.optionRowTop}>
                  <div className={styles.optionIndex}>选项 {index + 1}</div>
                  <div className={styles.optionRowActions}>
                    <button type="button" className={styles.rowAction} onClick={() => moveOption(option.id, -1)} disabled={index === 0}>
                      上移
                    </button>
                    <button type="button" className={styles.rowAction} onClick={() => moveOption(option.id, 1)} disabled={index === form.options.length - 1}>
                      下移
                    </button>
                    <button type="button" className={styles.rowDanger} onClick={() => removeOption(option.id)} disabled={form.options.length <= 2}>
                      删除
                    </button>
                  </div>
                </div>

                <div className={styles.optionGrid}>
                  <div className={styles.field}>
                    <label className={styles.label}>选项名称</label>
                    <input
                      className={styles.input}
                      value={option.label}
                      onChange={(event) => patchOption(option.id, { label: event.target.value })}
                      placeholder={index === 0 ? "例如：是" : "例如：主胜"}
                    />
                  </div>
                  <div className={styles.field}>
                    <label className={styles.label}>短标签</label>
                    <input
                      className={styles.input}
                      value={option.shortLabel}
                      onChange={(event) => patchOption(option.id, { shortLabel: event.target.value })}
                      placeholder="列表或盘口中显示的短词"
                    />
                  </div>
                  <div className={styles.field}>
                    <label className={styles.label}>选项键值</label>
                    <input
                      className={styles.input}
                      value={option.key}
                      onChange={(event) => patchOption(option.id, { key: event.target.value })}
                      placeholder={index < 2 ? (index === 0 ? "YES" : "NO") : `OPTION_${index + 1}`}
                    />
                  </div>
                  <label className={styles.toggleRow}>
                    <input
                      type="checkbox"
                      checked={option.isActive}
                      onChange={(event) => patchOption(option.id, { isActive: event.target.checked })}
                    />
                    <span>启用这个选项</span>
                  </label>
                </div>
              </article>
            ))}
          </div>
        </section>

        <button className={styles.publish} type="submit" disabled={busy || operatorBusy === "connect" || operatorBusy === "sign"}>
          {busy ? "发布中..." : "发布市场"}
        </button>
      </form>

      <div className={styles.status} aria-live="polite">
        {status}
      </div>
    </section>
  );
}
