"use client";

import { createContext, startTransition, useContext, useEffect, useState } from "react";

import {
  authorizeSession,
  bumpSessionOrderNonce,
  clearStoredSession,
  connectWallet,
  loadStoredSession,
  revokeRemoteSession,
  type OrderSignaturePayload,
  type SessionRecord,
  signOrderWithSession,
  type WalletConnection
} from "@/lib/session-client";

interface TradingSessionContextValue {
  wallet: WalletConnection | null;
  session: SessionRecord | null;
  busy: "connect" | "session" | null;
  statusMessage: string;
  connect: () => Promise<WalletConnection | null>;
  createSession: (wallet?: WalletConnection | null) => Promise<SessionRecord | null>;
  prepareTrading: () => Promise<SessionRecord | null>;
  signOrder: (order: {
    marketId: number;
    outcome: string;
    side: string;
    orderType: string;
    timeInForce: string;
    price: number;
    quantity: number;
    clientOrderId: string;
  }, activeSession?: SessionRecord | null) => Promise<OrderSignaturePayload | null>;
  commitOrderNonce: (nonce: number) => void;
  revokeCurrentSession: () => Promise<void>;
  clear: () => void;
}

const TradingSessionContext = createContext<TradingSessionContextValue | null>(null);

function formatError(error: unknown) {
  return error instanceof Error ? error.message : "Unknown wallet error";
}

export function TradingSessionProvider({ children }: { children: React.ReactNode }) {
  const [wallet, setWallet] = useState<WalletConnection | null>(null);
  const [session, setSession] = useState<SessionRecord | null>(null);
  const [busy, setBusy] = useState<"connect" | "session" | null>(null);
  const [statusMessage, setStatusMessage] = useState("Wallet idle");

  useEffect(() => {
    const stored = loadStoredSession();
    if (stored) {
      setSession(stored);
      setWallet({ walletAddress: stored.walletAddress, chainId: stored.chainId });
      setStatusMessage("Session restored");
    }
  }, []);

  useEffect(() => {
    if (typeof window === "undefined" || !window.ethereum?.on) {
      return;
    }

    const handleAccountsChanged = (accounts: unknown) => {
      const walletAddress = Array.isArray(accounts) && typeof accounts[0] === "string" ? accounts[0].toLowerCase() : "";
      startTransition(() => {
        if (!walletAddress) {
          setWallet(null);
          setSession(null);
          clearStoredSession();
          setStatusMessage("Wallet disconnected");
          return;
        }

        setWallet((current) => ({ walletAddress, chainId: current?.chainId ?? 0 }));
        if (session && session.walletAddress !== walletAddress) {
          setSession(null);
          clearStoredSession();
          setStatusMessage("Wallet switched, local session cleared");
        }
      });
    };

    const handleChainChanged = (chainIdHex: unknown) => {
      if (typeof chainIdHex !== "string") return;
      const chainId = Number.parseInt(chainIdHex, 16);
      startTransition(() => {
        setWallet((current) => (current ? { ...current, chainId } : current));
        setStatusMessage(`Chain changed to ${chainId}`);
      });
    };

    window.ethereum.on("accountsChanged", handleAccountsChanged);
    window.ethereum.on("chainChanged", handleChainChanged);

    return () => {
      window.ethereum?.removeListener?.("accountsChanged", handleAccountsChanged);
      window.ethereum?.removeListener?.("chainChanged", handleChainChanged);
    };
  }, [session]);

  async function handleConnect() {
    if (wallet) {
      setStatusMessage(`Wallet ready on chain ${wallet.chainId}`);
      return wallet;
    }

    setBusy("connect");
    setStatusMessage("Connecting wallet...");
    try {
      const connection = await connectWallet();
      startTransition(() => {
        setWallet(connection);
        setStatusMessage(`Wallet linked on chain ${connection.chainId}`);
      });
      return connection;
    } catch (error) {
      const message = formatError(error);
      startTransition(() => setStatusMessage(message));
      throw error;
    } finally {
      setBusy(null);
    }
  }

  async function handleCreateSession(activeWallet?: WalletConnection | null) {
    setBusy("session");
    setStatusMessage("Authorizing session key...");
    try {
      const created = await authorizeSession(activeWallet ?? wallet ?? undefined);
      startTransition(() => {
        setSession(created);
        setWallet({ walletAddress: created.walletAddress, chainId: created.chainId });
        setStatusMessage("Trading is enabled");
      });
      return created;
    } catch (error) {
      const message = formatError(error);
      startTransition(() => setStatusMessage(message));
      throw error;
    } finally {
      setBusy(null);
    }
  }

  async function handlePrepareTrading() {
    if (session) {
      setStatusMessage("Trading is already enabled");
      return session;
    }

    const activeWallet = wallet ?? (await handleConnect());
    if (!activeWallet) {
      return null;
    }

    return handleCreateSession(activeWallet);
  }

  async function handleSignOrder(order: {
    marketId: number;
    outcome: string;
    side: string;
    orderType: string;
    timeInForce: string;
    price: number;
    quantity: number;
    clientOrderId: string;
  }, activeSession?: SessionRecord | null) {
    const sessionRecord = activeSession ?? session;
    if (!sessionRecord) return null;
    const payload = await signOrderWithSession(sessionRecord, order);
    startTransition(() => setStatusMessage(`Signed order #${payload.orderNonce}`));
    return payload;
  }

  function handleCommitOrderNonce(nonce: number) {
    startTransition(() => {
      setSession((current) => (current ? bumpSessionOrderNonce(current, nonce) : current));
      setStatusMessage(`Queued order #${nonce}`);
    });
  }

  function handleClear() {
    clearStoredSession();
    setSession(null);
    setStatusMessage("Session cleared");
  }

  async function handleRevokeCurrentSession() {
    if (!session) {
      handleClear();
      return;
    }
    await revokeRemoteSession(session.sessionId);
    handleClear();
    setStatusMessage("Session revoked");
  }

  return (
    <TradingSessionContext.Provider
      value={{
        wallet,
        session,
        busy,
        statusMessage,
        connect: handleConnect,
        createSession: handleCreateSession,
        prepareTrading: handlePrepareTrading,
        signOrder: handleSignOrder,
        commitOrderNonce: handleCommitOrderNonce,
        revokeCurrentSession: handleRevokeCurrentSession,
        clear: handleClear
      }}
    >
      {children}
    </TradingSessionContext.Provider>
  );
}

export function useTradingSession() {
  const context = useContext(TradingSessionContext);
  if (!context) {
    throw new Error("useTradingSession must be used within TradingSessionProvider");
  }
  return context;
}
