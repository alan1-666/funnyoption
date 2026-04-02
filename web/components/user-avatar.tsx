"use client";

import type { CSSProperties } from "react";

import { getAvatarMonogram, getAvatarStyle } from "@/lib/avatar";
import type { UserProfile } from "@/lib/types";
import styles from "@/components/user-avatar.module.css";

export function UserAvatar({
  profile,
  walletAddress,
  size = "md",
  shape = "circle",
  className = "",
  title
}: {
  profile?: UserProfile | null;
  walletAddress?: string;
  size?: "sm" | "md" | "lg" | "xl";
  shape?: "circle" | "panel";
  className?: string;
  title?: string;
}) {
  const visualStyle = getAvatarStyle(profile?.avatar_preset);
  const label = getAvatarMonogram(profile, walletAddress);
  const mergedStyle = visualStyle as CSSProperties;

  return (
    <span
      className={`${styles.avatar} ${styles[`size-${size}`]} ${styles[`shape-${shape}`]} ${className}`.trim()}
      style={mergedStyle}
      title={title}
      aria-hidden="true"
    >
      {label}
    </span>
  );
}
