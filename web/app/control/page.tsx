import { SiteHeader } from "@/components/site-header";
import styles from "@/app/control/page.module.css";

export default async function ControlPage() {
  return (
    <main className="page-shell">
      <SiteHeader />

      <section className={styles.hero}>
        <div className={`panel ${styles.heroMain} float-in`}>
          <span className="eyebrow">仅后台可用</span>
          <h1 className={styles.title}>运营工具不会出现在用户端。</h1>
          <p className={styles.copy}>市场创建、运营看板、队列检查和配置管理都已经迁移到 `/admin`，不再出现在公开页面里。</p>
        </div>
      </section>
    </main>
  );
}
