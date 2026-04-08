import { encodeFunctionData, erc20Abi, keccak256, maxUint256, numberToHex, parseEther, parseUnits, stringToHex } from "viem";

const TARGET_CHAIN_ID = Number(process.env.NEXT_PUBLIC_CHAIN_ID ?? "97");
const TARGET_CHAIN_NAME = process.env.NEXT_PUBLIC_CHAIN_NAME ?? "BSC Testnet";
const TARGET_VAULT_ADDRESS = process.env.NEXT_PUBLIC_VAULT_ADDRESS ?? "";
const TARGET_COLLATERAL_TOKEN_ADDRESS = process.env.NEXT_PUBLIC_COLLATERAL_TOKEN_ADDRESS ?? "";
const TARGET_COLLATERAL_SYMBOL = process.env.NEXT_PUBLIC_COLLATERAL_SYMBOL ?? "USDT";
const TARGET_COLLATERAL_DECIMALS = Number(process.env.NEXT_PUBLIC_COLLATERAL_DECIMALS ?? "6");
const TARGET_EXPLORER_URL = process.env.NEXT_PUBLIC_CHAIN_EXPLORER_URL ?? "";
const TARGET_RPC_URL = process.env.NEXT_PUBLIC_CHAIN_RPC_URL ?? "https://data-seed-prebsc-1-s1.bnbchain.org:8545";
const TARGET_NATIVE_CURRENCY_NAME = process.env.NEXT_PUBLIC_NATIVE_CURRENCY_NAME ?? (TARGET_CHAIN_ID === 31337 ? "Ethereum" : "BNB");
const TARGET_NATIVE_CURRENCY_SYMBOL = process.env.NEXT_PUBLIC_NATIVE_CURRENCY_SYMBOL ?? (TARGET_CHAIN_ID === 31337 ? "ETH" : "tBNB");
const TARGET_NATIVE_CURRENCY_DECIMALS = Number(process.env.NEXT_PUBLIC_NATIVE_CURRENCY_DECIMALS ?? "18");

const vaultAbi = [
  {
    inputs: [{ internalType: "uint256", name: "amount", type: "uint256" }],
    name: "deposit",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [],
    name: "depositNative",
    outputs: [],
    stateMutability: "payable",
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

function parseTokenAmount(amount: string) {
  if (!amount.trim()) {
    throw new Error("Amount is required");
  }
  return parseUnits(amount, TARGET_COLLATERAL_DECIMALS);
}

async function readVaultAllowance(walletAddress: string): Promise<bigint> {
  ensureConfigured();
  const ethereum = ensureEthereum();
  const data = encodeFunctionData({
    abi: erc20Abi,
    functionName: "allowance",
    args: [walletAddress as `0x${string}`, TARGET_VAULT_ADDRESS as `0x${string}`]
  });
  const raw = (await ethereum.request({
    method: "eth_call",
    params: [{ to: TARGET_COLLATERAL_TOKEN_ADDRESS, data }, "latest"]
  })) as string;
  if (typeof raw !== "string" || !raw.startsWith("0x")) {
    throw new Error("无法读取代币授权额度");
  }
  return BigInt(raw);
}

export async function getVaultAllowance(walletAddress: string) {
  return readVaultAllowance(walletAddress);
}

function sleep(ms: number) {
  return new Promise<void>((resolve) => {
    setTimeout(resolve, ms);
  });
}

/**
 * 额度不足时对 Vault 做 **无限额 approve**（一次上链），之后每次充值通常只需签 deposit 一笔。
 * 首次在本站充值仍可能连续两次签名：无限授权 + 充值（ERC-20 无法把二者合并为一笔，除非代币支持 permit 且合约支持 permit 充值）。
 */
export async function depositCollateralWithAutoApprove(walletAddress: string, amount: string) {
  ensureConfigured();
  const needed = parseTokenAmount(amount);
  await ensureTargetChain();
  let allowance = await readVaultAllowance(walletAddress);
  let approveTxHash: string | undefined;
  if (allowance < needed) {
    approveTxHash = await approveVaultUnlimited(walletAddress);
    for (let i = 0; i < 60; i += 1) {
      await sleep(1000);
      allowance = await readVaultAllowance(walletAddress);
      if (allowance >= needed) break;
    }
    if (allowance < needed) {
      throw new Error("授权交易确认超时，请在浏览器中确认上一笔授权已成功，然后重试充值。");
    }
  }
  const depositTxHash = await depositToVault(walletAddress, amount);
  return { approveTxHash, depositTxHash };
}

/** 对 Vault 授予最大额度，避免每次充值额度不足时再签 approve。 */
export async function approveVaultUnlimited(walletAddress: string) {
  ensureConfigured();
  const ethereum = ensureEthereum();
  const data = encodeFunctionData({
    abi: erc20Abi,
    functionName: "approve",
    args: [TARGET_VAULT_ADDRESS as `0x${string}`, maxUint256]
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

export async function depositNativeToVault(walletAddress: string, nativeAmount: string) {
  ensureConfigured();
  const ethereum = ensureEthereum();
  const data = encodeFunctionData({
    abi: vaultAbi,
    functionName: "depositNative"
  });

  return (await ethereum.request({
    method: "eth_sendTransaction",
    params: [
      {
        from: walletAddress,
        to: TARGET_VAULT_ADDRESS,
        data,
        value: numberToHex(parseEther(nativeAmount))
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
