"use client";

import { createContext, startTransition, useContext, useEffect, useMemo, useState } from "react";

import {
  buildBootstrapMarketMessage,
  buildCreateMarketMessage,
  buildResolveMarketMessage,
  parseOperatorWallets,
  type BootstrapMarketDraft,
  type CreateMarketDraft,
  type ResolveMarketDraft,
  type SignedOperatorAction
} from "@/lib/operator-auth";
import { connectWallet, getWalletConnection, signPersonalMessage, type WalletConnection } from "@/lib/operator-wallet";

interface OperatorAccessContextValue {
  wallet: WalletConnection | null;
  busy: "connect" | "sign" | null;
  statusMessage: string;
  allowlistedWallets: string[];
  isWalletAllowlisted: boolean;
  connect: () => Promise<WalletConnection | null>;
  signCreateMarket: (market: CreateMarketDraft) => Promise<SignedOperatorAction | null>;
  signResolveMarket: (market: ResolveMarketDraft) => Promise<SignedOperatorAction | null>;
  signBootstrapMarket: (bootstrap: BootstrapMarketDraft) => Promise<SignedOperatorAction | null>;
  signGenericMessage: (message: string) => Promise<SignedOperatorAction | null>;
}

const OperatorAccessContext = createContext<OperatorAccessContextValue | null>(null);
const ALLOWLISTED_WALLETS = parseOperatorWallets(process.env.NEXT_PUBLIC_OPERATOR_WALLETS ?? "");
const TARGET_CHAIN_ID = Number(process.env.NEXT_PUBLIC_CHAIN_ID ?? "97");
const TARGET_CHAIN_NAME = process.env.NEXT_PUBLIC_CHAIN_NAME ?? "BSC Testnet";

function formatError(error: unknown) {
  return error instanceof Error ? error.message : "钱包操作失败";
}

function buildWalletStatusMessage(connection: WalletConnection) {
  const allowlisted = ALLOWLISTED_WALLETS.includes(connection.walletAddress);
  if (connection.chainId !== TARGET_CHAIN_ID) {
    return `当前连接的是链 ${connection.chainId}，请切换到 ${TARGET_CHAIN_NAME}（${TARGET_CHAIN_ID}）。`;
  }
  if (allowlisted) {
    return `运营钱包已就绪：${TARGET_CHAIN_NAME}（${TARGET_CHAIN_ID}）`;
  }
  return "钱包已连接，但不在运营白名单内";
}

