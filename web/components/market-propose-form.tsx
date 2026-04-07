"use client";

import { useCallback, useState } from "react";
import { createPortal } from "react-dom";

import { proposeMarket } from "@/lib/api";
import styles from "@/components/shell-top-bar.module.css";

interface OptionEntry {
  label: string;
  shortLabel: string;
}

const DEFAULT_OPTIONS: OptionEntry[] = [
  { label: "是", shortLabel: "是" },
  { label: "否", shortLabel: "否" },
];

export function MarketProposeForm({ onClose }: { onClose: () => void }) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [resolutionSource, setResolutionSource] = useState("");
  const [closeAt, setCloseAt] = useState("");
  const [resolveAt, setResolveAt] = useState("");
  const [options, setOptions] = useState<OptionEntry[]>(() =>
    DEFAULT_OPTIONS.map((o) => ({ ...o }))
  );
  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateOption = useCallback(
    (idx: number, field: keyof OptionEntry, value: string) => {
      setOptions((prev) => {
        const next = [...prev];
        next[idx] = { ...next[idx], [field]: value };
        return next;
      });
    },
    []
  );

  const addOption = useCallback(() => {
    setOptions((prev) => {
      if (prev.length >= 6) return prev;
      return [...prev, { label: "", shortLabel: "" }];
    });
  }, []);

  const removeOption = useCallback((idx: number) => {
    setOptions((prev) => {
      if (prev.length <= 2) return prev;
      return prev.filter((_, i) => i !== idx);
    });
  }, []);

  const validOptions = options.filter((o) => o.label.trim().length > 0);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (validOptions.length < 2) {
      setError("At least 2 options are required");
      return;
    }

    setSubmitting(true);
    try {
      await proposeMarket({
        title: title.trim(),
        description: description.trim() || undefined,
        resolution_source: resolutionSource.trim() || undefined,
        close_at: closeAt ? Math.floor(new Date(closeAt).getTime() / 1000) : undefined,
        resolve_at: resolveAt ? Math.floor(new Date(resolveAt).getTime() / 1000) : undefined,
        options: validOptions.map((o) => ({
          label: o.label.trim(),
          short_label: o.shortLabel.trim() || undefined,
        })),
      });
      setSuccess(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Submission failed");
    } finally {
      setSubmitting(false);
    }
  }

  return createPortal(
    <div className={styles.proposeOverlay} onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className={styles.proposeModal}>
        <h2>Propose a Market</h2>

        {success ? (
          <div className={styles.formSuccess}>
            <p>Your market proposal has been submitted!</p>
            <p style={{ color: "rgba(255,255,255,0.4)", marginTop: 8, fontSize: "0.82rem" }}>
              An operator will review and approve it.
            </p>
            <div className={styles.formActions} style={{ justifyContent: "center", marginTop: 20 }}>
              <button className={styles.btnSecondary} onClick={onClose}>Close</button>
            </div>
          </div>
        ) : (
          <form onSubmit={handleSubmit}>
            <div className={styles.formGroup}>
              <label>Title *</label>
              <input
                className={styles.formInput}
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Will X happen by Y date?"
                required
                minLength={5}
                maxLength={200}
              />
            </div>

            <div className={styles.formGroup}>
              <label>Description</label>
              <textarea
                className={styles.formTextarea}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Provide context and details about this market..."
                rows={3}
              />
            </div>

            <div className={styles.formGroup}>
              <label>Resolution Criteria</label>
              <textarea
                className={styles.formTextarea}
                value={resolutionSource}
                onChange={(e) => setResolutionSource(e.target.value)}
                placeholder="How should this market be resolved? What source of truth will be used?"
                rows={2}
              />
            </div>

            <div className={styles.formGroup}>
              <label>Options *</label>
              <div className={styles.optionsList}>
                {options.map((opt, idx) => (
                  <div key={idx} className={styles.optionRow}>
                    <input
                      className={styles.formInput}
                      value={opt.label}
                      onChange={(e) => updateOption(idx, "label", e.target.value)}
                      placeholder={`Option ${idx + 1}`}
                      maxLength={64}
                      style={{ flex: 1 }}
                    />
                    <input
                      className={styles.formInput}
                      value={opt.shortLabel}
                      onChange={(e) => updateOption(idx, "shortLabel", e.target.value)}
                      placeholder="Short"
                      maxLength={16}
                      style={{ width: 72 }}
                    />
                    <button
                      type="button"
                      className={styles.optionRemove}
                      onClick={() => removeOption(idx)}
                      disabled={options.length <= 2}
                      aria-label="Remove option"
                    >
                      ×
                    </button>
                  </div>
                ))}
              </div>
              {options.length < 6 && (
                <button type="button" className={styles.optionAdd} onClick={addOption}>
                  + Add option
                </button>
              )}
            </div>

            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>
              <div className={styles.formGroup}>
                <label>Close Time</label>
                <input
                  className={styles.formInput}
                  type="datetime-local"
                  value={closeAt}
                  onChange={(e) => setCloseAt(e.target.value)}
                />
              </div>
              <div className={styles.formGroup}>
                <label>Resolve Time</label>
                <input
                  className={styles.formInput}
                  type="datetime-local"
                  value={resolveAt}
                  onChange={(e) => setResolveAt(e.target.value)}
                />
              </div>
            </div>

            {error && (
              <p style={{ color: "#ff6b6b", fontSize: "0.82rem", margin: "0 0 12px" }}>{error}</p>
            )}

            <div className={styles.formActions}>
              <button type="button" className={styles.btnSecondary} onClick={onClose}>
                Cancel
              </button>
              <button
                type="submit"
                className={styles.btnPrimary}
                disabled={submitting || title.trim().length < 5 || validOptions.length < 2}
              >
                {submitting ? "Submitting..." : "Submit Proposal"}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>,
    document.body
  );
}
