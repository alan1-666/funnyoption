"use client";

import { ShellTopBar } from "@/components/shell-top-bar";
import { VaultConsole } from "@/components/vault-console";

export function DepositShell() {
  return (
    <>
      <ShellTopBar searchPlaceholder="资金管理" />
      <VaultConsole />
    </>
  );
}
