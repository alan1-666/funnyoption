"use client";

import { useMemo, useState } from "react";

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
  const { wallet, session, busy, createSession, prepareTrading, revokeCurrentSession } = useTradingSession();
  const [sessions, setSessions] = useState(initialSessions);
  const [status, setStatus] = useState("交易会话已就绪");

  const currentSessionId = session?.sessionId;
  const sortedSessions = useMemo(
    () => [...sessions].sort((left, right) => Number(right.updated_at || 0) - Number(left.updated_at || 0)),
    [sessions]
  );

  async function refresh() {
    try {
      const items = await listSessions({ walletAddress: wallet?.walletAddress });
      setSessions(items);
      setStatus("会话列表已刷新");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "刷新会话失败");
    }
  }

  async function handleCreate() {
    try {
      const created = session ? await createSession(wallet) : await prepareTrading();
      if (session) {
        setStatus("交易会话已轮换");
      } else {
        setStatus("交易会话已开启");
      }
      const items = await listSessions({ walletAddress: created?.walletAddress ?? wallet?.walletAddress });
      setSessions(items);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "创建会话失败");
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
      setStatus(`已撤销会话 ${sessionId}`);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "撤销会话失败");
    }
  }

  return (
    <section className={`panel ${styles.console}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">交易会话</span>
          <p className={styles.copy}>完成一次交易授权后，可以持续复用该会话，直到你手动轮换或撤销。</p>
        </div>
        <div className={styles.actions}>
          <button className={styles.ghost} onClick={refresh}>刷新</button>
          <button className={styles.button} onClick={handleCreate}>
            {busy === "session" ? "授权中..." : session ? "轮换会话" : wallet ? "开启交易" : "开始交易"}
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
                <h3 className={styles.title}>{isCurrent ? "当前会话" : zhGenericStatus(item.status)}</h3>
                <div className={styles.meta}>
                  <span>{item.session_id}</span>
                  <span>{shortenAddress(item.wallet_address)}</span>
                  <span>过期时间 {formatTimestamp(Math.floor(item.expires_at / 1000))}</span>
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
