"use client";

import { useTicker } from "@/hooks/use-ticker";

interface Props {
  marketId: number;
  outcome: "YES" | "NO";
  fallback: number;
}

export function LiveOutcomePrice({ marketId, outcome, fallback }: Props) {
  const ticker = useTicker(marketId);
  const snap = outcome === "YES" ? ticker.yes : ticker.no;

  let price = fallback;
  if (snap) {
    if (snap.lastPrice > 0) price = snap.lastPrice;
    else if (snap.bestBid > 0 && snap.bestAsk > 0) price = Math.round((snap.bestBid + snap.bestAsk) / 2);
    else if (snap.bestBid > 0) price = snap.bestBid;
    else if (snap.bestAsk > 0) price = snap.bestAsk;
  }

  return <>{price}%</>;
}