export function OperatorAccessProvider({ children }: { children: React.ReactNode }) {
  const [wallet, setWallet] = useState<WalletConnection | null>(null);
  const [busy, setBusy] = useState<"connect" | "sign" | null>(null);
  const [statusMessage, setStatusMessage] = useState(
    ALLOWLISTED_WALLETS.length > 0
      ? "请连接白名单运营钱包"
      : "当前后台还没有配置运营钱包白名单"
  );

  const isWalletAllowlisted = useMemo(() => {
    if (!wallet) {
      return false;
    }
    return ALLOWLISTED_WALLETS.includes(wallet.walletAddress);
  }, [wallet]);

  useEffect(() => {
    getWalletConnection()
      .then((connection) => {
        if (!connection) return;
        startTransition(() => {
          setWallet(connection);
          setStatusMessage(buildWalletStatusMessage(connection));
        });
      })
      .catch(() => undefined);
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
          setStatusMessage("钱包已断开");
          return;
        }

        setWallet((current) => {
          const next = { walletAddress, chainId: current?.chainId ?? 0 };
          setStatusMessage(buildWalletStatusMessage(next));
          return next;
        });
      });
    };

    const handleChainChanged = (chainIdHex: unknown) => {
      if (typeof chainIdHex !== "string") return;
      const chainId = Number.parseInt(chainIdHex, 16);
      startTransition(() => {
        setWallet((current) => {
          if (!current) {
            return current;
          }
          const next = { ...current, chainId };
          setStatusMessage(buildWalletStatusMessage(next));
          return next;
        });
      });
    };

    window.ethereum.on("accountsChanged", handleAccountsChanged);
    window.ethereum.on("chainChanged", handleChainChanged);

    return () => {
      window.ethereum?.removeListener?.("accountsChanged", handleAccountsChanged);
      window.ethereum?.removeListener?.("chainChanged", handleChainChanged);
    };
  }, []);

  async function handleConnect() {
    if (wallet && wallet.chainId === TARGET_CHAIN_ID) {
      setStatusMessage(
        ALLOWLISTED_WALLETS.includes(wallet.walletAddress)
          ? `运营钱包已连接到 ${TARGET_CHAIN_NAME}（${wallet.chainId}）`
          : "钱包已连接，但不在运营白名单内"
      );
      return wallet;
    }

    setBusy("connect");
    setStatusMessage("连接钱包中...");
    try {
      const connection = await connectWallet();
      startTransition(() => {
        setWallet(connection);
        setStatusMessage(buildWalletStatusMessage(connection));
      });
      return connection;
    } catch (error) {
      startTransition(() => setStatusMessage(formatError(error)));
      throw error;
    } finally {
      setBusy(null);
    }
  }

  async function signAuthorizedMessage(message: string, requestedAt: number) {
    const activeWallet = !wallet || wallet.chainId !== TARGET_CHAIN_ID ? await handleConnect() : wallet;
    if (!activeWallet) {
      return null;
    }
    if (activeWallet.chainId !== TARGET_CHAIN_ID) {
      setStatusMessage(`请先切换到 ${TARGET_CHAIN_NAME}（${TARGET_CHAIN_ID}）`);
      return null;
    }
    if (!ALLOWLISTED_WALLETS.includes(activeWallet.walletAddress)) {
      setStatusMessage("当前钱包不在运营白名单内");
      return null;
    }

    setBusy("sign");
    setStatusMessage("等待运营钱包签名...");
    try {
      const signature = await signPersonalMessage(message, activeWallet.walletAddress);
      startTransition(() => setStatusMessage("签名已完成"));
      return {
        walletAddress: activeWallet.walletAddress,
        requestedAt,
        signature
      };
    } catch (error) {
      startTransition(() => setStatusMessage(formatError(error)));
      throw error;
    } finally {
      setBusy(null);
    }
  }

  async function handleSignCreateMarket(market: CreateMarketDraft) {
    const activeWallet = wallet ?? (await handleConnect());
    if (!activeWallet || !ALLOWLISTED_WALLETS.includes(activeWallet.walletAddress)) {
      return null;
    }
    const requestedAt = Date.now();
    return signAuthorizedMessage(
      buildCreateMarketMessage({
        walletAddress: activeWallet.walletAddress,
        market,
        requestedAt
      }),
      requestedAt
    );
  }

  async function handleSignResolveMarket(market: ResolveMarketDraft) {
    const activeWallet = wallet ?? (await handleConnect());
    if (!activeWallet || !ALLOWLISTED_WALLETS.includes(activeWallet.walletAddress)) {
      return null;
    }
    const requestedAt = Date.now();
    return signAuthorizedMessage(
      buildResolveMarketMessage({
        walletAddress: activeWallet.walletAddress,
        market,
        requestedAt
      }),
      requestedAt
    );
  }

  async function handleSignBootstrapMarket(bootstrap: BootstrapMarketDraft) {
    const activeWallet = wallet ?? (await handleConnect());
    if (!activeWallet || !ALLOWLISTED_WALLETS.includes(activeWallet.walletAddress)) {
      return null;
    }
    const requestedAt = Date.now();
    return signAuthorizedMessage(
      buildBootstrapMarketMessage({
        walletAddress: activeWallet.walletAddress,
        bootstrap,
        requestedAt
      }),
      requestedAt
    );
  }

  async function handleSignGenericMessage(message: string) {
    const activeWallet = wallet ?? (await handleConnect());
    if (!activeWallet || !ALLOWLISTED_WALLETS.includes(activeWallet.walletAddress)) {
      return null;
    }
    const requestedAt = Date.now();
    return signAuthorizedMessage(message, requestedAt);
  }

  return (
    <OperatorAccessContext.Provider
      value={{
        wallet,
        busy,
        statusMessage,
        allowlistedWallets: ALLOWLISTED_WALLETS,
        isWalletAllowlisted,
        connect: handleConnect,
        signCreateMarket: handleSignCreateMarket,
        signResolveMarket: handleSignResolveMarket,
        signBootstrapMarket: handleSignBootstrapMarket,
        signGenericMessage: handleSignGenericMessage
      }}
    >
      {children}
    </OperatorAccessContext.Provider>
  );
}

export function useOperatorAccess() {
  const context = useContext(OperatorAccessContext);
  if (!context) {
    throw new Error("useOperatorAccess must be used within OperatorAccessProvider");
  }
  return context;
}
