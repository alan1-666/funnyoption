"use client";

import Link from "next/link";

import styles from "@/components/site-header.module.css";
import { shortenAddress } from "@/lib/format";
import { useTradingSession } from "@/components/trading-session-provider";
import { getChainMeta } from "@/lib/chain";

const links = [
  { href: "/" as const, label: "市场" },
  { href: "/portfolio" as const, label: "资产" }
];

export function SiteHeader() {
  const { wallet, session, busy, statusMessage, connect, createSession, revokeCurrentSession } = useTradingSession();
  const chain = getChainMeta();
  const sessionTag = session ? `${session.sessionId.slice(0, 12)}…` : null;
  const actionLabel =
    busy === "connect"
      ? "连接中..."
      : busy === "session"
        ? "授权中..."
        : session
          ? "已开启交易"
          : wallet
            ? "开启交易"
            : "连接钱包";
  const statusLabel = session
    ? `已开启交易 · ${sessionTag} · ${statusMessage}`
    : wallet
      ? `钱包已连接 · ${statusMessage}`
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
        <span className={styles.brandText}>
          <strong>FunnyOption</strong>
          <span>交易前台</span>
        </span>
      </Link>

      <nav className={styles.nav}>
        {links.map((item) => (
          <Link key={item.href} href={item.href} className={styles.link}>
            {item.label}
          </Link>
        ))}
      </nav>

      <div className={styles.actions}>
        <span className="pill">{chain.chainName}</span>
        {wallet ? <span className={styles.identity}>{shortenAddress(wallet.walletAddress)}</span> : null}
        {session ? (
          <button className={styles.ghost} onClick={revokeCurrentSession}>
            撤销会话
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
