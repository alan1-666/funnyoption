import { HomeMarketBoard } from "@/components/home-market-board";
import { getMarketsRead, getTradesRead } from "@/lib/api";

export default async function HomePage() {
  const chainName = process.env.NEXT_PUBLIC_CHAIN_NAME ?? "本地链";
  const [marketsResult, tradesResult] = await Promise.all([getMarketsRead(), getTradesRead()]);

  return (
    <main className="page-shell">
      <HomeMarketBoard
        markets={marketsResult.items}
        trades={tradesResult.items}
        chainName={chainName}
        marketsUnavailable={marketsResult.state === "unavailable"}
        tradesUnavailable={tradesResult.state === "unavailable"}
        marketsError={marketsResult.error?.message}
        tradesError={tradesResult.error?.message}
      />
    </main>
  );
}
