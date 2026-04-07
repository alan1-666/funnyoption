"use client";

import { useState } from "react";
import { createPortal } from "react-dom";

import { proposeMarket } from "@/lib/api";
import styles from "@/components/shell-top-bar.module.css";

export function MarketProposeForm({ onClose }: { onClose: () => void }) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [resolutionSource, setResolutionSource] = useState("");
  const [closeAt, setCloseAt] = useState("");
  const [resolveAt, setResolveAt] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    try {
      await proposeMarket({
        title: title.trim(),
        description: description.trim() || undefined,
        resolution_source: resolutionSource.trim() || undefined,
        close_at: closeAt ? Math.floor(new Date(closeAt).getTime() / 1000) : undefined,
        resolve_at: resolveAt ? Math.floor(new Date(resolveAt).getTime() / 1000) : undefined,
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
                disabled={submitting || title.trim().length < 5}
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
