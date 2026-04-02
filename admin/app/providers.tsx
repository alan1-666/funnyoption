"use client";

import { OperatorAccessProvider } from "@/components/operator-access-provider";

export function Providers({ children }: { children: React.ReactNode }) {
  return <OperatorAccessProvider>{children}</OperatorAccessProvider>;
}
