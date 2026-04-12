import type { Balance, ChainTask, Deposit, Market, Payout, Position, SessionGrant, Trade, Withdrawal } from "@/lib/types";

export const mockMarkets: Market[] = [
  {
    market_id: 1001,
    title: "Bitcoin closes April above 95k",
    description: "A directional macro tape for the next local cycle close. Resolution is the Binance BTCUSDT daily candle close on April 30.",
    collateral_asset: "USDT",
    status: "OPEN",
    open_at: 1743408000,
    close_at: 1746057600,
    resolve_at: 1746061200,
    resolved_outcome: "",
    created_by: 1,
    metadata: { category: "macro", yesOdds: 0.61, noOdds: 0.39, volume: 182340 },
    runtime: { trade_count: 0, matched_quantity: 0, matched_notional: 0, last_trade_at: 0, last_price_yes: 0, last_price_no: 0, active_order_count: 0, payout_count: 0, completed_payout_count: 0, pending_claim_count: 0, submitted_claim_count: 0, failed_claim_count: 0 },
    created_at: 1743380000,
    updated_at: 1743380000
  },
  {
    market_id: 1002,
    title: "ETH ETF net inflow > 1.2B by month end",
    description: "Tracks aggregate US spot ETH ETF net inflows. A good market for flow traders and momentum desks.",
    collateral_asset: "USDT",
    status: "OPEN",
    open_at: 1743408000,
    close_at: 1745798400,
    resolve_at: 1745802000,
    resolved_outcome: "",
    created_by: 1,
    metadata: { category: "flow", yesOdds: 0.48, noOdds: 0.52, volume: 128120 },
    runtime: { trade_count: 0, matched_quantity: 0, matched_notional: 0, last_trade_at: 0, last_price_yes: 0, last_price_no: 0, active_order_count: 0, payout_count: 0, completed_payout_count: 0, pending_claim_count: 0, submitted_claim_count: 0, failed_claim_count: 0 },
    created_at: 1743384000,
    updated_at: 1743384000
  },
  {
    market_id: 1003,
    title: "BSC meme index prints a fresh 30d high",
    description: "A high-beta sentiment gauge for the chain we settle on. If the index tags a new 30-day high, YES wins.",
    collateral_asset: "USDT",
    status: "PAUSED",
    open_at: 1743408000,
    close_at: 1745276400,
    resolve_at: 1745280000,
    resolved_outcome: "",
    created_by: 1,
    metadata: { category: "chain-native", yesOdds: 0.72, noOdds: 0.28, volume: 93220 },
    runtime: { trade_count: 0, matched_quantity: 0, matched_notional: 0, last_trade_at: 0, last_price_yes: 0, last_price_no: 0, active_order_count: 0, payout_count: 0, completed_payout_count: 0, pending_claim_count: 0, submitted_claim_count: 0, failed_claim_count: 0 },
    created_at: 1743386000,
    updated_at: 1743386000
  }
];

export const mockTrades: Trade[] = [
  {
    trade_id: "trd_1",
    sequence_no: 60021,
    market_id: 1001,
    outcome: "YES",
    collateral_asset: "USDT",
    price: 61,
    quantity: 4200,
    taker_order_id: "ord_1",
    maker_order_id: "ord_2",
    taker_user_id: 1001,
    maker_user_id: 1002,
    taker_side: "BUY",
    maker_side: "SELL",
    occurred_at: 1743469200
  },
  {
    trade_id: "trd_2",
    sequence_no: 60022,
    market_id: 1001,
    outcome: "NO",
    collateral_asset: "USDT",
    price: 39,
    quantity: 2800,
    taker_order_id: "ord_3",
    maker_order_id: "ord_4",
    taker_user_id: 1004,
    maker_user_id: 1001,
    taker_side: "BUY",
    maker_side: "SELL",
    occurred_at: 1743472800
  },
  {
    trade_id: "trd_3",
    sequence_no: 60023,
    market_id: 1002,
    outcome: "NO",
    collateral_asset: "USDT",
    price: 52,
    quantity: 3500,
    taker_order_id: "ord_5",
    maker_order_id: "ord_6",
    taker_user_id: 1002,
    maker_user_id: 1007,
    taker_side: "BUY",
    maker_side: "SELL",
    occurred_at: 1743476400
  }
];

export const mockBalances: Balance[] = [
  { user_id: 1001, asset: "USDT", available: 124500, frozen: 27100, created_at: 1743380000, updated_at: 1743476400 },
  { user_id: 1001, asset: "POSITION:1001:YES", available: 0, frozen: 0, created_at: 1743380000, updated_at: 1743476400 }
];

export const mockPositions: Position[] = [
  {
    market_id: 1001,
    user_id: 1001,
    outcome: "YES",
    position_asset: "POSITION:1001:YES",
    quantity: 18000,
    settled_quantity: 0,
    created_at: 1743380000,
    updated_at: 1743476400
  },
  {
    market_id: 1002,
    user_id: 1001,
    outcome: "NO",
    position_asset: "POSITION:1002:NO",
    quantity: 7400,
    settled_quantity: 0,
    created_at: 1743384000,
    updated_at: 1743476400
  }
];

export const mockDeposits: Deposit[] = [
  {
    deposit_id: "cdep_1",
    user_id: 1001,
    address: "0x9f4f6c91c4ce5d7f50da8f2cd2e6a6123ec4e7c1",
    asset: "USDT",
    chain_amount: "50000000000000000000",
    credit_amount: 5000,
    chain_id: 97,
    tx_hash: "0xabcd1",
    status: "CREDITED",
    created_at: 1743476480,
  }
];

export const mockPayouts: Payout[] = [
  {
    event_id: "evt_settlement_1000",
    market_id: 998,
    user_id: 1001,
    winning_outcome: "YES",
    position_asset: "POSITION:998:YES",
    settled_quantity: 6200,
    payout_asset: "USDT",
    payout_amount: 6200,
    status: "COMPLETED",
    created_at: 1743389000,
    updated_at: 1743389100
  }
];

export const mockWithdrawals: Withdrawal[] = [
  {
    withdraw_id: "cwdr_1",
    user_id: 1001,
    to_address: "0x9f4f6c91c4ce5d7f50da8f2cd2e6a6123ec4e7c1",
    asset: "USDT",
    amount: 12500,
    status: "SUBMITTED",
    tx_hash: "0xw1234",
    created_at: 1743477790,
  }
];

export const mockChainTasks: ChainTask[] = [
  {
    id: 1,
    biz_type: "CLAIM",
    ref_id: "evt_settlement_1000",
    chain_name: "bsc",
    network_name: "testnet",
    wallet_address: "0x9f4f6c91c4ce5d7f50da8f2cd2e6a6123ec4e7c1",
    tx_hash: "",
    status: "PENDING",
    error_message: "",
    attempt_count: 0,
    created_at: 1743480100,
    updated_at: 1743480100
  }
];

export const mockSessions: SessionGrant[] = [
  {
    session_id: "sess_2b39c7aa",
    user_id: 1001,
    wallet_address: "0x9f4f6c91c4ce5d7f50da8f2cd2e6a6123ec4e7c1",
    session_public_key: "0x8f24b10bcce2f6",
    scope: "TRADE",
    chain_id: 97,
    session_nonce: "sess_1c7e7b11",
    last_order_nonce: 12,
    status: "ACTIVE",
    issued_at: 1743472000000,
    expires_at: 1743558400000,
    revoked_at: 0,
    created_at: 1743472000,
    updated_at: 1743476600
  }
];
