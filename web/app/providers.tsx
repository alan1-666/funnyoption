"use client";

import { TradingSessionProvider } from "@/components/trading-session-provider";

export function Providers({ children }: { children: React.ReactNode }) {
  return <TradingSessionProvider>{children}</TradingSessionProvider>;
}
