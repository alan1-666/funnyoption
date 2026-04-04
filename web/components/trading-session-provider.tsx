"use client";

import { createContext, startTransition, useContext, useEffect, useRef, useState } from "react";
import { getChainMeta } from "@/lib/chain";

import {
  authorizeSession,
  bumpSessionOrderNonce,
  clearStoredSession,
  connectWallet,
  getWalletConnection,
  revokeRemoteSession,
  restoreStoredSession,
  type OrderSignaturePayload,
  type RestoreSessionResult,
  type RestoreSessionStatus,
  type SessionRecord,
  signOrderWithSession,
  type WalletConnection
} from "@/lib/session-client";

interface TradingSessionContextValue {
  wallet: WalletConnection | null;
  session: SessionRecord | null;
  busy: "connect" | "session" | null;
  restoring: boolean;
  restoreStatus: RestoreSessionStatus | "idle";
  statusMessage: string;
  connect: () => Promise<WalletConnection | null>;
  createSession: (wallet?: WalletConnection | null, options?: { forceRotate?: boolean }) => Promise<SessionRecord | null>;
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
  const [restoring, setRestoring] = useState(true);
  const [restoreStatus, setRestoreStatus] = useState<RestoreSessionStatus | "idle">("idle");
  const [statusMessage, setStatusMessage] = useState("钱包待命");
  const mountedRef = useRef(true);
  const restoreRequestRef = useRef(0);

  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  function applyRestoreState(restored: RestoreSessionResult, activeWallet?: WalletConnection | null) {
    startTransition(() => {
      setRestoreStatus(restored.status);
      if (restored.session) {
        setSession(restored.session);
        setWallet({
          walletAddress: restored.session.walletAddress,
          chainId: restored.session.chainId
        });
        setStatusMessage(restored.message);
        return;
      }

      setSession(null);
      if (activeWallet) {
        setWallet(activeWallet);
      }
      if (restored.status === "missing") {
        if (activeWallet?.chainId === TARGET_CHAIN.chainId) {
          setStatusMessage(`钱包已连接到 ${TARGET_CHAIN.chainName}（${activeWallet.chainId}）`);
        } else if (activeWallet) {
          setStatusMessage(`已切换到链 ${activeWallet.chainId}`);
        } else {
          setStatusMessage("钱包待命");
        }
        return;
      }
      setStatusMessage(restored.message);
    });
  }

  async function reconcile(activeWallet?: WalletConnection | null) {
    const requestID = restoreRequestRef.current + 1;
    restoreRequestRef.current = requestID;
    startTransition(() => setRestoring(true));
    try {
      const restored = await restoreStoredSession(activeWallet ?? null);
      if (!mountedRef.current || restoreRequestRef.current !== requestID) {
        return restored;
      }
      applyRestoreState(restored, activeWallet);
      return restored;
    } catch (error) {
      if (!mountedRef.current || restoreRequestRef.current !== requestID) {
        throw error;
      }
      startTransition(() => {
        setRestoreStatus("idle");
        setSession(null);
        setStatusMessage(formatError(error));
      });
      throw error;
    } finally {
      if (mountedRef.current && restoreRequestRef.current === requestID) {
        startTransition(() => setRestoring(false));
      }
    }
  }

  useEffect(() => {
    void (async () => {
      const activeWallet = await getWalletConnection().catch(() => null);
      await reconcile(activeWallet).catch(() => undefined);
    })();
  }, []);

