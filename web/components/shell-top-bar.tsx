"use client";

import type { Route } from "next";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import { UserAvatar } from "@/components/user-avatar";
import { formatAssetAmount, shortenAddress } from "@/lib/format";
import { getBalancesRead } from "@/lib/api";
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
  const response = await fetch(`${API_BASE_URL}/api/v1/profile?user_id=${userId}`, {
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
  const { wallet, session, busy, statusMessage, prepareTrading } = useTradingSession();
  const searchRef = useRef<HTMLInputElement | null>(null);
  const [internalQuery, setInternalQuery] = useState("");
  const [fetchedBalance, setFetchedBalance] = useState<Balance | null>(null);
  const [profile, setProfile] = useState<UserProfile | null>(profileProp ?? null);
  const [balanceState, setBalanceState] = useState<"idle" | "loading" | "ready" | "error">("idle");

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
    if (!session?.userId || balanceDisplay) {
      setFetchedBalance(null);
      setBalanceState("idle");
      return;
    }

    let cancelled = false;
    setBalanceState("loading");

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
  }, [balanceDisplay, session?.userId]);

  useEffect(() => {
    if (!session?.userId || profileProp) {
      if (!profileProp) {
        setProfile(null);
      }
      return;
    }

    let cancelled = false;
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
  }, [profileProp, session?.userId]);

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
    <Link href={"/portfolio" as Route} className={styles.profileDock}>
      <div className={styles.profileMeta}>
        <strong>{profilePrimary}</strong>
        <span>{profileSecondary}</span>
      </div>
      <UserAvatar profile={profile} walletAddress={activeWalletAddress} size="md" />
    </Link>
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
          <button className={styles.iconButton} disabled aria-label="创建市场即将开放">
            <svg viewBox="0 0 20 20" className={styles.iconSvg} aria-hidden="true">
              <path d="M10 4v12M4 10h12" />
            </svg>
          </button>
          <button className={styles.iconButton} disabled aria-label="站内信即将开放">
            <svg viewBox="0 0 20 20" className={styles.iconSvg} aria-hidden="true">
              <path d="M10 16a2.2 2.2 0 0 0 2-1.2M5.6 14h8.8c-.7-.9-1.1-2-1.1-3.2V9.2c0-1.9-1.5-3.4-3.3-3.4S6.7 7.3 6.7 9.2v1.6c0 1.2-.4 2.3-1.1 3.2Z" />
            </svg>
          </button>
          {profileNode}
        </div>
      </div>
    </div>
  );
}
