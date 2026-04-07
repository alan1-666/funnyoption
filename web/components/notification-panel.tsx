"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import type { Route } from "next";

import { getNotifications, getUnreadCount, markNotificationRead, markAllNotificationsRead } from "@/lib/api";
import type { Notification } from "@/lib/types";
import styles from "@/components/shell-top-bar.module.css";

const WS_BASE_URL = process.env.NEXT_PUBLIC_WS_BASE_URL ?? "ws://127.0.0.1:8085";

function formatTimeAgo(unixSeconds: number): string {
  const now = Math.floor(Date.now() / 1000);
  const diff = now - unixSeconds;
  if (diff < 60) return "just now";
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

export function NotificationPanel({
  userId,
  open,
  onClose,
  unreadCount,
  setUnreadCount,
}: {
  userId: number;
  open: boolean;
  onClose: () => void;
  unreadCount: number;
  setUnreadCount: (n: number | ((prev: number) => number)) => void;
}) {
  const router = useRouter();
  const [items, setItems] = useState<Notification[]>([]);
  const [loaded, setLoaded] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open || !userId) return;
    let cancelled = false;
    getNotifications(userId).then((data) => {
      if (cancelled) return;
      setItems(data);
      setLoaded(true);
    });
    return () => { cancelled = true; };
  }, [open, userId]);

  useEffect(() => {
    if (!userId) return;
    const ws = new WebSocket(`${WS_BASE_URL}/ws?stream=notifications&user_id=${userId}`);
    ws.onmessage = (event) => {
      try {
        const envelope = JSON.parse(event.data);
        if (envelope?.type === "notification") {
          const payload = envelope.payload;
          const notif: Notification = {
            notification_id: payload.notification_id,
            user_id: payload.user_id,
            type: payload.type,
            title: payload.title,
            body: "",
            metadata: {},
            is_read: false,
            created_at: payload.created_at,
          };
          setItems((prev) => [notif, ...prev]);
          setUnreadCount((prev: number) => prev + 1);
        }
      } catch { /* ignore */ }
    };
    return () => { ws.close(); };
  }, [userId, setUnreadCount]);

  useEffect(() => {
    if (!open) return;
    function handleClickOutside(e: MouseEvent) {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        onClose();
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [open, onClose]);

  const handleMarkAll = useCallback(async () => {
    await markAllNotificationsRead();
    setItems((prev) => prev.map((n) => ({ ...n, is_read: true })));
    setUnreadCount(0);
  }, [setUnreadCount]);

  const handleClickItem = useCallback(async (notif: Notification) => {
    if (!notif.is_read) {
      await markNotificationRead(notif.notification_id);
      setItems((prev) =>
        prev.map((n) => n.notification_id === notif.notification_id ? { ...n, is_read: true } : n)
      );
      setUnreadCount((prev: number) => Math.max(0, prev - 1));
    }
    const marketId = notif.metadata?.market_id;
    if (marketId) {
      router.push(`/market/${marketId}` as Route);
      onClose();
    }
  }, [router, onClose, setUnreadCount]);

  if (!open) return null;

  return (
    <div ref={panelRef} className={styles.notifPanel}>
      <div className={styles.notifHeader}>
        <h3>Notifications</h3>
        {unreadCount > 0 && (
          <button className={styles.markAllBtn} onClick={handleMarkAll}>
            Mark all read
          </button>
        )}
      </div>
      <div className={styles.notifList}>
        {loaded && items.length === 0 && (
          <div className={styles.notifEmpty}>No notifications yet</div>
        )}
        {items.map((n) => (
          <button
            key={n.notification_id}
            className={`${styles.notifItem} ${!n.is_read ? styles.notifItemUnread : ""}`}
            onClick={() => handleClickItem(n)}
          >
            <p className={styles.notifTitle}>{n.title}</p>
            <span className={styles.notifTime}>{formatTimeAgo(n.created_at)}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
