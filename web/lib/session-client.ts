import * as ed from "@noble/ed25519";
import { ensureTargetChain, getChainMeta } from "@/lib/chain";

const STORAGE_KEY = "funnyoption:session:v1";
const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";
const DEFAULT_USER_ID = Number(process.env.NEXT_PUBLIC_DEFAULT_USER_ID ?? "1001");
const DEFAULT_SCOPE = "TRADE";
const TARGET_CHAIN_ID = getChainMeta().chainId;

export interface WalletConnection {
  walletAddress: string;
  chainId: number;
}

export interface SessionRecord {
  userId: number;
  walletAddress: string;
  chainId: number;
  sessionId: string;
  sessionPublicKey: string;
  sessionPrivateKey: string;
  sessionNonce: string;
  lastOrderNonce: number;
  expiresAt: number;
  issuedAt: number;
  scope: string;
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
  session_nonce: string;
  last_order_nonce: number;
  status: string;
  issued_at: number;
  expires_at: number;
  revoked_at: number;
  created_at: number;
  updated_at: number;
}

export interface RemoteChainTask {
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

function utf8ToHex(value: string) {
  return toHex(new TextEncoder().encode(value));
}

function randomNonce(prefix: string) {
  return `${prefix}_${Math.random().toString(16).slice(2, 10)}_${Date.now()}`;
}

function normalizeAddress(value: string) {
  return value.trim().toLowerCase();
}

function normalizeOrderField(value: string) {
  return value.trim().toUpperCase();
}

function ensureEthereum() {
  if (typeof window === "undefined" || !window.ethereum) {
    throw new Error("MetaMask or an EIP-1193 wallet is required");
  }
  return window.ethereum;
}

export function loadStoredSession() {
  if (typeof window === "undefined") return null;
  const raw = window.localStorage.getItem(STORAGE_KEY);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as SessionRecord;
    if (!parsed?.sessionId || !parsed?.sessionPrivateKey) return null;
    if (parsed.expiresAt <= Date.now()) {
      window.localStorage.removeItem(STORAGE_KEY);
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

export function saveStoredSession(session: SessionRecord) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
}

export function clearStoredSession() {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(STORAGE_KEY);
}

export async function listSessions(filters?: { walletAddress?: string; userId?: number }) {
  const query = new URLSearchParams();
  if (filters?.walletAddress) query.set("wallet_address", filters.walletAddress);
  if (filters?.userId) query.set("user_id", String(filters.userId));
  query.set("limit", "20");

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

function buildSessionGrantMessage(input: {
  walletAddress: string;
  sessionPublicKey: string;
  scope: string;
  chainId: number;
  issuedAt: number;
  expiresAt: number;
  nonce: string;
}) {
  return `FunnyOption Session Authorization

wallet: ${normalizeAddress(input.walletAddress)}
session_public_key: ${input.sessionPublicKey.toLowerCase()}
scope: ${input.scope}
chain_id: ${input.chainId}
issued_at: ${input.issuedAt}
expires_at: ${input.expiresAt}
nonce: ${input.nonce}
`;
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

async function signPersonalMessage(message: string, walletAddress: string) {
  const ethereum = ensureEthereum();
  return (await ethereum.request({
    method: "personal_sign",
    params: [utf8ToHex(message), walletAddress]
  })) as string;
}

export async function authorizeSession(existingWallet?: WalletConnection) {
  const wallet = existingWallet && existingWallet.chainId === TARGET_CHAIN_ID ? existingWallet : await connectWallet();
  const privateKey = ed.utils.randomPrivateKey();
  const publicKey = await ed.getPublicKeyAsync(privateKey);
  const issuedAt = Date.now();
  const expiresAt = issuedAt + 24 * 60 * 60 * 1000;
  const nonce = randomNonce("sess");
  const sessionPublicKey = toHex(publicKey);
  const message = buildSessionGrantMessage({
    walletAddress: wallet.walletAddress,
    sessionPublicKey,
    scope: DEFAULT_SCOPE,
    chainId: wallet.chainId,
    issuedAt,
    expiresAt,
    nonce
  });
  const walletSignature = await signPersonalMessage(message, wallet.walletAddress);

  const response = await fetch(`${API_BASE_URL}/api/v1/sessions`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      user_id: DEFAULT_USER_ID,
      wallet_address: wallet.walletAddress,
      session_public_key: sessionPublicKey,
      scope: DEFAULT_SCOPE,
      chain_id: wallet.chainId,
      nonce,
      issued_at: issuedAt,
      expires_at: expiresAt,
      wallet_signature: walletSignature
    })
  });

  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new Error(payload?.error ?? `HTTP ${response.status}`);
  }

  const payload = (await response.json()) as {
    session_id: string;
    wallet_address: string;
    chain_id: number;
    session_public_key: string;
    session_nonce: string;
    last_order_nonce: number;
    expires_at: number;
    issued_at: number;
    scope: string;
    user_id: number;
  };

  const record: SessionRecord = {
    userId: payload.user_id,
    walletAddress: normalizeAddress(payload.wallet_address),
    chainId: payload.chain_id,
    sessionId: payload.session_id,
    sessionPublicKey: payload.session_public_key,
    sessionPrivateKey: toHex(privateKey),
    sessionNonce: payload.session_nonce,
    lastOrderNonce: payload.last_order_nonce,
    expiresAt: payload.expires_at,
    issuedAt: payload.issued_at,
    scope: payload.scope
  };

  saveStoredSession(record);
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
  const signature = await ed.signAsync(new TextEncoder().encode(message), hexToBytes(session.sessionPrivateKey));

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
