"use client";

import { ShellTopBar } from "@/components/shell-top-bar";
import { VaultConsole } from "@/components/vault-console";

export function DepositShell() {
  return (
    <>
      <ShellTopBar searchPlaceholder="充值与提现" />
      <VaultConsole />
    </>
  );
}
