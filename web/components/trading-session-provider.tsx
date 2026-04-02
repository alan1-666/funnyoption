"use client";

import { createContext, startTransition, useContext, useEffect, useState } from "react";
import { getChainMeta } from "@/lib/chain";

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
const TARGET_CHAIN = getChainMeta();

function formatError(error: unknown) {
  if (!(error instanceof Error)) {
    return "钱包操作失败";
  }

  if (/metamask or an eip-1193 wallet is required/i.test(error.message) || /metamask or a compatible wallet is required/i.test(error.message)) {
    return "需要 MetaMask 或兼容的钱包扩展";
  }

  return error.message;
}

export function TradingSessionProvider({ children }: { children: React.ReactNode }) {
  const [wallet, setWallet] = useState<WalletConnection | null>(null);
  const [session, setSession] = useState<SessionRecord | null>(null);
  const [busy, setBusy] = useState<"connect" | "session" | null>(null);
  const [statusMessage, setStatusMessage] = useState("钱包待命");

  useEffect(() => {
    const stored = loadStoredSession();
    if (!stored) {
      return;
    }
    if (stored.chainId !== TARGET_CHAIN.chainId) {
      clearStoredSession();
      setStatusMessage(`本地会话来自链 ${stored.chainId}，请切换到 ${TARGET_CHAIN.chainName}（${TARGET_CHAIN.chainId}）。`);
      return;
    }
    setSession(stored);
    setWallet({ walletAddress: stored.walletAddress, chainId: stored.chainId });
    setStatusMessage("已恢复本地会话");
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
          setStatusMessage("钱包已断开");
          return;
        }

        setWallet((current) => ({ walletAddress, chainId: current?.chainId ?? 0 }));
        if (session && session.walletAddress !== walletAddress) {
          setSession(null);
          clearStoredSession();
          setStatusMessage("钱包已切换，已清空旧会话");
        }
      });
    };

    const handleChainChanged = (chainIdHex: unknown) => {
      if (typeof chainIdHex !== "string") return;
      const chainId = Number.parseInt(chainIdHex, 16);
      startTransition(() => {
        setWallet((current) => (current ? { ...current, chainId } : current));
        setStatusMessage(`已切换到链 ${chainId}`);
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
    if (wallet && wallet.chainId === TARGET_CHAIN.chainId) {
      setStatusMessage(`钱包已连接到 ${TARGET_CHAIN.chainName}（${wallet.chainId}）`);
      return wallet;
    }

    setBusy("connect");
    setStatusMessage("连接钱包中...");
    try {
      const connection = await connectWallet();
      startTransition(() => {
        setWallet(connection);
        setStatusMessage(`钱包已连接到 ${TARGET_CHAIN.chainName}（${connection.chainId}）`);
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
    setStatusMessage("正在授权交易会话...");
    try {
      const readyWallet =
        activeWallet && activeWallet.chainId === TARGET_CHAIN.chainId
          ? activeWallet
          : wallet && wallet.chainId === TARGET_CHAIN.chainId
            ? wallet
            : await handleConnect();
      const created = await authorizeSession(readyWallet ?? undefined);
      startTransition(() => {
        setSession(created);
        setWallet({ walletAddress: created.walletAddress, chainId: created.chainId });
        setStatusMessage("交易已开启");
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
      setStatusMessage("交易已开启");
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
    startTransition(() => setStatusMessage(`订单签名已完成，序号 ${payload.orderNonce}`));
    return payload;
  }

  function handleCommitOrderNonce(nonce: number) {
    startTransition(() => {
      setSession((current) => (current ? bumpSessionOrderNonce(current, nonce) : current));
      setStatusMessage(`订单已入队，序号 ${nonce}`);
    });
  }

  function handleClear() {
    clearStoredSession();
    setSession(null);
    setStatusMessage("本地会话已清空");
  }

  async function handleRevokeCurrentSession() {
    if (!session) {
      handleClear();
      return;
    }
    await revokeRemoteSession(session.sessionId);
    handleClear();
    setStatusMessage("会话已撤销");
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
