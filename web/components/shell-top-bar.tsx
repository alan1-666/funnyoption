"use client";

import type { Route } from "next";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import { UserAvatar } from "@/components/user-avatar";
import { NotificationPanel } from "@/components/notification-panel";
import { MarketProposeForm } from "@/components/market-propose-form";
import { formatAssetAmount, shortenAddress } from "@/lib/format";
import { authenticatedFetch, getBalancesRead, getUnreadCount, setApiSessionId } from "@/lib/api";
import type { Balance, UserProfile } from "@/lib/types";
import styles from "@/components/shell-top-bar.module.css";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";
const COLLATERAL_SYMBOL = (process.env.NEXT_PUBLIC_COLLATERAL_SYMBOL ?? "USDT").toUpperCase();

async function fetchAvailableBalance(userId: number) {
  const balancesResult = await getBalancesRead(userId, {
    ensureAsset: COLLATERAL_SYMBOL
  });
  if (balancesResult.state === "unavailable") {
    throw new Error(balancesResult.error?.message ?? "读取余额失败");
  }
  const balances = balancesResult.items;
  return balances.find((balance) => balance.asset.toUpperCase() === COLLATERAL_SYMBOL) ?? null;
}

async function fetchUserProfile(userId: number) {
  const response = await authenticatedFetch(`${API_BASE_URL}/api/v1/profile?user_id=${userId}`, {
    cache: "no-store"
  });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }
  return (await response.json()) as UserProfile;
}

