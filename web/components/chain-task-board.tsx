"use client";

import { useEffect, useState } from "react";

import { getChainMeta } from "@/lib/chain";
import { formatTimestamp, formatToken, shortenAddress } from "@/lib/format";
import type { ChainTask } from "@/lib/types";
import { fetchChainTasks } from "@/lib/session-client";
import styles from "@/components/chain-task-board.module.css";

interface ChainTaskBoardProps {
  initialTasks: ChainTask[];
  title?: string;
  copy?: string;
}

export function ChainTaskBoard({
  initialTasks,
  title = "Chain queue",
  copy = "Queued claims and chain-side work refresh on a timer so the desk stays honest after you click."
}: ChainTaskBoardProps) {
  const [tasks, setTasks] = useState(initialTasks);
  const [status, setStatus] = useState(initialTasks.length > 0 ? "Polling queue every 5s" : "Queue is clear in the local API snapshot");
  const chain = getChainMeta();
  const counts = tasks.reduce<Record<string, number>>((accumulator, task) => {
    const key = String(task.status || "UNKNOWN").toUpperCase();
    accumulator[key] = (accumulator[key] ?? 0) + 1;
    return accumulator;
  }, {});

  async function refresh() {
    try {
      const items = await fetchChainTasks();
      setTasks(items);
      setStatus(items.length > 0 ? `Last refresh ${new Date().toLocaleTimeString()}` : "Queue is clear in the local API snapshot");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Failed to refresh chain queue");
    }
  }

  useEffect(() => {
    const timer = window.setInterval(() => {
      void refresh();
    }, 5000);
    return () => window.clearInterval(timer);
  }, []);

  return (
    <section className={`panel ${styles.board}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">Chain queue</span>
          <p className={styles.copy}>{copy}</p>
        </div>
        <div className={styles.actions}>
          <button className={styles.button} onClick={refresh}>Refresh</button>
        </div>
      </div>

      <div className={styles.status}>{status}</div>

      {tasks.length > 0 ? (
        <div className={styles.status}>
          {Object.entries(counts).map(([label, count]) => `${label.toLowerCase()} ${count}`).join(" · ")}
        </div>
      ) : null}

      <div className={styles.list}>
        {tasks.length > 0 ? (
          tasks.map((task) => (
            <article key={task.id} className={styles.row}>
              <div>
                <div className={styles.title}>{title} · {task.biz_type} · {task.ref_id}</div>
                <div className={styles.meta}>
                  <span>{task.chain_name}/{task.network_name}</span>
                  <span>{shortenAddress(task.wallet_address)}</span>
                  {task.payload?.market_id ? <span>market {task.payload.market_id}</span> : null}
                  {task.payload?.payout_amount ? <span>payout {formatToken(Number(task.payload.payout_amount) / 100, 0)} USDT</span> : null}
                  <span>{formatTimestamp(task.updated_at || task.created_at)}</span>
                  {task.error_message ? <span>{task.error_message}</span> : null}
                </div>
              </div>
              <div className={styles.side}>
                <span className="pill">{task.status}</span>
                <span>attempt {task.attempt_count}</span>
                {task.tx_hash ? (
                  <a
                    className={styles.link}
                    href={`${chain.explorerUrl}/tx/${task.tx_hash}`}
                    target="_blank"
                    rel="noreferrer"
                  >
                    {shortenAddress(task.tx_hash)}
                  </a>
                ) : (
                  <span>tx pending</span>
                )}
              </div>
            </article>
          ))
        ) : (
          <article className={styles.row}>
            <div>
              <div className={styles.title}>Queue clear</div>
              <div className={styles.meta}>
                <span>No `chain_transactions` rows are pending right now.</span>
                <span>This is the real local queue state, not a fallback claim fixture.</span>
              </div>
            </div>
            <div className={styles.side}>
              <span className="pill">EMPTY</span>
              <span>0 tasks</span>
            </div>
          </article>
        )}
      </div>
    </section>
  );
}