  useEffect(() => {
    if (typeof window === "undefined" || !window.ethereum?.on) {
      return;
    }

    const handleAccountsChanged = (accounts: unknown) => {
      const walletAddress = Array.isArray(accounts) && typeof accounts[0] === "string" ? accounts[0].toLowerCase() : "";
      if (!walletAddress) {
        startTransition(() => {
          setWallet(null);
          setSession(null);
          setStatusMessage("钱包已断开，重新连接后可恢复交易密钥。");
        });
        return;
      }

      const nextWallet = { walletAddress, chainId: wallet?.chainId ?? 0 };
      startTransition(() => {
        setWallet(nextWallet);
      });
      void (async () => {
        try {
          const restored = await reconcile(nextWallet);
          if (!restored?.session && restored?.status === "missing") {
            startTransition(() => setStatusMessage("钱包已切换，当前没有可恢复的交易密钥。"));
          }
        } catch (error) {
          startTransition(() => setStatusMessage(formatError(error)));
        }
      })();
    };

    const handleChainChanged = (chainIdHex: unknown) => {
      if (typeof chainIdHex !== "string") return;
      const chainId = Number.parseInt(chainIdHex, 16);
      const nextWallet = wallet ? { ...wallet, chainId } : null;
      startTransition(() => {
        setWallet(nextWallet);
        setStatusMessage(`已切换到链 ${chainId}`);
      });
      if (!nextWallet) return;
      void (async () => {
        try {
          const restored = await reconcile(nextWallet);
          if (!restored?.session && restored?.status === "missing") {
            startTransition(() => setStatusMessage(`已切换到链 ${chainId}`));
          }
        } catch (error) {
          startTransition(() => setStatusMessage(formatError(error)));
        }
      })();
    };

    window.ethereum.on("accountsChanged", handleAccountsChanged);
    window.ethereum.on("chainChanged", handleChainChanged);

    return () => {
      window.ethereum?.removeListener?.("accountsChanged", handleAccountsChanged);
      window.ethereum?.removeListener?.("chainChanged", handleChainChanged);
    };
  }, [wallet]);

  async function handleConnect() {
    if (wallet && wallet.chainId === TARGET_CHAIN.chainId) {
      await reconcile(wallet).catch(() => undefined);
      return wallet;
    }

    setBusy("connect");
    setStatusMessage("连接钱包中...");
    try {
      const connection = await connectWallet();
      const restored = await reconcile(connection);
      if (!restored?.session && restored?.status === "missing") {
        startTransition(() => setStatusMessage(`钱包已连接到 ${TARGET_CHAIN.chainName}（${connection.chainId}）`));
      }
      return connection;
    } catch (error) {
      const message = formatError(error);
      startTransition(() => setStatusMessage(message));
      throw error;
    } finally {
      setBusy(null);
    }
  }

  async function handleCreateSession(activeWallet?: WalletConnection | null, options?: { forceRotate?: boolean }) {
    setBusy("session");
    setStatusMessage("正在授权交易密钥...");
    try {
      const readyWallet =
        activeWallet && activeWallet.chainId === TARGET_CHAIN.chainId
          ? activeWallet
          : wallet && wallet.chainId === TARGET_CHAIN.chainId
            ? wallet
            : await handleConnect();
      if (!options?.forceRotate) {
        const restored = await reconcile(readyWallet ?? null);
        if (restored?.session) {
          return restored.session;
        }
      }
      const created = await authorizeSession(readyWallet ?? undefined);
      startTransition(() => {
        setSession(created);
        setWallet({ walletAddress: created.walletAddress, chainId: created.chainId });
        setRestoreStatus("restored");
        setStatusMessage("交易密钥已开启");
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
      setStatusMessage("交易密钥已就绪");
      return session;
    }

    const activeWallet = wallet ?? (await handleConnect());
    if (!activeWallet) {
      return null;
    }

    const restored = await reconcile(activeWallet).catch(() => null);
    if (restored?.session) {
      return restored.session;
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
    void clearStoredSession(session);
    startTransition(() => {
      setSession(null);
      setRestoreStatus("missing");
      setStatusMessage("本地交易密钥已清空");
    });
  }

  async function handleRevokeCurrentSession() {
    if (!session) {
      handleClear();
      return;
    }
    await revokeRemoteSession(session.sessionId);
    void clearStoredSession(session);
    startTransition(() => {
      setSession(null);
      setRestoreStatus("revoked");
      setStatusMessage("交易密钥已撤销");
    });
  }

  return (
    <TradingSessionContext.Provider
      value={{
        wallet,
        session,
        busy,
        restoring,
        restoreStatus,
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
