"use client";

import styles from "@/components/operator-access-card.module.css";
import { shortenAddress } from "@/lib/format";
import { useOperatorAccess } from "@/components/operator-access-provider";

export function OperatorAccessCard() {
  const { wallet, busy, statusMessage, allowlistedWallets, isWalletAllowlisted, connect } = useOperatorAccess();
  const hasAllowlist = allowlistedWallets.length > 0;

  return (
    <section className={`panel ${styles.panel}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">Operator identity</span>
          <h2 className={styles.title}>Connect an allowlisted wallet before creating, bootstrapping, or resolving markets.</h2>
          <p className={styles.copy}>This service signs each operator action with the connected wallet, exposes the active identity in the UI, and denies requests that do not come from the configured operator wallet set.</p>
        </div>
        <div className={styles.badges}>
          <span className="pill">{hasAllowlist ? `${allowlistedWallets.length} wallet(s) allowlisted` : "No allowlist configured"}</span>
          <span className="pill">{wallet ? `Chain ${wallet.chainId}` : "Wallet disconnected"}</span>
        </div>
      </div>

      <div className={styles.grid}>
        <div className={styles.identityCard}>
          <span className={styles.label}>Connected wallet</span>
          <strong className={styles.value}>{wallet ? shortenAddress(wallet.walletAddress) : "Not connected"}</strong>
          <p className={styles.meta}>{wallet ? wallet.walletAddress : "Connect the operator wallet to unlock admin actions."}</p>
        </div>

        <div className={styles.identityCard}>
          <span className={styles.label}>Access state</span>
          <strong className={isWalletAllowlisted ? styles.valueOk : styles.valueDanger}>
            {!hasAllowlist ? "Blocked" : wallet && isWalletAllowlisted ? "Allowed" : "Denied"}
          </strong>
          <p className={styles.meta}>
            {!hasAllowlist
              ? "Set FUNNYOPTION_OPERATOR_WALLETS before using the admin runtime."
              : wallet && isWalletAllowlisted
                ? "This wallet can sign create, first-liquidity, and resolve actions in the dedicated admin service."
                : "Connected wallets outside the allowlist can inspect reads but cannot run privileged actions."}
          </p>
        </div>
      </div>

      <div className={styles.actions}>
        <button className={styles.primary} type="button" disabled={busy === "connect"} onClick={() => connect()}>
          {busy === "connect" ? "Connecting..." : wallet ? "Reconnect Wallet" : "Connect Wallet"}
        </button>
        <div className={styles.status} aria-live="polite">
          {statusMessage}
        </div>
      </div>

      {allowlistedWallets.length > 0 ? (
        <div className={styles.allowlist}>
          {allowlistedWallets.map((entry) => (
            <span key={entry} className="pill">{shortenAddress(entry)}</span>
          ))}
        </div>
      ) : null}
    </section>
  );
}