export function ShellTopBar({
  query,
  onQueryChange,
  searchPlaceholder = "Search markets",
  balanceDisplay,
  profile: profileProp
}: {
  query?: string;
  onQueryChange?: (next: string) => void;
  searchPlaceholder?: string;
  balanceDisplay?: string;
  profile?: UserProfile | null;
}) {
  const { wallet, session, busy, statusMessage, prepareTrading, revokeCurrentSession } = useTradingSession();
  const searchRef = useRef<HTMLInputElement | null>(null);
  const [internalQuery, setInternalQuery] = useState("");
  const [fetchedBalance, setFetchedBalance] = useState<Balance | null>(null);
  const [profile, setProfile] = useState<UserProfile | null>(profileProp ?? null);
  const [balanceState, setBalanceState] = useState<"idle" | "loading" | "ready" | "error">("idle");
  const [notifOpen, setNotifOpen] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [proposeOpen, setProposeOpen] = useState(false);
  const [showDisconnect, setShowDisconnect] = useState(false);
  const bellWrapRef = useRef<HTMLDivElement>(null);
  const disconnectRef = useRef<HTMLDivElement>(null);

  const controlled = typeof query === "string";
  const value = controlled ? query : internalQuery;
  const activeWalletAddress = wallet?.walletAddress ?? session?.walletAddress ?? profile?.wallet_address ?? "";

  useEffect(() => {
    setProfile(profileProp ?? null);
  }, [profileProp]);

  useEffect(() => {
    const handleSlashFocus = (event: KeyboardEvent) => {
      if (event.key !== "/" || event.metaKey || event.ctrlKey || event.altKey) {
        return;
      }
      const target = event.target as HTMLElement | null;
      if (target && (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable)) {
        return;
      }
      event.preventDefault();
      searchRef.current?.focus();
    };

    window.addEventListener("keydown", handleSlashFocus);
    return () => window.removeEventListener("keydown", handleSlashFocus);
  }, []);

  useEffect(() => {
    if (!session?.userId || !session?.sessionId || balanceDisplay) {
      setFetchedBalance(null);
      setBalanceState("idle");
      return;
    }

    let cancelled = false;
    setBalanceState("loading");

    // Ensure the module-level session ID is set before fetching,
    // because parent useEffect (setApiSessionId) runs after child effects.
    setApiSessionId(session.sessionId);

    fetchAvailableBalance(session.userId)
      .then((nextBalance) => {
        if (cancelled) return;
        setFetchedBalance(nextBalance);
        setBalanceState("ready");
      })
      .catch(() => {
        if (cancelled) return;
        setFetchedBalance(null);
        setBalanceState("error");
      });

    return () => {
      cancelled = true;
    };
  }, [balanceDisplay, session?.userId, session?.sessionId]);

  useEffect(() => {
    if (!session?.userId || !session?.sessionId || profileProp) {
      if (!profileProp) {
        setProfile(null);
      }
      return;
    }

    let cancelled = false;
    setApiSessionId(session.sessionId);

    fetchUserProfile(session.userId)
      .then((nextProfile) => {
        if (cancelled) return;
        setProfile(nextProfile);
      })
      .catch(() => {
        if (cancelled) return;
        setProfile(null);
      });

    return () => {
      cancelled = true;
    };
  }, [profileProp, session?.userId, session?.sessionId]);

  useEffect(() => {
    if (!session?.userId || !session?.sessionId) {
      setUnreadCount(0);
      return;
    }
    let cancelled = false;
    setApiSessionId(session.sessionId);
    getUnreadCount(session.userId).then((count) => {
      if (!cancelled) setUnreadCount(count);
    });
    return () => { cancelled = true; };
  }, [session?.userId, session?.sessionId]);

  const handleCloseNotif = useCallback(() => setNotifOpen(false), []);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (disconnectRef.current && !disconnectRef.current.contains(e.target as Node)) {
        setShowDisconnect(false);
      }
    }
    if (showDisconnect) {
      document.addEventListener("mousedown", handleClickOutside);
      return () => document.removeEventListener("mousedown", handleClickOutside);
    }
  }, [showDisconnect]);

  async function handleDisconnect() {
    setShowDisconnect(false);
    try {
      await revokeCurrentSession();
    } catch {
      // Provider handles status.
    }
  }

  async function handleProfileAction() {
    if (session) {
      return;
    }
    try {
      await prepareTrading();
    } catch {
      // Provider updates status state already.
    }
  }

  function handleSearchChange(next: string) {
    if (onQueryChange) {
      onQueryChange(next);
      return;
    }
    setInternalQuery(next);
  }

  const resolvedBalanceDisplay =
    balanceDisplay ??
    (balanceState === "loading"
      ? "读取中…"
      : fetchedBalance
        ? `${formatAssetAmount(fetchedBalance.available, fetchedBalance.asset)} ${fetchedBalance.asset}`
        : "0 USDT");

  const profilePrimary = session
    ? resolvedBalanceDisplay
    : busy === "connect" || busy === "session"
      ? "连接中..."
      : "Connect";

  const profileSecondary = session
    ? shortenAddress(activeWalletAddress)
    : statusMessage
      ? "点击完成钱包授权"
      : "钱包未连接";

  const profileNode = session ? (
    <div className={styles.profileDock} ref={disconnectRef} style={{ position: "relative" }}>
      <Link href={"/portfolio" as Route} className={styles.profileLink}>
        <div className={styles.profileMeta}>
          <strong>{profilePrimary}</strong>
          <span>{profileSecondary}</span>
        </div>
        <UserAvatar profile={profile} walletAddress={activeWalletAddress} size="md" />
      </Link>
      <button
        className={styles.disconnectToggle}
        onClick={() => setShowDisconnect((prev) => !prev)}
        aria-label="Wallet menu"
        title="钱包菜单"
      >
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M3 5L6 8L9 5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </button>
      {showDisconnect && (
        <div className={styles.disconnectDropdown}>
          <Link href={"/portfolio" as Route} className={styles.disconnectItem} onClick={() => setShowDisconnect(false)}>
            个人中心
          </Link>
          <Link href={"/deposit" as Route} className={styles.disconnectItem} onClick={() => setShowDisconnect(false)}>
            充值与提现
          </Link>
          <button className={styles.disconnectItem} onClick={handleDisconnect}>
            断开钱包
          </button>
        </div>
      )}
    </div>
  ) : (
    <button className={styles.profileDock} onClick={handleProfileAction} disabled={busy !== null}>
      <div className={styles.profileMeta}>
        <strong>{profilePrimary}</strong>
        <span>{profileSecondary}</span>
      </div>
      <UserAvatar profile={profile} walletAddress={activeWalletAddress} size="md" />
    </button>
  );

  return (
    <div className={styles.wrap}>
      <div className={styles.bar}>
        <Link href={"/" as Route} className={styles.brand}>
          <span className={styles.brandMark}>fo</span>
        </Link>

        <label className={styles.searchBar}>
          <span className={styles.searchIcon} aria-hidden="true">⌕</span>
          <input
            ref={searchRef}
            className={styles.searchInput}
            value={value}
            onChange={(event) => handleSearchChange(event.target.value)}
            placeholder={searchPlaceholder}
            type="search"
          />
          <span className={styles.searchHint}>/</span>
        </label>

        <div className={styles.actions}>
          <Link href={"/deposit" as Route} className={`${styles.iconButton} ${styles.iconButtonEnabled}`} aria-label="充值">
            <svg viewBox="0 0 20 20" className={styles.iconSvg} aria-hidden="true">
              <path d="M10 3v10M6 9l4 4 4-4M4 16h12" />
            </svg>
          </Link>
          <button
            className={`${styles.iconButton} ${session ? styles.iconButtonEnabled : ""}`}
            disabled={!session}
            aria-label="Propose a market"
            onClick={() => session && setProposeOpen(true)}
          >
            <svg viewBox="0 0 20 20" className={styles.iconSvg} aria-hidden="true">
              <path d="M10 4v12M4 10h12" />
            </svg>
          </button>
          <div className={styles.bellWrap} ref={bellWrapRef}>
            <button
              className={`${styles.iconButton} ${session ? styles.iconButtonEnabled : ""}`}
              disabled={!session}
              aria-label="Notifications"
              onClick={() => session && setNotifOpen((prev) => !prev)}
            >
              <svg viewBox="0 0 20 20" className={styles.iconSvg} aria-hidden="true">
                <path d="M10 16a2.2 2.2 0 0 0 2-1.2M5.6 14h8.8c-.7-.9-1.1-2-1.1-3.2V9.2c0-1.9-1.5-3.4-3.3-3.4S6.7 7.3 6.7 9.2v1.6c0 1.2-.4 2.3-1.1 3.2Z" />
              </svg>
            </button>
            {unreadCount > 0 && <span className={styles.badgeDot} />}
            {session && (
              <NotificationPanel
                userId={session.userId}
                open={notifOpen}
                onClose={handleCloseNotif}
                unreadCount={unreadCount}
                setUnreadCount={setUnreadCount}
              />
            )}
          </div>
          {profileNode}
        </div>

        {proposeOpen && <MarketProposeForm onClose={() => setProposeOpen(false)} />}
      </div>
    </div>
  );
}
