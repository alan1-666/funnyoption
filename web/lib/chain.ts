import { encodeFunctionData, erc20Abi, keccak256, parseUnits, stringToHex } from "viem";

const TARGET_CHAIN_ID = Number(process.env.NEXT_PUBLIC_CHAIN_ID ?? "97");
const TARGET_CHAIN_NAME = process.env.NEXT_PUBLIC_CHAIN_NAME ?? "BSC Testnet";
const TARGET_VAULT_ADDRESS = process.env.NEXT_PUBLIC_VAULT_ADDRESS ?? "";
const TARGET_COLLATERAL_TOKEN_ADDRESS = process.env.NEXT_PUBLIC_COLLATERAL_TOKEN_ADDRESS ?? "";
const TARGET_COLLATERAL_SYMBOL = process.env.NEXT_PUBLIC_COLLATERAL_SYMBOL ?? "USDT";
const TARGET_COLLATERAL_DECIMALS = Number(process.env.NEXT_PUBLIC_COLLATERAL_DECIMALS ?? "6");
const TARGET_EXPLORER_URL = process.env.NEXT_PUBLIC_CHAIN_EXPLORER_URL ?? "https://testnet.bscscan.com";
const TARGET_RPC_URL = process.env.NEXT_PUBLIC_CHAIN_RPC_URL ?? "https://data-seed-prebsc-1-s1.bnbchain.org:8545";

const vaultAbi = [
  {
    inputs: [{ internalType: "uint256", name: "amount", type: "uint256" }],
    name: "deposit",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [
      { internalType: "bytes32", name: "withdrawalId", type: "bytes32" },
      { internalType: "uint256", name: "amount", type: "uint256" },
      { internalType: "address", name: "recipient", type: "address" }
    ],
    name: "queueWithdrawal",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  }
] as const;

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

function ensureConfigured() {
  if (!TARGET_VAULT_ADDRESS) throw new Error("NEXT_PUBLIC_VAULT_ADDRESS is not configured");
  if (!TARGET_COLLATERAL_TOKEN_ADDRESS) throw new Error("NEXT_PUBLIC_COLLATERAL_TOKEN_ADDRESS is not configured");
}

export function getChainMeta() {
  return {
    chainId: TARGET_CHAIN_ID,
    chainName: TARGET_CHAIN_NAME,
    vaultAddress: TARGET_VAULT_ADDRESS,
    tokenAddress: TARGET_COLLATERAL_TOKEN_ADDRESS,
    tokenSymbol: TARGET_COLLATERAL_SYMBOL,
    tokenDecimals: TARGET_COLLATERAL_DECIMALS,
    explorerUrl: TARGET_EXPLORER_URL
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
          nativeCurrency: { name: "BNB", symbol: "tBNB", decimals: 18 },
          rpcUrls: [TARGET_RPC_URL],
          blockExplorerUrls: [TARGET_EXPLORER_URL]
        }
      ]
    });
  }
}

function parseTokenAmount(amount: string) {
  if (!amount.trim()) {
    throw new Error("Amount is required");
  }
  return parseUnits(amount, TARGET_COLLATERAL_DECIMALS);
}

export async function approveVault(walletAddress: string, amount: string) {
  ensureConfigured();
  const ethereum = ensureEthereum();
  const data = encodeFunctionData({
    abi: erc20Abi,
    functionName: "approve",
    args: [TARGET_VAULT_ADDRESS as `0x${string}`, parseTokenAmount(amount)]
  });

  return (await ethereum.request({
    method: "eth_sendTransaction",
    params: [
      {
        from: walletAddress,
        to: TARGET_COLLATERAL_TOKEN_ADDRESS,
        data
      }
    ]
  })) as string;
}

export async function depositToVault(walletAddress: string, amount: string) {
  ensureConfigured();
  const ethereum = ensureEthereum();
  const data = encodeFunctionData({
    abi: vaultAbi,
    functionName: "deposit",
    args: [parseTokenAmount(amount)]
  });

  return (await ethereum.request({
    method: "eth_sendTransaction",
    params: [
      {
        from: walletAddress,
        to: TARGET_VAULT_ADDRESS,
        data
      }
    ]
  })) as string;
}

export async function queueWithdrawal(walletAddress: string, amount: string, recipientAddress: string) {
  ensureConfigured();
  const ethereum = ensureEthereum();
  const withdrawalId = keccak256(
    stringToHex(`${walletAddress.toLowerCase()}:${recipientAddress.toLowerCase()}:${amount}:${Date.now()}`)
  );
  const data = encodeFunctionData({
    abi: vaultAbi,
    functionName: "queueWithdrawal",
    args: [withdrawalId, parseTokenAmount(amount), recipientAddress as `0x${string}`]
  });

  const txHash = (await ethereum.request({
    method: "eth_sendTransaction",
    params: [
      {
        from: walletAddress,
        to: TARGET_VAULT_ADDRESS,
        data
      }
    ]
  })) as string;

  return { txHash, withdrawalId };
}
