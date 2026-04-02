"use client";

import Link from "next/link";

import styles from "@/components/site-header.module.css";
import { shortenAddress } from "@/lib/format";
import { useTradingSession } from "@/components/trading-session-provider";

const links = [
  { href: "/" as const, label: "Tape" },
  { href: "/portfolio" as const, label: "Portfolio" }
];

export function SiteHeader() {
  const { wallet, session, busy, statusMessage, connect, createSession, revokeCurrentSession } = useTradingSession();
  const sessionTag = session ? `${session.sessionId.slice(0, 12)}…` : null;
  const actionLabel =
    busy === "connect"
      ? "Connecting..."
      : busy === "session"
        ? "Authorizing..."
        : session
          ? "Trading Enabled"
          : wallet
            ? "Enable Trading"
            : "Connect Wallet";
  const statusLabel = session
    ? `Trading enabled · ${sessionTag} · ${statusMessage}`
    : wallet
      ? `Wallet connected · ${statusMessage}`
      : statusMessage;

  async function handleAction() {
    if (session) {
      return;
    }
    if (!wallet) {
      await connect();
      return;
    }
    await createSession(wallet);
  }

  return (
    <header className={`${styles.header} float-in`}>
      <Link href="/" className={styles.brand}>
        <span className={styles.mark}>∿</span>
        <span>FunnyOption</span>
      </Link>

      <nav className={styles.nav}>
        {links.map((item) => (
          <Link key={item.href} href={item.href} className={styles.link}>
            {item.label}
          </Link>
        ))}
      </nav>

      <div className={styles.actions}>
        <span className="pill">BSC Testnet</span>
        {wallet ? <span className={styles.identity}>{shortenAddress(wallet.walletAddress)}</span> : null}
        {session ? (
          <button className={styles.ghost} onClick={revokeCurrentSession}>
            Revoke Session
          </button>
        ) : null}
        <button className={styles.button} onClick={handleAction}>
          {actionLabel}
        </button>
      </div>

      <div className={styles.status}>{statusLabel}</div>
    </header>
  );
}
