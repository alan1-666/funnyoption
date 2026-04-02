import type { CSSProperties } from "react";

import type { UserProfile } from "@/lib/types";

const AVATAR_PRESET_STYLES: Record<string, { background: string; color: string; shadow: string }> = {
  aurora: {
    background: "linear-gradient(135deg, #ffd45c 0%, #ff8d5a 34%, #ff6fb6 68%, #8f7cff 100%)",
    color: "rgba(17, 17, 18, 0.96)",
    shadow: "0 16px 28px rgba(255, 111, 182, 0.22)"
  },
  ember: {
    background: "linear-gradient(135deg, #ffb36b 0%, #ff6d55 45%, #f5367f 100%)",
    color: "rgba(22, 11, 8, 0.96)",
    shadow: "0 16px 28px rgba(245, 54, 127, 0.22)"
  },
  ocean: {
    background: "linear-gradient(135deg, #73f2ff 0%, #3997ff 44%, #3454ff 100%)",
    color: "rgba(6, 17, 38, 0.96)",
    shadow: "0 16px 28px rgba(57, 151, 255, 0.2)"
  },
  violet: {
    background: "linear-gradient(135deg, #d09cff 0%, #8d7dff 46%, #5e53ff 100%)",
    color: "rgba(15, 10, 30, 0.96)",
    shadow: "0 16px 28px rgba(141, 125, 255, 0.2)"
  },
  mono: {
    background: "linear-gradient(135deg, #f4f4f4 0%, #b8bcc8 52%, #6f7584 100%)",
    color: "rgba(16, 17, 19, 0.96)",
    shadow: "0 16px 28px rgba(111, 117, 132, 0.22)"
  },
  forest: {
    background: "linear-gradient(135deg, #98ffcf 0%, #2fc287 46%, #1f7f5d 100%)",
    color: "rgba(9, 28, 22, 0.96)",
    shadow: "0 16px 28px rgba(47, 194, 135, 0.2)"
  }
};

export const USER_AVATAR_PRESETS = [
  { key: "aurora", label: "极光" },
  { key: "ember", label: "落霞" },
  { key: "ocean", label: "深海" },
  { key: "violet", label: "夜紫" },
  { key: "mono", label: "银灰" },
  { key: "forest", label: "森林" }
] as const;

export function getAvatarPresetStyle(preset?: string) {
  return AVATAR_PRESET_STYLES[preset ?? ""] ?? AVATAR_PRESET_STYLES.aurora;
}

export function getAvatarStyle(preset?: string): CSSProperties {
  const visual = getAvatarPresetStyle(preset);
  return {
    background: visual.background,
    color: visual.color,
    boxShadow: visual.shadow
  };
}

export function getAvatarMonogram(profile?: Pick<UserProfile, "display_name"> | null, walletAddress?: string) {
  const displayName = profile?.display_name?.trim() ?? "";
  if (displayName) {
    const compact = displayName.replace(/\s+/g, "");
    return compact.slice(0, Math.min(2, compact.length)).toUpperCase();
  }
  const normalizedWallet = (walletAddress ?? "").trim();
  if (normalizedWallet.startsWith("0x") && normalizedWallet.length >= 4) {
    return normalizedWallet.slice(2, 4).toUpperCase();
  }
  return "FO";
}
