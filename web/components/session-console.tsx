"use client";

import { useEffect, useMemo, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import { formatTimestamp, shortenAddress } from "@/lib/format";
import { zhGenericStatus } from "@/lib/locale";
import type { SessionGrant } from "@/lib/types";
import { listSessions, revokeRemoteSession } from "@/lib/session-client";
import styles from "@/components/session-console.module.css";

interface SessionConsoleProps {
  initialSessions: SessionGrant[];
}

export function SessionConsole({ initialSessions }: SessionConsoleProps) {
  const { wallet, session, busy, restoring, restoreStatus, statusMessage, createSession, prepareTrading, revokeCurrentSession } = useTradingSession();
  const [sessions, setSessions] = useState(initialSessions);
  const [statusOverride, setStatusOverride] = useState<string | null>(null);

  const currentSessionId = session?.sessionId;
  const status = statusOverride ?? statusMessage;
  const needsReauthorization =
    !session &&
    wallet &&
    ["expired", "revoked", "rotated", "missing_private_key", "remote_missing", "remote_mismatch", "vault_mismatch"].includes(restoreStatus);
  const sortedSessions = useMemo(
    () => [...sessions].sort((left, right) => Number(right.updated_at || 0) - Number(left.updated_at || 0)),
    [sessions]
  );

  useEffect(() => {
    setStatusOverride(null);
  }, [statusMessage, restoreStatus, session?.sessionId, wallet?.walletAddress, wallet?.chainId]);

  async function refresh() {
    try {
      const items = await listSessions({ walletAddress: wallet?.walletAddress });
      setSessions(items);
      setStatusOverride("交易密钥列表已刷新");
    } catch (error) {
      setStatusOverride(error instanceof Error ? error.message : "刷新交易密钥失败");
    }
  }

  async function handleCreate() {
    try {
      const created = session ? await createSession(wallet, { forceRotate: true }) : await prepareTrading();
      if (session) {
        setStatusOverride("交易密钥已轮换");
      } else {
        setStatusOverride("交易密钥已开启");
      }
      const items = await listSessions({ walletAddress: created?.walletAddress ?? wallet?.walletAddress });
      setSessions(items);
    } catch (error) {
      setStatusOverride(error instanceof Error ? error.message : "创建交易密钥失败");
    }
  }

  async function handleRevoke(sessionId: string) {
    try {
      if (sessionId === currentSessionId) {
        await revokeCurrentSession();
      } else {
        await revokeRemoteSession(sessionId);
      }
      const items = await listSessions({ walletAddress: wallet?.walletAddress });
      setSessions(items);
      setStatusOverride(`已撤销交易密钥 ${sessionId}`);
    } catch (error) {
      setStatusOverride(error instanceof Error ? error.message : "撤销交易密钥失败");
    }
  }

  return (
    <section className={`panel ${styles.console}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">交易密钥</span>
          <p className={styles.copy}>完成一次钱包授权后，可以持续复用浏览器本地交易密钥，直到你手动轮换或撤销。</p>
        </div>
        <div className={styles.actions}>
          <button className={styles.ghost} onClick={refresh}>刷新</button>
          <button className={styles.button} onClick={handleCreate} disabled={restoring || busy !== null}>
            {restoring ? "恢复中..." : busy === "session" ? "授权中..." : session ? "轮换密钥" : needsReauthorization ? "重新授权" : wallet ? "开启交易" : "开始交易"}
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
                <h3 className={styles.title}>{isCurrent ? "当前密钥" : zhGenericStatus(item.status)}</h3>
                <div className={styles.meta}>
                  <span>{item.session_id}</span>
                  <span>{shortenAddress(item.wallet_address)}</span>
                  <span>{item.expires_at > 0 ? `过期时间 ${formatTimestamp(Math.floor(item.expires_at / 1000))}` : "长期有效，直到轮换或撤销"}</span>
                </div>
              </div>
              <div className={styles.side}>
                <span className="pill">{isCurrent ? "使用中" : zhGenericStatus(item.status)}</span>
                {item.status === "ACTIVE" ? (
                  <button className={styles.ghost} onClick={() => handleRevoke(item.session_id)}>
                    撤销
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
