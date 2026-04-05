export type MarketStatus = "DRAFT" | "OPEN" | "PAUSED" | "CLOSED" | "WAITING_RESOLUTION" | "RESOLVED";

export interface ApiReadError {
  status?: number;
  message: string;
}

export interface ApiCollectionResult<T> {
  state: "ok" | "empty" | "unavailable";
  items: T[];
  error?: ApiReadError;
}

export interface ApiItemResult<T> {
  state: "ok" | "not-found" | "unavailable";
  item: T | null;
  error?: ApiReadError;
}

export interface MarketMetadata {
  category?: string;
  yesOdds?: number;
  noOdds?: number;
  volume?: number;
  matchedQuantity?: number;
  matchedNotional?: number;
  tradeCount?: number;
  lastTradeAt?: number;
  coverImage?: string;
  coverImageUrl?: string;
  cover_image_url?: string;
  sourceUrl?: string;
  sourceName?: string;
  coverSourceName?: string;
  cover_source_name?: string;
  sourceSlug?: string;
  sourceKind?: string;
  sourceTitle?: string;
  sourceDescription?: string;
  [key: string]: unknown;
}

export interface MarketCategory {
  category_id: number;
  category_key: string;
  display_name: string;
  description?: string;
  sort_order?: number;
}

export interface MarketOption {
  key: string;
  label: string;
  short_label?: string;
  sort_order: number;
  is_active: boolean;
}

export interface MarketRuntime {
  trade_count: number;
  matched_quantity: number;
  matched_notional: number;
  last_trade_at: number;
  last_price_yes: number;
  last_price_no: number;
  active_order_count: number;
  payout_count: number;
  completed_payout_count: number;
  pending_claim_count: number;
  submitted_claim_count: number;
  failed_claim_count: number;
}

export interface Market {
  market_id: number;
  title: string;
  description: string;
  collateral_asset: string;
  category?: MarketCategory | null;
  status: MarketStatus;
  open_at: number;
  close_at: number;
  resolve_at: number;
  resolved_outcome: string;
  created_by: number;
  options?: MarketOption[];
  metadata?: MarketMetadata | null;
  runtime: MarketRuntime;
  created_at: number;
  updated_at: number;
}

export interface Trade {
  trade_id: string;
  sequence_no: number;
  market_id: number;
  outcome: string;
  collateral_asset: string;
  price: number;
  quantity: number;
  taker_order_id: string;
  maker_order_id: string;
  taker_user_id: number;
  maker_user_id: number;
  taker_side: string;
  maker_side: string;
  occurred_at: number;
}

export interface Order {
  order_id: string;
  client_order_id: string;
  command_id: string;
  user_id: number;
  market_id: number;
  outcome: string;
  side: string;
  order_type: string;
  time_in_force: string;
  collateral_asset: string;
  freeze_id: string;
  freeze_asset: string;
  freeze_amount: number;
  price: number;
  quantity: number;
  filled_quantity: number;
  remaining_quantity: number;
  status: string;
  cancel_reason: string;
  created_at: number;
  updated_at: number;
}

export interface Balance {
  user_id: number;
  asset: string;
  available: number;
  frozen: number;
  created_at: number;
  updated_at: number;
}

export interface UserProfile {
  user_id: number;
  wallet_address: string;
  display_name: string;
  avatar_preset: string;
  created_at: number;
  updated_at: number;
}

export interface Position {
  market_id: number;
  user_id: number;
  outcome: string;
  position_asset: string;
  quantity: number;
  settled_quantity: number;
  created_at: number;
  updated_at: number;
}

export interface Deposit {
  deposit_id: string;
  user_id: number;
  wallet_address: string;
  vault_address: string;
  asset: string;
  amount: number;
  chain_name: string;
  network_name: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  status: string;
  credited_at: number;
  created_at: number;
  updated_at: number;
}

export interface Withdrawal {
  withdrawal_id: string;
  user_id: number;
  wallet_address: string;
  recipient_address: string;
  vault_address: string;
  asset: string;
  amount: number;
  chain_name: string;
  network_name: string;
  tx_hash: string;
  log_index: number;
  block_number: number;
  status: string;
  debited_at: number;
  created_at: number;
  updated_at: number;
}

export interface Payout {
  event_id: string;
  market_id: number;
  user_id: number;
  winning_outcome: string;
  position_asset: string;
  settled_quantity: number;
  payout_asset: string;
  payout_amount: number;
  status: string;
  created_at: number;
  updated_at: number;
}

export interface SessionGrant {
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

export interface ChainTask {
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
