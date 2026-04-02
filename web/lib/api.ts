import type {
  ApiCollectionResult,
  ApiItemResult,
  ApiReadError,
  Balance,
  ChainTask,
  Deposit,
  Market,
  MarketRuntime,
  Order,
  Payout,
  Position,
  SessionGrant,
  Trade,
  UserProfile,
  Withdrawal
} from "@/lib/types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";

const EMPTY_MARKET_RUNTIME: MarketRuntime = {
  trade_count: 0,
  matched_quantity: 0,
  matched_notional: 0,
  last_trade_at: 0,
  last_price_yes: 0,
  last_price_no: 0,
  active_order_count: 0,
  payout_count: 0,
  completed_payout_count: 0,
  pending_claim_count: 0,
  submitted_claim_count: 0,
  failed_claim_count: 0
};

async function fetchItems<T>(path: string): Promise<T[]> {
  const result = await fetchCollection<T>(path);
  return result.items;
}

async function fetchObject<T>(path: string): Promise<T | null> {
  const result = await fetchItem<T>(path);
  return result.item;
}

function normalizeMarket(market: Market): Market {
  return {
    ...market,
    metadata: market.metadata ?? {},
    runtime: {
      ...EMPTY_MARKET_RUNTIME,
      ...(market.runtime ?? {})
    }
  };
}

type FetchResult<T> =
  | { kind: "ok"; data: T }
  | { kind: "not-found" }
  | { kind: "error"; error: ApiReadError };

function buildError(path: string, message: string, status?: number): ApiReadError {
  return {
    ...(status ? { status } : {}),
    message: `${message} (${path})`
  };
}

async function fetchJson<T>(path: string): Promise<FetchResult<T>> {
  let response: Response;

  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      cache: "no-store"
    });
  } catch {
    return {
      kind: "error",
      error: buildError(path, "Network error while contacting the local API")
    };
  }

  if (response.status === 404) {
    return { kind: "not-found" };
  }

  if (!response.ok) {
    return {
      kind: "error",
      error: buildError(path, `HTTP ${response.status} from the local API`, response.status)
    };
  }

  try {
    return {
      kind: "ok",
      data: (await response.json()) as T
    };
  } catch {
    return {
      kind: "error",
      error: buildError(path, "Invalid JSON from the local API", response.status)
    };
  }
}

function normalizeCollectionState<T>(items: T[], error?: ApiReadError): ApiCollectionResult<T> {
  if (items.length === 0) {
    return {
      state: "empty",
      items,
      ...(error ? { error } : {})
    };
  }

  return {
    state: "ok",
    items,
    ...(error ? { error } : {})
  };
}

async function fetchCollection<T>(path: string): Promise<ApiCollectionResult<T>> {
  const result = await fetchJson<{ items?: unknown }>(path);

  if (result.kind === "error") {
    return {
      state: "unavailable",
      items: [],
      error: result.error
    };
  }

  if (result.kind === "not-found") {
    return {
      state: "unavailable",
      items: [],
      error: buildError(path, "Unexpected 404 from collection endpoint", 404)
    };
  }

  if (!("items" in result.data) || !Array.isArray(result.data.items)) {
    return {
      state: "unavailable",
      items: [],
      error: buildError(path, "Unexpected collection response shape")
    };
  }

  return normalizeCollectionState(result.data.items as T[]);
}

async function fetchItem<T>(path: string): Promise<ApiItemResult<T>> {
  const result = await fetchJson<unknown>(path);

  if (result.kind === "not-found") {
    return {
      state: "not-found",
      item: null
    };
  }

  if (result.kind === "error") {
    return {
      state: "unavailable",
      item: null,
      error: result.error
    };
  }

  if (result.data === null || typeof result.data !== "object" || Array.isArray(result.data)) {
    return {
      state: "unavailable",
      item: null,
      error: buildError(path, "Unexpected object response shape")
    };
  }

  return {
    state: "ok",
    item: result.data as T
  };
}

function normalizeMarketCollection(result: ApiCollectionResult<Market>): ApiCollectionResult<Market> {
  if (result.state === "unavailable") {
    return result;
  }

  return {
    ...result,
    items: result.items.map(normalizeMarket)
  };
}

function normalizeMarketItem(result: ApiItemResult<Market>): ApiItemResult<Market> {
  if (result.state !== "ok" || !result.item) {
    return result;
  }

  return {
    ...result,
    item: normalizeMarket(result.item)
  };
}

export async function getMarketsRead() {
  return normalizeMarketCollection(await fetchCollection<Market>("/api/v1/markets?limit=24"));
}

export async function getMarketRead(marketId: number) {
  return normalizeMarketItem(await fetchItem<Market>(`/api/v1/markets/${marketId}`));
}

export async function getTradesRead(marketId?: number) {
  const query = marketId ? `?market_id=${marketId}&limit=20` : "?limit=20";
  return fetchCollection<Trade>(`/api/v1/trades${query}`);
}

export async function getChainTasksRead() {
  return fetchCollection<ChainTask>("/api/v1/chain-transactions?limit=20");
}

export async function getMarkets() {
  return (await getMarketsRead()).items;
}

export async function getMarket(marketId: number) {
  return (await getMarketRead(marketId)).item;
}

export async function getTrades(marketId?: number) {
  return (await getTradesRead(marketId)).items;
}

export async function getOrders(userId = 1001, marketId?: number) {
  const query = new URLSearchParams({
    user_id: String(userId),
    limit: "20"
  });
  if (marketId) {
    query.set("market_id", String(marketId));
  }
  return fetchItems<Order>(`/api/v1/orders?${query.toString()}`);
}

export async function getBalances(userId = 1001) {
  return fetchItems<Balance>(`/api/v1/balances?user_id=${userId}&limit=10`);
}

export async function getProfileRead(userId?: number, walletAddress?: string) {
  const query = new URLSearchParams();
  if (userId && userId > 0) {
    query.set("user_id", String(userId));
  }
  if (walletAddress) {
    query.set("wallet_address", walletAddress);
  }
  const suffix = query.size > 0 ? `?${query.toString()}` : "";
  return fetchItem<UserProfile>(`/api/v1/profile${suffix}`);
}

export async function getProfile(userId = 1001) {
  return (await getProfileRead(userId)).item;
}

export async function updateProfile(input: {
  userId: number;
  sessionId: string;
  displayName?: string;
  avatarPreset: string;
}) {
  const response = await fetch(`${API_BASE_URL}/api/v1/profile`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      user_id: input.userId,
      session_id: input.sessionId,
      display_name: input.displayName ?? "",
      avatar_preset: input.avatarPreset
    })
  });

  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new Error(payload?.error ?? `HTTP ${response.status}`);
  }

  return (await response.json()) as UserProfile;
}

export async function getPositions(userId = 1001) {
  return fetchItems<Position>(`/api/v1/positions?user_id=${userId}&limit=20`);
}

export async function getDeposits(userId = 1001) {
  return fetchItems<Deposit>(`/api/v1/deposits?user_id=${userId}&limit=20`);
}

export async function getWithdrawals(userId = 1001) {
  return fetchItems<Withdrawal>(`/api/v1/withdrawals?user_id=${userId}&limit=20`);
}

export async function getPayouts(userId = 1001) {
  return fetchItems<Payout>(`/api/v1/payouts?user_id=${userId}&limit=20`);
}

export async function getSessions(userId = 1001) {
  return fetchItems<SessionGrant>(`/api/v1/sessions?user_id=${userId}&limit=20`);
}

export async function getChainTasks() {
  return (await getChainTasksRead()).items;
}
