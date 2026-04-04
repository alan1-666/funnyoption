import { PortfolioShell } from "@/components/portfolio-shell";
import { getMarketsRead } from "@/lib/api";
import styles from "@/app/portfolio/page.module.css";

export default async function PortfolioPage() {
  const marketsResult = await getMarketsRead();

  return (
    <main className={styles.shell}>
      <PortfolioShell
        balances={[]}
        positions={[]}
        orders={[]}
        payouts={[]}
        markets={marketsResult.items}
        marketsUnavailable={marketsResult.state === "unavailable"}
        marketsError={marketsResult.error?.message}
        profile={null}
      />
    </main>
  );
}
