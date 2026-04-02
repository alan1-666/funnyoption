import { PortfolioShell } from "@/components/portfolio-shell";
import { getBalances, getMarkets, getOrders, getPayouts, getPositions, getProfile } from "@/lib/api";
import styles from "@/app/portfolio/page.module.css";

export default async function PortfolioPage() {
  const [balances, positions, orders, payouts, markets, profile] = await Promise.all([
    getBalances(),
    getPositions(),
    getOrders(),
    getPayouts(),
    getMarkets(),
    getProfile()
  ]);

  return (
    <main className={styles.shell}>
      <PortfolioShell
        balances={balances}
        positions={positions}
        orders={orders}
        payouts={payouts}
        markets={markets}
        profile={profile}
      />
    </main>
  );
}
