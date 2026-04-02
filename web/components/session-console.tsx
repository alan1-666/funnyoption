"use client";

import { useMemo, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import { formatTimestamp, shortenAddress } from "@/lib/format";
import type { SessionGrant } from "@/lib/types";
import { listSessions, revokeRemoteSession } from "@/lib/session-client";
import styles from "@/components/session-console.module.css";

interface SessionConsoleProps {
  initialSessions: SessionGrant[];
}

export function SessionConsole({ initialSessions }: SessionConsoleProps) {
  const { wallet, session, busy, createSession, prepareTrading, revokeCurrentSession } = useTradingSession();
  const [sessions, setSessions] = useState(initialSessions);
  const [status, setStatus] = useState("Trading access is ready");

  const currentSessionId = session?.sessionId;
  const sortedSessions = useMemo(
    () => [...sessions].sort((left, right) => Number(right.updated_at || 0) - Number(left.updated_at || 0)),
    [sessions]
  );

  async function refresh() {
    try {
      const items = await listSessions({ walletAddress: wallet?.walletAddress });
      setSessions(items);
      setStatus("Session list refreshed");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Failed to refresh sessions");
    }
  }

  async function handleCreate() {
    try {
      const created = session ? await createSession(wallet) : await prepareTrading();
      if (session) {
        setStatus("Trading access rotated");
      } else {
        setStatus("Trading access enabled");
      }
      const items = await listSessions({ walletAddress: created?.walletAddress ?? wallet?.walletAddress });
      setSessions(items);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Failed to create session");
    }
  }

  async function handleRevoke(sessionId: string) {
    try {
      await revokeRemoteSession(sessionId);
      if (sessionId === currentSessionId) {
        await revokeCurrentSession();
      }
      const items = await listSessions({ walletAddress: wallet?.walletAddress });
      setSessions(items);
      setStatus(`Trading access ${sessionId} revoked`);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Failed to revoke session");
    }
  }

  return (
    <section className={`panel ${styles.console}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">Trading access</span>
          <p className={styles.copy}>Approve trading once, then reuse that access for future orders until you rotate or revoke it.</p>
        </div>
        <div className={styles.actions}>
          <button className={styles.ghost} onClick={refresh}>Refresh</button>
          <button className={styles.button} onClick={handleCreate}>
            {busy === "session" ? "Authorizing..." : session ? "Rotate Session" : wallet ? "Enable Trading" : "Start Trading"}
          </button>
        </div>
      </div>

      <div className={styles.status}>{status}</div>

      <div className={styles.list}>
        {sortedSessions.map((item) => {
          const isCurrent = item.session_id === currentSessionId;
          return (
            <article key={item.session_id} className={styles.item}>
              <div>
                <h3 className={styles.title}>{isCurrent ? "Current session" : item.status}</h3>
                <div className={styles.meta}>
                  <span>{item.session_id}</span>
                  <span>{shortenAddress(item.wallet_address)}</span>
                  <span>exp {formatTimestamp(Math.floor(item.expires_at / 1000))}</span>
                </div>
              </div>
              <div className={styles.side}>
                <span className="pill">{isCurrent ? "IN USE" : item.status}</span>
                {item.status === "ACTIVE" ? (
                  <button className={styles.ghost} onClick={() => handleRevoke(item.session_id)}>
                    Revoke
                  </button>
                ) : null}
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}
