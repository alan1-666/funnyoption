export interface WalletConnection {
  walletAddress: string;
  chainId: number;
}

declare global {
  interface Window {
    ethereum?: {
      request(args: { method: string; params?: unknown[] | object }): Promise<unknown>;
      on?(event: string, handler: (...args: unknown[]) => void): void;
      removeListener?(event: string, handler: (...args: unknown[]) => void): void;
    };
  }
}

function toHex(value: string) {
  return `0x${Array.from(new TextEncoder().encode(value), (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
}

function normalizeAddress(value: string) {
  return value.trim().toLowerCase();
}

function ensureEthereum() {
  if (typeof window === "undefined" || !window.ethereum) {
    throw new Error("MetaMask or an EIP-1193 wallet is required");
  }
  return window.ethereum;
}

export async function connectWallet(): Promise<WalletConnection> {
  const ethereum = ensureEthereum();
  const accounts = (await ethereum.request({ method: "eth_requestAccounts" })) as string[];
  const chainIdHex = (await ethereum.request({ method: "eth_chainId" })) as string;
  const walletAddress = normalizeAddress(accounts[0] ?? "");

  if (!walletAddress) {
    throw new Error("No wallet account returned");
  }

  return {
    walletAddress,
    chainId: Number.parseInt(chainIdHex, 16)
  };
}

export async function getWalletConnection(): Promise<WalletConnection | null> {
  const ethereum = ensureEthereum();
  const accounts = (await ethereum.request({ method: "eth_accounts" })) as string[];
  if (!accounts?.length) return null;
  const chainIdHex = (await ethereum.request({ method: "eth_chainId" })) as string;
  return {
    walletAddress: normalizeAddress(accounts[0]),
    chainId: Number.parseInt(chainIdHex, 16)
  };
}

export async function signPersonalMessage(message: string, walletAddress: string) {
  const ethereum = ensureEthereum();
  return (await ethereum.request({
    method: "personal_sign",
    params: [toHex(message), walletAddress]
  })) as string;
}
