import * as ed from "@noble/ed25519";
import { ensureTargetChain, getChainMeta } from "@/lib/chain";

const STORAGE_KEY_PREFIX = "funnyoption:trading-key:v2:meta:";
const KEY_DB_NAME = "funnyoption-trading-key";
const KEY_STORE_NAME = "keys";
const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";
const DEFAULT_SCOPE = "TRADE";
const DEFAULT_KEY_SCHEME = "ED25519";
const DEFAULT_WALLET_SIGNATURE_STANDARD = "EIP712_V4";
const TARGET_CHAIN = getChainMeta();
const TARGET_CHAIN_ID = TARGET_CHAIN.chainId;
const TARGET_VAULT_ADDRESS = normalizeAddress(TARGET_CHAIN.vaultAddress);

export interface WalletConnection {
  walletAddress: string;
  chainId: number;
}

export interface SessionRecord {
  userId: number;
  walletAddress: string;
  chainId: number;
  vaultAddress: string;
  sessionId: string;
  sessionPublicKey: string;
  lastOrderNonce: number;
  expiresAt: number;
  issuedAt: number;
  scope: string;
  status: string;
}

export interface OrderSignaturePayload {
  sessionId: string;
  walletAddress: string;
  sessionSignature: string;
  orderNonce: number;
  requestedAt: number;
  userId: number;
}

export interface RemoteSession {
  session_id: string;
  user_id: number;
  wallet_address: string;
  session_public_key: string;
  scope: string;
  chain_id: number;
  vault_address?: string;
  session_nonce: string;
  last_order_nonce: number;
  status: string;
  issued_at: number;
  expires_at: number;
  revoked_at: number;
  created_at: number;
  updated_at: number;
}

interface TradingKeyChallenge {
  challenge_id: string;
  challenge: string;
  challenge_expires_at: number;
}

export type RestoreSessionStatus =
  | "missing"
  | "wallet_required"
  | "restored"
  | "wallet_mismatch"
  | "chain_mismatch"
  | "vault_mismatch"
  | "missing_private_key"
  | "expired"
  | "revoked"
  | "rotated"
  | "remote_missing"
  | "remote_mismatch";

export interface RestoreSessionResult {
  session: SessionRecord | null;
  status: RestoreSessionStatus;
  message: string;
}

interface RestoreSessionOptions {
  allowWalletProbe?: boolean;
}

interface RemoteChainTask {
  id: number;
  biz_type: string;
  ref_id: string;
  chain_name: string;
  network_name: string;
  wallet_address: string;
  tx_hash: string;
  status: string;
  payload?: {
    event_id?: string;
    market_id?: number;
    user_id?: number;
    payout_asset?: string;
    payout_amount?: number;
    recipient_address?: string;
    wallet_address?: string;
    [key: string]: unknown;
  } | null;
  error_message: string;
  attempt_count: number;
  created_at: number;
  updated_at: number;
}

