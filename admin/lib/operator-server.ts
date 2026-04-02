import { NextResponse } from "next/server";
import { recoverMessageAddress } from "viem";

import {
  normalizeAddress,
  parseOperatorWallets,
  type SignedOperatorAction,
  OPERATOR_SIGNATURE_WINDOW_MS
} from "@/lib/operator-auth";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";

function operatorWallets() {
  return parseOperatorWallets(process.env.FUNNYOPTION_OPERATOR_WALLETS ?? process.env.NEXT_PUBLIC_OPERATOR_WALLETS ?? "");
}

export function operatorUserID() {
  const parsed = Number(process.env.FUNNYOPTION_DEFAULT_OPERATOR_USER_ID ?? process.env.NEXT_PUBLIC_DEFAULT_OPERATOR_USER_ID ?? "1001");
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 1001;
}

export async function authorizeOperatorAction(message: string, operator: SignedOperatorAction) {
  const allowedWallets = operatorWallets();
  if (allowedWallets.length === 0) {
    return {
      ok: false as const,
      response: NextResponse.json(
        { error: "FUNNYOPTION_OPERATOR_WALLETS is not configured for the admin service" },
        { status: 403 }
      )
    };
  }

  const walletAddress = normalizeAddress(operator.walletAddress);
  if (!walletAddress || !operator.signature.trim() || operator.requestedAt <= 0) {
    return {
      ok: false as const,
      response: NextResponse.json({ error: "operator wallet, signature, and requested_at are required" }, { status: 400 })
    };
  }

  const age = Math.abs(Date.now() - Math.floor(operator.requestedAt));
  if (age > OPERATOR_SIGNATURE_WINDOW_MS) {
    return {
      ok: false as const,
      response: NextResponse.json({ error: "operator signature expired" }, { status: 401 })
    };
  }

  let recoveredWallet = "";
  try {
    recoveredWallet = normalizeAddress(
      await recoverMessageAddress({
        message,
        signature: operator.signature as `0x${string}`
      })
    );
  } catch {
    return {
      ok: false as const,
      response: NextResponse.json({ error: "invalid operator signature" }, { status: 401 })
    };
  }

  if (recoveredWallet !== walletAddress) {
    return {
      ok: false as const,
      response: NextResponse.json({ error: "operator signature does not match wallet" }, { status: 401 })
    };
  }

  if (!allowedWallets.includes(recoveredWallet)) {
    return {
      ok: false as const,
      response: NextResponse.json({ error: "wallet is not authorized for operator actions" }, { status: 403 })
    };
  }

  return {
    ok: true as const,
    walletAddress: recoveredWallet
  };
}

export function toCoreOperatorProof(operator: SignedOperatorAction) {
  return {
    wallet_address: operator.walletAddress,
    requested_at: operator.requestedAt,
    signature: operator.signature
  };
}

export async function forwardJSON(path: string, init: { method: "POST"; body: unknown }) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: init.method,
    headers: {
      "Content-Type": "application/json"
    },
    cache: "no-store",
    body: JSON.stringify(init.body)
  });

  const payload = (await response.json().catch(() => null)) as Record<string, unknown> | null;
  if (!response.ok) {
    return {
      ok: false as const,
      status: response.status,
      error: String(payload?.error ?? `HTTP ${response.status}`),
      response: NextResponse.json(
        { error: payload?.error ?? `HTTP ${response.status}` },
        { status: response.status }
      )
    };
  }

  return {
    ok: true as const,
    status: response.status,
    payload: payload ?? {}
  };
}
