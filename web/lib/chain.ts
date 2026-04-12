const TARGET_CHAIN_ID = Number(process.env.NEXT_PUBLIC_CHAIN_ID ?? "97");
const TARGET_CHAIN_NAME = process.env.NEXT_PUBLIC_CHAIN_NAME ?? "BSC Testnet";
const TARGET_VAULT_ADDRESS = process.env.NEXT_PUBLIC_VAULT_ADDRESS ?? "";
const TARGET_COLLATERAL_SYMBOL = process.env.NEXT_PUBLIC_COLLATERAL_SYMBOL ?? "USDT";
const TARGET_COLLATERAL_DECIMALS = Number(process.env.NEXT_PUBLIC_COLLATERAL_DECIMALS ?? "6");
const TARGET_EXPLORER_URL = process.env.NEXT_PUBLIC_CHAIN_EXPLORER_URL ?? "";
const TARGET_RPC_URL = process.env.NEXT_PUBLIC_CHAIN_RPC_URL ?? "https://data-seed-prebsc-1-s1.bnbchain.org:8545";
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

function ensureEthereum() {
  if (typeof window === "undefined" || !window.ethereum) {
    throw new Error("MetaMask or a compatible wallet is required");
  }
  return window.ethereum;
}

export function getChainMeta() {
  return {
    chainId: TARGET_CHAIN_ID,
    chainName: TARGET_CHAIN_NAME,
    vaultAddress: TARGET_VAULT_ADDRESS,
    tokenSymbol: TARGET_COLLATERAL_SYMBOL,
    tokenDecimals: TARGET_COLLATERAL_DECIMALS,
    explorerUrl: TARGET_EXPLORER_URL,
    nativeCurrencyName: TARGET_NATIVE_CURRENCY_NAME,
    nativeCurrencySymbol: TARGET_NATIVE_CURRENCY_SYMBOL,
    nativeCurrencyDecimals: TARGET_NATIVE_CURRENCY_DECIMALS
  };
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