interface StoredPrivateKeyRecord {
  storageKey: string;
  privateKey: string;
  updatedAt: number;
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

function toHex(bytes: Uint8Array) {
  return `0x${Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
}

function hexToBytes(value: string) {
  const normalized = value.startsWith("0x") ? value.slice(2) : value;
  const bytes = new Uint8Array(normalized.length / 2);
  for (let index = 0; index < normalized.length; index += 2) {
    bytes[index / 2] = Number.parseInt(normalized.slice(index, index + 2), 16);
  }
  return bytes;
}

function normalizeAddress(value: string) {
  return value.trim().toLowerCase();
}

function normalizeOptionalAddress(value?: string | null) {
  return typeof value === "string" ? normalizeAddress(value) : "";
}

function normalizeOrderField(value: string) {
  return value.trim().toUpperCase();
}

function normalizeSessionStatus(value: string) {
  return value.trim().toUpperCase();
}

function ensureEthereum() {
  if (typeof window === "undefined" || !window.ethereum) {
    throw new Error("MetaMask or an EIP-1193 wallet is required");
  }
  return window.ethereum;
}

function ensureTargetVaultAddress() {
  if (!TARGET_VAULT_ADDRESS) {
    throw new Error("NEXT_PUBLIC_VAULT_ADDRESS is not configured");
  }
  return TARGET_VAULT_ADDRESS;
}

function buildStorageKey(walletAddress: string, chainId: number, vaultAddress: string) {
  return `${normalizeAddress(walletAddress)}:${chainId}:${normalizeAddress(vaultAddress)}`;
}

function buildSessionRecord(input: RemoteSession): SessionRecord {
  const remoteVaultAddress = normalizeOptionalAddress(input.vault_address);
  return {
    userId: input.user_id,
    walletAddress: normalizeAddress(input.wallet_address),
    chainId: input.chain_id,
    vaultAddress: remoteVaultAddress || ensureTargetVaultAddress(),
    sessionId: input.session_id,
    sessionPublicKey: normalizeAddress(input.session_public_key),
    lastOrderNonce: input.last_order_nonce,
    expiresAt: input.expires_at,
    issuedAt: input.issued_at,
    scope: input.scope,
    status: input.status
  };
}

function buildMetadataStorageKey(walletAddress: string, chainId: number, vaultAddress: string) {
  return `${STORAGE_KEY_PREFIX}${buildStorageKey(walletAddress, chainId, vaultAddress)}`;
}

function saveStoredSession(session: SessionRecord) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(buildMetadataStorageKey(session.walletAddress, session.chainId, session.vaultAddress), JSON.stringify(session));
}

function parseStoredSession(raw: string | null) {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as SessionRecord;
    if (!parsed?.sessionId || !parsed?.sessionPublicKey || !parsed?.walletAddress) {
      return null;
    }
    return {
      ...parsed,
      walletAddress: normalizeAddress(parsed.walletAddress),
      vaultAddress: normalizeAddress(parsed.vaultAddress || ""),
      sessionPublicKey: normalizeAddress(parsed.sessionPublicKey)
    };
  } catch {
    return null;
  }
}

function loadStoredSessionMetadata(walletAddress: string, chainId: number, vaultAddress: string) {
  if (typeof window === "undefined") return null;
  return parseStoredSession(window.localStorage.getItem(buildMetadataStorageKey(walletAddress, chainId, vaultAddress)));
}

function listStoredSessionMetadata() {
  if (typeof window === "undefined") return [] as SessionRecord[];
  const items: SessionRecord[] = [];
  for (let index = 0; index < window.localStorage.length; index += 1) {
    const key = window.localStorage.key(index);
    if (!key?.startsWith(STORAGE_KEY_PREFIX)) continue;
    const parsed = parseStoredSession(window.localStorage.getItem(key));
    if (parsed) {
      items.push(parsed);
    }
  }
  return items;
}

function openKeyDatabase() {
  if (typeof window === "undefined" || !window.indexedDB) {
    throw new Error("Browser secure storage is unavailable");
  }

  return new Promise<IDBDatabase>((resolve, reject) => {
    const request = window.indexedDB.open(KEY_DB_NAME, 1);
    request.onupgradeneeded = () => {
      const database = request.result;
      if (!database.objectStoreNames.contains(KEY_STORE_NAME)) {
        database.createObjectStore(KEY_STORE_NAME, { keyPath: "storageKey" });
      }
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error ?? new Error("Failed to open browser key storage"));
  });
}

async function putStoredPrivateKey(storageKey: string, privateKey: string) {
  const database = await openKeyDatabase();
  try {
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction(KEY_STORE_NAME, "readwrite");
      const store = transaction.objectStore(KEY_STORE_NAME);
      store.put({
        storageKey,
        privateKey,
        updatedAt: Date.now()
      } satisfies StoredPrivateKeyRecord);
      transaction.oncomplete = () => resolve();
      transaction.onabort = () => reject(transaction.error ?? new Error("Failed to persist private key"));
      transaction.onerror = () => reject(transaction.error ?? new Error("Failed to persist private key"));
    });
  } finally {
    database.close();
  }
}

async function getStoredPrivateKey(storageKey: string) {
  const database = await openKeyDatabase();
  try {
    return await new Promise<string | null>((resolve, reject) => {
      const transaction = database.transaction(KEY_STORE_NAME, "readonly");
      const store = transaction.objectStore(KEY_STORE_NAME);
      const request = store.get(storageKey);
      request.onsuccess = () => {
        const result = request.result as StoredPrivateKeyRecord | undefined;
        resolve(result?.privateKey ?? null);
      };
      request.onerror = () => reject(request.error ?? new Error("Failed to read private key"));
      transaction.onabort = () => reject(transaction.error ?? new Error("Failed to read private key"));
      transaction.onerror = () => reject(transaction.error ?? new Error("Failed to read private key"));
    });
  } finally {
    database.close();
  }
}

async function deleteStoredPrivateKey(storageKey: string) {
  const database = await openKeyDatabase();
  try {
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction(KEY_STORE_NAME, "readwrite");
      const store = transaction.objectStore(KEY_STORE_NAME);
      store.delete(storageKey);
      transaction.oncomplete = () => resolve();
      transaction.onabort = () => reject(transaction.error ?? new Error("Failed to clear private key"));
      transaction.onerror = () => reject(transaction.error ?? new Error("Failed to clear private key"));
    });
  } finally {
    database.close();
  }
}

function buildOrderIntentMessage(input: {
  sessionId: string;
  walletAddress: string;
  userId: number;
  marketId: number;
  outcome: string;
  side: string;
  orderType: string;
  timeInForce: string;
  price: number;
  quantity: number;
  clientOrderId: string;
  nonce: number;
  requestedAt: number;
}) {
  const normalized = {
    ...input,
    walletAddress: normalizeAddress(input.walletAddress),
    outcome: normalizeOrderField(input.outcome),
    side: normalizeOrderField(input.side),
    orderType: normalizeOrderField(input.orderType),
    timeInForce: normalizeOrderField(input.timeInForce),
    clientOrderId: input.clientOrderId.trim()
  };
  return `FunnyOption Order Authorization

session_id: ${normalized.sessionId}
wallet: ${normalized.walletAddress}
user_id: ${normalized.userId}
market_id: ${normalized.marketId}
outcome: ${normalized.outcome}
side: ${normalized.side}
order_type: ${normalized.orderType}
time_in_force: ${normalized.timeInForce}
price: ${normalized.price}
quantity: ${normalized.quantity}
client_order_id: ${normalized.clientOrderId}
nonce: ${normalized.nonce}
requested_at: ${normalized.requestedAt}
`;
}

function buildTradingKeyAuthorizationTypedData(input: {
  walletAddress: string;
  chainId: number;
  vaultAddress: string;
  tradingPublicKey: string;
  challenge: string;
  challengeExpiresAt: number;
  keyExpiresAt: number;
  scope: string;
}) {
  return {
    types: {
      EIP712Domain: [
        { name: "name", type: "string" },
        { name: "version", type: "string" },
        { name: "chainId", type: "uint256" },
        { name: "verifyingContract", type: "address" }
      ],
      AuthorizeTradingKey: [
        { name: "action", type: "string" },
        { name: "wallet", type: "address" },
        { name: "tradingPublicKey", type: "bytes32" },
        { name: "tradingKeyScheme", type: "string" },
        { name: "scope", type: "string" },
        { name: "challenge", type: "bytes32" },
        { name: "challengeExpiresAt", type: "uint64" },
        { name: "keyExpiresAt", type: "uint64" }
      ]
    },
    primaryType: "AuthorizeTradingKey",
    domain: {
      name: "FunnyOption Trading Authorization",
      version: "2",
      chainId: input.chainId,
      verifyingContract: normalizeAddress(input.vaultAddress)
    },
    message: {
      action: "AUTHORIZE_TRADING_KEY",
      wallet: normalizeAddress(input.walletAddress),
      tradingPublicKey: normalizeAddress(input.tradingPublicKey),
      tradingKeyScheme: DEFAULT_KEY_SCHEME,
      scope: input.scope,
      challenge: normalizeAddress(input.challenge),
      challengeExpiresAt: input.challengeExpiresAt,
      keyExpiresAt: input.keyExpiresAt
    }
  } as const;
}

async function signTypedDataV4(walletAddress: string, typedData: ReturnType<typeof buildTradingKeyAuthorizationTypedData>) {
  const ethereum = ensureEthereum();
  return (await ethereum.request({
    method: "eth_signTypedData_v4",
    params: [walletAddress, JSON.stringify(typedData)]
  })) as string;
}

async function requestTradingKeyChallenge(walletAddress: string, chainId: number, vaultAddress: string) {
  const response = await fetch(`${API_BASE_URL}/api/v1/trading-keys/challenge`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      wallet_address: walletAddress,
      chain_id: chainId,
      vault_address: vaultAddress
    })
  });

  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new Error(payload?.error ?? `HTTP ${response.status}`);
  }

  return (await response.json()) as TradingKeyChallenge;
}

export async function listSessions(filters?: { walletAddress?: string; userId?: number; vaultAddress?: string; status?: string; limit?: number }) {
  const query = new URLSearchParams();
  if (filters?.walletAddress) query.set("wallet_address", filters.walletAddress);
  if (filters?.userId) query.set("user_id", String(filters.userId));
  if (filters?.vaultAddress) query.set("vault_address", filters.vaultAddress);
  if (filters?.status) query.set("status", filters.status);
  query.set("limit", String(Math.max(1, Math.min(filters?.limit ?? 20, 200))));

  const response = await fetch(`${API_BASE_URL}/api/v1/sessions?${query.toString()}`, {
    cache: "no-store"
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new Error(payload?.error ?? `HTTP ${response.status}`);
  }
  const payload = (await response.json()) as { items?: RemoteSession[] };
  return payload.items ?? [];
}

export async function revokeRemoteSession(sessionId: string) {
  const response = await fetch(`${API_BASE_URL}/api/v1/sessions/${sessionId}/revoke`, {
    method: "POST"
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new Error(payload?.error ?? `HTTP ${response.status}`);
  }
  return response.json();
}

export async function fetchChainTasks() {
  const response = await fetch(`${API_BASE_URL}/api/v1/chain-transactions?limit=20`, {
    cache: "no-store"
  });
  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new Error(payload?.error ?? `HTTP ${response.status}`);
  }
  const payload = (await response.json()) as { items?: RemoteChainTask[] };
  return payload.items ?? [];
}

export async function connectWallet(): Promise<WalletConnection> {
  await ensureTargetChain();
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

function isExpired(expiresAt: number, now = Date.now()) {
  return expiresAt > 0 && expiresAt <= now;
}

function matchesStoredRemoteSession(stored: SessionRecord, remote: RemoteSession) {
  return (
    remote.session_id === stored.sessionId &&
    remote.chain_id === stored.chainId &&
    normalizeOptionalAddress(remote.vault_address) === stored.vaultAddress &&
    normalizeAddress(remote.wallet_address) === stored.walletAddress &&
    normalizeAddress(remote.session_public_key) === stored.sessionPublicKey
  );
}

function buildRestoreFailure(status: RestoreSessionStatus, message: string): RestoreSessionResult {
  return {
    session: null,
    status,
    message
  };
}

function getTargetStoredSessions(storedItems: SessionRecord[], targetVaultAddress: string) {
  return storedItems.filter((item) => item.chainId === TARGET_CHAIN_ID && item.vaultAddress === targetVaultAddress);
}

async function restoreStoredSessionRecord(
  stored: SessionRecord,
  targetVaultAddress: string,
  restoredMessage = "已恢复本地交易密钥"
): Promise<RestoreSessionResult> {
  if (normalizeAddress(stored.vaultAddress) !== targetVaultAddress) {
    await clearStoredSession(stored);
    return {
      session: null,
      status: "vault_mismatch",
      message: "当前 vault 已变化，已清空旧交易密钥。"
    };
  }

  const storageKey = buildStorageKey(stored.walletAddress, stored.chainId, stored.vaultAddress);
  const privateKey = await getStoredPrivateKey(storageKey);
  if (!privateKey) {
    await clearStoredSession(stored);
    return buildRestoreFailure("missing_private_key", "浏览器本地交易私钥已丢失，必须重新授权。");
  }

  const remoteSessions = await listSessions({
    walletAddress: stored.walletAddress,
    vaultAddress: stored.vaultAddress,
    limit: 200
  });
  const exactRemote = remoteSessions.find((item) => item.session_id === stored.sessionId);
  const currentActive = remoteSessions.find(
    (item) =>
      item.chain_id === stored.chainId &&
      normalizeOptionalAddress(item.vault_address) === stored.vaultAddress &&
      normalizeSessionStatus(item.status) === "ACTIVE" &&
      !isExpired(item.expires_at)
  );

  if (exactRemote) {
    if (!matchesStoredRemoteSession(stored, exactRemote)) {
      await clearStoredSession(stored);
      return buildRestoreFailure("remote_mismatch", "服务端交易密钥记录与本地记录不一致，不能假装恢复，请重新授权。");
    }

    if (isExpired(exactRemote.expires_at)) {
      await clearStoredSession(stored);
      return buildRestoreFailure("expired", "当前交易密钥已过期，必须重新授权。");
    }

    switch (normalizeSessionStatus(exactRemote.status)) {
      case "ACTIVE": {
        const restored = buildSessionRecord(exactRemote);
        saveStoredSession(restored);
        return {
          session: restored,
          status: "restored",
          message: restoredMessage
        };
      }
      case "ROTATED":
        await clearStoredSession(stored);
        return buildRestoreFailure("rotated", "当前交易密钥已在别处轮换，必须重新授权。");
      case "REVOKED":
        await clearStoredSession(stored);
        return buildRestoreFailure("revoked", "当前交易密钥已撤销，必须重新授权。");
      default:
        await clearStoredSession(stored);
        return buildRestoreFailure("remote_missing", "服务端已不再接受这把交易密钥，必须重新授权。");
    }
  }

  if (currentActive) {
    await clearStoredSession(stored);
    return buildRestoreFailure("rotated", "当前钱包的活动交易密钥已轮换到另一把 key，必须重新授权。");
  }

  const historicalMatch = remoteSessions.find(
    (item) =>
      item.chain_id === stored.chainId &&
      normalizeOptionalAddress(item.vault_address) === stored.vaultAddress &&
      normalizeAddress(item.wallet_address) === stored.walletAddress &&
      normalizeAddress(item.session_public_key) === stored.sessionPublicKey
  );
  if (historicalMatch) {
    await clearStoredSession(stored);
    if (isExpired(historicalMatch.expires_at)) {
      return buildRestoreFailure("expired", "当前交易密钥已过期，必须重新授权。");
    }
    switch (normalizeSessionStatus(historicalMatch.status)) {
      case "ROTATED":
        return buildRestoreFailure("rotated", "当前交易密钥已在别处轮换，必须重新授权。");
      case "REVOKED":
        return buildRestoreFailure("revoked", "当前交易密钥已撤销，必须重新授权。");
      default:
        return buildRestoreFailure("remote_mismatch", "服务端交易密钥记录与本地记录不一致，不能假装恢复，请重新授权。");
    }
  }

  await clearStoredSession(stored);
  return buildRestoreFailure("remote_missing", "服务端已找不到这把交易密钥，必须重新授权。");
}

export async function restoreStoredSession(
  activeWallet?: WalletConnection | null,
  options?: RestoreSessionOptions
): Promise<RestoreSessionResult> {
  const targetVaultAddress = ensureTargetVaultAddress();
  const storedItems = listStoredSessionMetadata();
  const targetStoredItems = getTargetStoredSessions(storedItems, targetVaultAddress);
  const connectedWallet =
    activeWallet ?? ((options?.allowWalletProbe ?? true) ? await getWalletConnection().catch(() => null) : null);

  if (!connectedWallet) {
    if (targetStoredItems.length === 1) {
      return restoreStoredSessionRecord(
        targetStoredItems[0],
        targetVaultAddress,
        "已恢复本地交易密钥，钱包将在需要时校验。"
      );
    }
    if (targetStoredItems.length > 1) {
      return buildRestoreFailure("wallet_required", "检测到多个本地交易密钥，连接对应钱包后可恢复。");
    }
    return buildRestoreFailure("missing", "未发现本地交易密钥");
  }

  const walletScopedItems = storedItems.filter((item) => item.walletAddress === normalizeAddress(connectedWallet.walletAddress));
  if (connectedWallet.chainId !== TARGET_CHAIN_ID) {
    if (walletScopedItems.some((item) => item.chainId === TARGET_CHAIN_ID && item.vaultAddress === targetVaultAddress)) {
      return {
        session: null,
        status: "chain_mismatch",
        message: `钱包当前在链 ${connectedWallet.chainId}，切回 ${TARGET_CHAIN.chainName} 后可恢复交易密钥。`
      };
    }
    return buildRestoreFailure("missing", "当前钱包在错误链上，且没有可恢复的交易密钥。");
  }

  const stored = loadStoredSessionMetadata(connectedWallet.walletAddress, TARGET_CHAIN_ID, targetVaultAddress);
  if (!stored) {
    if (walletScopedItems.some((item) => item.chainId === TARGET_CHAIN_ID && item.vaultAddress !== targetVaultAddress)) {
      return {
        session: null,
        status: "vault_mismatch",
        message: "本地交易密钥属于另一个 vault 环境，当前环境需要重新授权。"
      };
    }
    if (targetStoredItems.length > 0) {
      return {
        session: null,
        status: "wallet_mismatch",
        message: "当前连接的钱包与本地交易密钥不匹配。"
      };
    }
    return buildRestoreFailure("missing", "当前钱包没有可恢复的交易密钥。");
  }

  return restoreStoredSessionRecord(stored, targetVaultAddress);
}

export async function clearStoredSession(record?: SessionRecord | null) {
  const stored = record ?? null;
  if (typeof window !== "undefined" && stored) {
    window.localStorage.removeItem(buildMetadataStorageKey(stored.walletAddress, stored.chainId, stored.vaultAddress));
  }
  if (!stored) {
    return;
  }
  await deleteStoredPrivateKey(buildStorageKey(stored.walletAddress, stored.chainId, stored.vaultAddress)).catch(() => undefined);
}

export async function authorizeSession(existingWallet?: WalletConnection) {
  const targetVaultAddress = ensureTargetVaultAddress();
  const wallet = existingWallet && existingWallet.chainId === TARGET_CHAIN_ID ? existingWallet : await connectWallet();
  const privateKey = ed.utils.randomPrivateKey();
  const publicKey = await ed.getPublicKeyAsync(privateKey);
  const sessionPublicKey = toHex(publicKey);
  const challenge = await requestTradingKeyChallenge(wallet.walletAddress, wallet.chainId, targetVaultAddress);
  const typedData = buildTradingKeyAuthorizationTypedData({
    walletAddress: wallet.walletAddress,
    chainId: wallet.chainId,
    vaultAddress: targetVaultAddress,
    tradingPublicKey: sessionPublicKey,
    challenge: challenge.challenge,
    challengeExpiresAt: challenge.challenge_expires_at,
    keyExpiresAt: 0,
    scope: DEFAULT_SCOPE
  });
  const walletSignature = await signTypedDataV4(wallet.walletAddress, typedData);

  const response = await fetch(`${API_BASE_URL}/api/v1/trading-keys`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      wallet_address: wallet.walletAddress,
      chain_id: wallet.chainId,
      vault_address: targetVaultAddress,
      challenge_id: challenge.challenge_id,
      challenge: challenge.challenge,
      challenge_expires_at: challenge.challenge_expires_at,
      trading_public_key: sessionPublicKey,
      trading_key_scheme: DEFAULT_KEY_SCHEME,
      scope: DEFAULT_SCOPE,
      key_expires_at: 0,
      wallet_signature_standard: DEFAULT_WALLET_SIGNATURE_STANDARD,
      wallet_signature: walletSignature
    })
  });

  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new Error(payload?.error ?? `HTTP ${response.status}`);
  }

  const payload = (await response.json()) as RemoteSession;
  const record = buildSessionRecord(payload);
  const storageKey = buildStorageKey(record.walletAddress, record.chainId, record.vaultAddress);
  try {
    await putStoredPrivateKey(storageKey, toHex(privateKey));
    saveStoredSession(record);
  } catch (error) {
    await clearStoredSession(record);
    throw error instanceof Error ? error : new Error("Failed to persist the local trading private key");
  }
  return record;
}

export async function signOrderWithSession(
  session: SessionRecord,
  order: {
    marketId: number;
    outcome: string;
    side: string;
    orderType: string;
    timeInForce: string;
    price: number;
    quantity: number;
    clientOrderId: string;
  }
): Promise<OrderSignaturePayload> {
  const storageKey = buildStorageKey(session.walletAddress, session.chainId, session.vaultAddress);
  const privateKey = await getStoredPrivateKey(storageKey);
  if (!privateKey) {
    await clearStoredSession(session);
    throw new Error("Local trading private key is missing; please authorize again");
  }

  const requestedAt = Date.now();
  const orderNonce = session.lastOrderNonce + 1;
  const message = buildOrderIntentMessage({
    sessionId: session.sessionId,
    walletAddress: session.walletAddress,
    userId: session.userId,
    marketId: order.marketId,
    outcome: order.outcome,
    side: order.side,
    orderType: order.orderType,
    timeInForce: order.timeInForce,
    price: order.price,
    quantity: order.quantity,
    clientOrderId: order.clientOrderId,
    nonce: orderNonce,
    requestedAt
  });
  const signature = await ed.signAsync(new TextEncoder().encode(message), hexToBytes(privateKey));

  return {
    sessionId: session.sessionId,
    walletAddress: session.walletAddress,
    sessionSignature: toHex(signature),
    orderNonce,
    requestedAt,
    userId: session.userId
  };
}

export function bumpSessionOrderNonce(session: SessionRecord, nextNonce: number) {
  const updated = { ...session, lastOrderNonce: nextNonce };
  saveStoredSession(updated);
  return updated;
}
