export interface WalletConnection {
  walletAddress: string;
  chainId: number;
}

const TARGET_CHAIN_ID = Number(process.env.NEXT_PUBLIC_CHAIN_ID ?? "97");
const TARGET_CHAIN_NAME = process.env.NEXT_PUBLIC_CHAIN_NAME ?? "BSC Testnet";
const TARGET_RPC_URL = process.env.NEXT_PUBLIC_CHAIN_RPC_URL ?? "https://data-seed-prebsc-1-s1.bnbchain.org:8545";
const TARGET_EXPLORER_URL = process.env.NEXT_PUBLIC_CHAIN_EXPLORER_URL ?? "";
const TARGET_NATIVE_CURRENCY_NAME = process.env.NEXT_PUBLIC_NATIVE_CURRENCY_NAME ?? (TARGET_CHAIN_ID === 31337 ? "Ethereum" : "BNB");
const TARGET_NATIVE_CURRENCY_SYMBOL = process.env.NEXT_PUBLIC_NATIVE_CURRENCY_SYMBOL ?? (TARGET_CHAIN_ID === 31337 ? "ETH" : "tBNB");
const TARGET_NATIVE_CURRENCY_DECIMALS = Number(process.env.NEXT_PUBLIC_NATIVE_CURRENCY_DECIMALS ?? "18");

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
    throw new Error("请先安装 MetaMask 或其他兼容钱包");
  }
  return window.ethereum;
}

function toChainHex(chainId: number) {
  return `0x${chainId.toString(16)}`;
}

export async function ensureTargetChain() {
  const ethereum = ensureEthereum();
  const targetHex = toChainHex(TARGET_CHAIN_ID);

  try {
    await ethereum.request({
      method: "wallet_switchEthereumChain",
      params: [{ chainId: targetHex }]
    });
  } catch (error) {
    const code = (error as { code?: number })?.code;
    if (code !== 4902) throw error;

    await ethereum.request({
      method: "wallet_addEthereumChain",
      params: [
        {
          chainId: targetHex,
          chainName: TARGET_CHAIN_NAME,
          nativeCurrency: {
            name: TARGET_NATIVE_CURRENCY_NAME,
            symbol: TARGET_NATIVE_CURRENCY_SYMBOL,
            decimals: TARGET_NATIVE_CURRENCY_DECIMALS
          },
          rpcUrls: [TARGET_RPC_URL],
          ...(TARGET_EXPLORER_URL ? { blockExplorerUrls: [TARGET_EXPLORER_URL] } : {})
        }
      ]
    });
  }
}

export async function connectWallet(): Promise<WalletConnection> {
  await ensureTargetChain();
  const ethereum = ensureEthereum();
  const accounts = (await ethereum.request({ method: "eth_requestAccounts" })) as string[];
  const chainIdHex = (await ethereum.request({ method: "eth_chainId" })) as string;
  const walletAddress = normalizeAddress(accounts[0] ?? "");

  if (!walletAddress) {
    throw new Error("没有读取到钱包账户");
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
