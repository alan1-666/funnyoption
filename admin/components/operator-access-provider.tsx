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
}

const OperatorAccessContext = createContext<OperatorAccessContextValue | null>(null);
const ALLOWLISTED_WALLETS = parseOperatorWallets(process.env.NEXT_PUBLIC_OPERATOR_WALLETS ?? "");

function formatError(error: unknown) {
  return error instanceof Error ? error.message : "Unknown wallet error";
}

export function OperatorAccessProvider({ children }: { children: React.ReactNode }) {
  const [wallet, setWallet] = useState<WalletConnection | null>(null);
  const [busy, setBusy] = useState<"connect" | "sign" | null>(null);
  const [statusMessage, setStatusMessage] = useState(
    ALLOWLISTED_WALLETS.length > 0
      ? "Connect an allowlisted wallet to unlock operator actions"
      : "No operator wallets are configured for this admin service"
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
          setStatusMessage(
            ALLOWLISTED_WALLETS.includes(connection.walletAddress)
              ? "Allowlisted operator wallet detected"
              : "Wallet detected but not allowlisted for operator actions"
          );
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
          setStatusMessage("Wallet disconnected");
          return;
        }

        setWallet((current) => ({ walletAddress, chainId: current?.chainId ?? 0 }));
        setStatusMessage(
          ALLOWLISTED_WALLETS.includes(walletAddress)
            ? "Allowlisted operator wallet detected"
            : "Wallet detected but not allowlisted for operator actions"
        );
      });
    };

    const handleChainChanged = (chainIdHex: unknown) => {
      if (typeof chainIdHex !== "string") return;
      const chainId = Number.parseInt(chainIdHex, 16);
      startTransition(() => {
        setWallet((current) => (current ? { ...current, chainId } : current));
        setStatusMessage(`Wallet connected on chain ${chainId}`);
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
    if (wallet) {
      setStatusMessage(
        ALLOWLISTED_WALLETS.includes(wallet.walletAddress)
          ? `Operator wallet ready on chain ${wallet.chainId}`
          : "Wallet connected but not allowlisted for operator actions"
      );
      return wallet;
    }

    setBusy("connect");
    setStatusMessage("Connecting wallet...");
    try {
      const connection = await connectWallet();
      startTransition(() => {
        setWallet(connection);
        setStatusMessage(
          ALLOWLISTED_WALLETS.includes(connection.walletAddress)
            ? `Operator wallet linked on chain ${connection.chainId}`
            : "Wallet connected but not allowlisted for operator actions"
        );
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
    const activeWallet = wallet ?? (await handleConnect());
    if (!activeWallet) {
      return null;
    }
    if (!ALLOWLISTED_WALLETS.includes(activeWallet.walletAddress)) {
      setStatusMessage("Wallet is not allowlisted for operator actions");
      return null;
    }

    setBusy("sign");
    setStatusMessage("Awaiting operator wallet signature...");
    try {
      const signature = await signPersonalMessage(message, activeWallet.walletAddress);
      startTransition(() => setStatusMessage("Operator wallet signature captured"));
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
        signBootstrapMarket: handleSignBootstrapMarket
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
