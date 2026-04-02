import styles from "@/app/admin/page.module.css";

const ADMIN_BASE_URL = process.env.NEXT_PUBLIC_ADMIN_BASE_URL ?? "http://127.0.0.1:3001";

export default function AdminPage() {
  return (
    <main className={`page-shell ${styles.pageShell}`}>
      <section className={styles.hero}>
        <div className={`panel ${styles.heroPrimary} float-in`}>
          <span className="eyebrow">Admin moved</span>
          <h1 className={styles.title}>Operator tooling now lives in a dedicated admin service.</h1>
          <p className={styles.heroCopy}>The public web shell no longer serves as the long-term home for market creation, first-liquidity bootstrap, or resolution controls.</p>
          <a href={ADMIN_BASE_URL} className={styles.linkCard}>
            Open Dedicated Admin Service
          </a>
        </div>

        <div className={`panel ${styles.heroSide} float-in float-in-delay-1`}>
          <span className="eyebrow">Service boundary</span>
          <h2 className={styles.sideTitle}>Privileged actions now run behind a separate runtime.</h2>
          <p className={styles.sideCopy}>Use the dedicated admin service for wallet-gated operator access, explicit operator identity, and the create/bootstrap/resolve API lane. Keep this route as a migration pointer only.</p>
          <code className={styles.command}>{ADMIN_BASE_URL}</code>
          <p className={styles.sideMeta}>If you started local dev with `scripts/dev-up.sh`, the admin runtime is already listening at the address above.</p>
        </div>
      </section>
    </main>
  );
}
