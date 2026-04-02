import { SiteHeader } from "@/components/site-header";
import styles from "@/app/control/page.module.css";

export default async function ControlPage() {
  return (
    <main className="page-shell">
      <SiteHeader />

      <section className={styles.hero}>
        <div className={`panel ${styles.heroMain} float-in`}>
          <span className="eyebrow">Admin Only</span>
          <h1 className={styles.title}>Operational tools are not exposed in the public app.</h1>
          <p className={styles.copy}>Market creation, operational dashboards, queue inspection, and configuration management now live under `/admin` instead of the public product flow.</p>
        </div>
      </section>
    </main>
  );
}
