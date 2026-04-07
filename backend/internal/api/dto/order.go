package dto

import (
	"encoding/json"
	"fmt"
	"strings"

	sharedauth "funnyoption/internal/shared/auth"

	"github.com/ethereum/go-ethereum/common"
)

type CreateOrderRequest struct {
	ClientOrderID     string          `json:"client_order_id"`
	TraceID           string          `json:"trace_id"`
	UserID            int64           `json:"user_id"`
	SessionID         string          `json:"session_id"`
	SessionSignature  string          `json:"session_signature"`
	Operator          *OperatorAction `json:"operator,omitempty"`
	OrderNonce        uint64          `json:"order_nonce"`
	RequestedAtMillis int64           `json:"requested_at"`
	MarketID          int64           `json:"market_id" binding:"required"`
	Outcome           string          `json:"outcome" binding:"required"`
	Side              string          `json:"side" binding:"required"`
	Type              string          `json:"type" binding:"required"`
	TimeInForce       string          `json:"time_in_force" binding:"required"`
	Price             int64           `json:"price"`
	Quantity          int64           `json:"quantity" binding:"required"`
}

type CreateOrderResponse struct {
	CommandID string `json:"command_id"`
	OrderID   string `json:"order_id"`
	FreezeID  string `json:"freeze_id,omitempty"`
	Asset     string `json:"asset,omitempty"`
	Amount    int64  `json:"amount,omitempty"`
	Topic     string `json:"topic"`
	Status    string `json:"status"`
}

type ResolveMarketRequest struct {
	Outcome  string          `json:"outcome" binding:"required"`
	Operator *OperatorAction `json:"operator,omitempty"`
}

type CreateMarketRequest struct {
	MarketID        int64           `json:"market_id"`
	Title           string          `json:"title" binding:"required"`
	Description     string          `json:"description"`
	CategoryKey     string          `json:"category_key"`
	CollateralAsset string          `json:"collateral_asset"`
	Status          string          `json:"status"`
	OpenAt          int64           `json:"open_at"`
	CloseAt         int64           `json:"close_at"`
	ResolveAt       int64           `json:"resolve_at"`
	CreatedBy       int64           `json:"created_by"`
	CoverImageURL   string          `json:"cover_image_url"`
	CoverSourceURL  string          `json:"cover_source_url"`
	CoverSourceName string          `json:"cover_source_name"`
	Options         []MarketOption  `json:"options"`
	Metadata        json.RawMessage `json:"metadata"`
	Operator        *OperatorAction `json:"operator,omitempty"`
}

type CreateFirstLiquidityRequest struct {
	UserID   int64           `json:"user_id" binding:"required"`
	Quantity int64           `json:"quantity" binding:"required"`
	Outcome  string          `json:"outcome,omitempty"`
	Price    int64           `json:"price,omitempty"`
	Operator *OperatorAction `json:"operator,omitempty"`
}

type FirstLiquidityInventoryResponse struct {
	Outcome       string `json:"outcome"`
	PositionAsset string `json:"position_asset"`
	Quantity      int64  `json:"quantity"`
}

type CreateFirstLiquidityResponse struct {
	FirstLiquidityID string                            `json:"first_liquidity_id"`
	MarketID         int64                             `json:"market_id"`
	UserID           int64                             `json:"user_id"`
	CollateralAsset  string                            `json:"collateral_asset"`
	CollateralDebit  int64                             `json:"collateral_debit"`
	Inventory        []FirstLiquidityInventoryResponse `json:"inventory"`
	Status           string                            `json:"status"`
	OrderID          string                            `json:"order_id,omitempty"`
	OrderStatus      string                            `json:"order_status,omitempty"`
}

type MarketResponse struct {
	MarketID        int64           `json:"market_id"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	CollateralAsset string          `json:"collateral_asset"`
	Category        *MarketCategory `json:"category,omitempty"`
	Status          string          `json:"status"`
	OpenAt          int64           `json:"open_at"`
	CloseAt         int64           `json:"close_at"`
	ResolveAt       int64           `json:"resolve_at"`
	ResolvedOutcome string          `json:"resolved_outcome"`
	CreatedBy       int64           `json:"created_by"`
	Options         []MarketOption  `json:"options"`
	Metadata        json.RawMessage `json:"metadata"`
	Runtime         MarketRuntime   `json:"runtime"`
	CreatedAt       int64           `json:"created_at"`
	UpdatedAt       int64           `json:"updated_at"`
}

type MarketRuntime struct {
	TradeCount           int64 `json:"trade_count"`
	MatchedQuantity      int64 `json:"matched_quantity"`
	MatchedNotional      int64 `json:"matched_notional"`
	LastTradeAt          int64 `json:"last_trade_at"`
	LastPriceYes         int64 `json:"last_price_yes"`
	LastPriceNo          int64 `json:"last_price_no"`
	ActiveOrderCount     int64 `json:"active_order_count"`
	PayoutCount          int64 `json:"payout_count"`
	CompletedPayoutCount int64 `json:"completed_payout_count"`
	PendingClaimCount    int64 `json:"pending_claim_count"`
	SubmittedClaimCount  int64 `json:"submitted_claim_count"`
	FailedClaimCount     int64 `json:"failed_claim_count"`
}

type ListMarketsRequest struct {
	Status      string `form:"status"`
	CreatedBy   int64  `form:"created_by"`
	CategoryKey string `form:"category_key"`
	Limit       int    `form:"limit"`
}

type MarketCategory struct {
	CategoryID  int64           `json:"category_id"`
	CategoryKey string          `json:"category_key"`
	DisplayName string          `json:"display_name"`
	Description string          `json:"description,omitempty"`
	SortOrder   int             `json:"sort_order,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type MarketOption struct {
	Key        string          `json:"key"`
	Label      string          `json:"label"`
	ShortLabel string          `json:"short_label,omitempty"`
	SortOrder  int             `json:"sort_order"`
	IsActive   bool            `json:"is_active"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

type ListOrdersRequest struct {
	UserID   int64  `form:"user_id"`
	MarketID int64  `form:"market_id"`
	Status   string `form:"status"`
	Limit    int    `form:"limit"`
}

type OrderResponse struct {
	OrderID           string `json:"order_id"`
	ClientOrderID     string `json:"client_order_id"`
	CommandID         string `json:"command_id"`
	UserID            int64  `json:"user_id"`
	MarketID          int64  `json:"market_id"`
	Outcome           string `json:"outcome"`
	Side              string `json:"side"`
	OrderType         string `json:"order_type"`
	TimeInForce       string `json:"time_in_force"`
	CollateralAsset   string `json:"collateral_asset"`
	FreezeID          string `json:"freeze_id"`
	FreezeAsset       string `json:"freeze_asset"`
	FreezeAmount      int64  `json:"freeze_amount"`
	Price             int64  `json:"price"`
	Quantity          int64  `json:"quantity"`
	FilledQuantity    int64  `json:"filled_quantity"`
	RemainingQuantity int64  `json:"remaining_quantity"`
	Status            string `json:"status"`
	CancelReason      string `json:"cancel_reason"`
	CreatedAt         int64  `json:"created_at"`
	UpdatedAt         int64  `json:"updated_at"`
}

type ListTradesRequest struct {
	UserID   int64  `form:"user_id"`
	MarketID int64  `form:"market_id"`
	Outcome  string `form:"outcome"`
	Limit    int    `form:"limit"`
}

type TradeResponse struct {
	TradeID         string `json:"trade_id"`
	SequenceNo      int64  `json:"sequence_no"`
	MarketID        int64  `json:"market_id"`
	Outcome         string `json:"outcome"`
	CollateralAsset string `json:"collateral_asset"`
	Price           int64  `json:"price"`
	Quantity        int64  `json:"quantity"`
	TakerOrderID    string `json:"taker_order_id"`
	MakerOrderID    string `json:"maker_order_id"`
	TakerUserID     int64  `json:"taker_user_id"`
	MakerUserID     int64  `json:"maker_user_id"`
	TakerSide       string `json:"taker_side"`
	MakerSide       string `json:"maker_side"`
	OccurredAt      int64  `json:"occurred_at"`
}

type ListBalancesRequest struct {
	UserID int64  `form:"user_id" binding:"required"`
	Asset  string `form:"asset"`
	Limit  int    `form:"limit"`
}

type BalanceResponse struct {
	UserID    int64  `json:"user_id"`
	Asset     string `json:"asset"`
	Available int64  `json:"available"`
	Frozen    int64  `json:"frozen"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type ListPositionsRequest struct {
	UserID   int64  `form:"user_id" binding:"required"`
	MarketID int64  `form:"market_id"`
	Outcome  string `form:"outcome"`
	Limit    int    `form:"limit"`
}

type PositionResponse struct {
	MarketID        int64  `json:"market_id"`
	UserID          int64  `json:"user_id"`
	Outcome         string `json:"outcome"`
	PositionAsset   string `json:"position_asset"`
	Quantity        int64  `json:"quantity"`
	SettledQuantity int64  `json:"settled_quantity"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

type ListPayoutsRequest struct {
	UserID   int64 `form:"user_id" binding:"required"`
	MarketID int64 `form:"market_id"`
	Limit    int   `form:"limit"`
}

type PayoutResponse struct {
	EventID         string `json:"event_id"`
	MarketID        int64  `json:"market_id"`
	UserID          int64  `json:"user_id"`
	WinningOutcome  string `json:"winning_outcome"`
	PositionAsset   string `json:"position_asset"`
	SettledQuantity int64  `json:"settled_quantity"`
	PayoutAsset     string `json:"payout_asset"`
	PayoutAmount    int64  `json:"payout_amount"`
	Status          string `json:"status"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

type ListRollupEscapeCollateralClaimsRequest struct {
	UserID        int64  `form:"user_id"`
	WalletAddress string `form:"wallet_address"`
	Status        string `form:"status"`
	Limit         int    `form:"limit"`
}

type RollupEscapeCollateralClaimResponse struct {
	BatchID          int64    `json:"batch_id"`
	AccountID        int64    `json:"account_id"`
	StateRoot        string   `json:"state_root"`
	CollateralAsset  string   `json:"collateral_asset"`
	MerkleRoot       string   `json:"merkle_root"`
	LeafCount        int64    `json:"leaf_count"`
	TotalAmount      int64    `json:"total_amount"`
	WalletAddress    string   `json:"wallet_address"`
	ClaimAmount      int64    `json:"claim_amount"`
	LeafIndex        int64    `json:"leaf_index"`
	LeafHash         string   `json:"leaf_hash"`
	ProofHashes      []string `json:"proof_hashes"`
	ClaimID          string   `json:"claim_id"`
	ClaimStatus      string   `json:"claim_status"`
	ClaimTxHash      string   `json:"claim_tx_hash"`
	ClaimSubmittedAt int64    `json:"claim_submitted_at"`
	ClaimedAt        int64    `json:"claimed_at"`
	AnchorStatus     string   `json:"anchor_status"`
	AnchorTxHash     string   `json:"anchor_tx_hash"`
	AnchorSubmittedAt int64   `json:"anchor_submitted_at"`
	AnchoredAt       int64    `json:"anchored_at"`
	LastError        string   `json:"last_error"`
	LastErrorAt      int64    `json:"last_error_at"`
	CreatedAt        int64    `json:"created_at"`
	UpdatedAt        int64    `json:"updated_at"`
}

type ListRollupWithdrawalClaimsRequest struct {
	UserID        int64  `form:"user_id"`
	WalletAddress string `form:"wallet_address"`
	BatchID       int64  `form:"batch_id"`
	Status        string `form:"status"`
	Limit         int    `form:"limit"`
}

type RollupWithdrawalClaimResponse struct {
	BatchID          int64    `json:"batch_id"`
	WithdrawalID     string   `json:"withdrawal_id"`
	AccountID        int64    `json:"account_id"`
	WalletAddress    string   `json:"wallet_address"`
	RecipientAddress string   `json:"recipient_address"`
	Amount           int64    `json:"amount"`
	LeafIndex        int64    `json:"leaf_index"`
	LeafHash         string   `json:"leaf_hash"`
	ProofHashes      []string `json:"proof_hashes"`
	ClaimID          string   `json:"claim_id"`
	ClaimStatus      string   `json:"claim_status"`
	ClaimTxHash      string   `json:"claim_tx_hash"`
	ClaimSubmittedAt int64    `json:"claim_submitted_at"`
	ClaimedAt        int64    `json:"claimed_at"`
	LastError        string   `json:"last_error"`
	LastErrorAt      int64    `json:"last_error_at"`
	CreatedAt        int64    `json:"created_at"`
	UpdatedAt        int64    `json:"updated_at"`
}

type ListFreezesRequest struct {
	UserID int64  `form:"user_id"`
	Status string `form:"status"`
	Limit  int    `form:"limit"`
}

type FreezeResponse struct {
	FreezeID        string `json:"freeze_id"`
	UserID          int64  `json:"user_id"`
	Asset           string `json:"asset"`
	RefType         string `json:"ref_type"`
	RefID           string `json:"ref_id"`
	OriginalAmount  int64  `json:"original_amount"`
	RemainingAmount int64  `json:"remaining_amount"`
	Status          string `json:"status"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

type ListLedgerEntriesRequest struct {
	BizType string `form:"biz_type"`
	RefID   string `form:"ref_id"`
	Limit   int    `form:"limit"`
}

type LedgerEntryResponse struct {
	EntryID    string `json:"entry_id"`
	BizType    string `json:"biz_type"`
	RefID      string `json:"ref_id"`
	Status     string `json:"status"`
	PostingCnt int64  `json:"posting_count"`
	CreatedAt  int64  `json:"created_at"`
}

type LedgerPostingResponse struct {
	ID         int64  `json:"id"`
	EntryID    string `json:"entry_id"`
	AccountRef string `json:"account_ref"`
	Asset      string `json:"asset"`
	Direction  string `json:"direction"`
	Amount     int64  `json:"amount"`
	CreatedAt  int64  `json:"created_at"`
}

type LiabilityReportLine struct {
	Asset             string `json:"asset"`
	UserAvailable     int64  `json:"user_available"`
	UserFrozen        int64  `json:"user_frozen"`
	PendingSettlement int64  `json:"pending_settlement"`
	PendingWithdraw   int64  `json:"pending_withdraw"`
	PlatformFee       int64  `json:"platform_fee_liability"`
	InternalTotal     int64  `json:"internal_total"`
}

type CreateSessionRequest struct {
	SessionID        string `json:"-"`
	UserID           int64  `json:"user_id" binding:"required"`
	WalletAddress    string `json:"wallet_address" binding:"required"`
	SessionPublicKey string `json:"session_public_key" binding:"required"`
	Scope            string `json:"scope"`
	ChainID          int64  `json:"chain_id" binding:"required"`
	Nonce            string `json:"nonce" binding:"required"`
	IssuedAtMillis   int64  `json:"issued_at" binding:"required"`
	ExpiresAtMillis  int64  `json:"expires_at" binding:"required"`
	WalletSignature  string `json:"wallet_signature" binding:"required"`
}

type CreateTradingKeyChallengeRequest struct {
	ChallengeID        string `json:"-"`
	Challenge          string `json:"-"`
	ChallengeExpiresAt int64  `json:"-"`
	WalletAddress      string `json:"wallet_address" binding:"required"`
	ChainID            int64  `json:"chain_id" binding:"required"`
	VaultAddress       string `json:"vault_address" binding:"required"`
}

type TradingKeyChallengeResponse struct {
	ChallengeID        string `json:"challenge_id"`
	Challenge          string `json:"challenge"`
	ChallengeExpiresAt int64  `json:"challenge_expires_at"`
}

type RegisterTradingKeyRequest struct {
	SessionID                string `json:"-"`
	UserID                   int64  `json:"-"`
	WalletAddress            string `json:"wallet_address" binding:"required"`
	ChainID                  int64  `json:"chain_id" binding:"required"`
	VaultAddress             string `json:"vault_address" binding:"required"`
	ChallengeID              string `json:"challenge_id" binding:"required"`
	Challenge                string `json:"challenge" binding:"required"`
	ChallengeExpiresAtMillis int64  `json:"challenge_expires_at" binding:"required"`
	TradingPublicKey         string `json:"trading_public_key" binding:"required"`
	TradingKeyScheme         string `json:"trading_key_scheme" binding:"required"`
	Scope                    string `json:"scope"`
	KeyExpiresAtMillis       int64  `json:"key_expires_at"`
	WalletSignatureStandard  string `json:"wallet_signature_standard" binding:"required"`
	WalletSignature          string `json:"wallet_signature" binding:"required"`
}

type SessionResponse struct {
	SessionID        string `json:"session_id"`
	UserID           int64  `json:"user_id"`
	WalletAddress    string `json:"wallet_address"`
	SessionPublicKey string `json:"session_public_key"`
	Scope            string `json:"scope"`
	ChainID          int64  `json:"chain_id"`
	VaultAddress     string `json:"vault_address"`
	SessionNonce     string `json:"session_nonce"`
	LastOrderNonce   int64  `json:"last_order_nonce"`
	Status           string `json:"status"`
	IssuedAtMillis   int64  `json:"issued_at"`
	ExpiresAtMillis  int64  `json:"expires_at"`
	RevokedAtMillis  int64  `json:"revoked_at"`
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
}

type AdvanceSessionNonceRequest struct {
	SessionID            string
	Nonce                uint64
	AuthorizationWitness *sharedauth.OrderAuthorizationWitness
}

type ListSessionsRequest struct {
	UserID        int64  `form:"user_id"`
	WalletAddress string `form:"wallet_address"`
	VaultAddress  string `form:"vault_address"`
	Status        string `form:"status"`
	Limit         int    `form:"limit"`
}

type ListDepositsRequest struct {
	UserID        int64  `form:"user_id"`
	WalletAddress string `form:"wallet_address"`
	Status        string `form:"status"`
	Limit         int    `form:"limit"`
}

type ListWithdrawalsRequest struct {
	UserID        int64  `form:"user_id"`
	WalletAddress string `form:"wallet_address"`
	Status        string `form:"status"`
	Limit         int    `form:"limit"`
}

type ListRollupForcedWithdrawalsRequest struct {
	WalletAddress string `form:"wallet_address"`
	Status        string `form:"status"`
	Limit         int    `form:"limit"`
}

type DepositResponse struct {
	DepositID     string `json:"deposit_id"`
	UserID        int64  `json:"user_id"`
	WalletAddress string `json:"wallet_address"`
	VaultAddress  string `json:"vault_address"`
	Asset         string `json:"asset"`
	Amount        int64  `json:"amount"`
	ChainName     string `json:"chain_name"`
	NetworkName   string `json:"network_name"`
	TxHash        string `json:"tx_hash"`
	LogIndex      int64  `json:"log_index"`
	BlockNumber   int64  `json:"block_number"`
	Status        string `json:"status"`
	CreditedAt    int64  `json:"credited_at"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

type WithdrawalResponse struct {
	WithdrawalID     string `json:"withdrawal_id"`
	UserID           int64  `json:"user_id"`
	WalletAddress    string `json:"wallet_address"`
	RecipientAddress string `json:"recipient_address"`
	VaultAddress     string `json:"vault_address"`
	Asset            string `json:"asset"`
	Amount           int64  `json:"amount"`
	ChainName        string `json:"chain_name"`
	NetworkName      string `json:"network_name"`
	TxHash           string `json:"tx_hash"`
	LogIndex         int64  `json:"log_index"`
	BlockNumber      int64  `json:"block_number"`
	Status           string `json:"status"`
	ClaimStatus      string `json:"claim_status"`
	ClaimTxHash      string `json:"claim_tx_hash"`
	ClaimSubmittedAt int64  `json:"claim_submitted_at"`
	ClaimedAt        int64  `json:"claimed_at"`
	LastError        string `json:"last_error"`
	DebitedAt        int64  `json:"debited_at"`
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
}

type RollupForcedWithdrawalResponse struct {
	RequestID               int64  `json:"request_id"`
	WalletAddress           string `json:"wallet_address"`
	RecipientAddress        string `json:"recipient_address"`
	Amount                  int64  `json:"amount"`
	RequestedAt             int64  `json:"requested_at"`
	DeadlineAt              int64  `json:"deadline_at"`
	SatisfiedClaimID        string `json:"satisfied_claim_id"`
	SatisfiedAt             int64  `json:"satisfied_at"`
	FrozenAt                int64  `json:"frozen_at"`
	Status                  string `json:"status"`
	MatchedWithdrawalID     string `json:"matched_withdrawal_id"`
	MatchedClaimID          string `json:"matched_claim_id"`
	SatisfactionStatus      string `json:"satisfaction_status"`
	SatisfactionTxHash      string `json:"satisfaction_tx_hash"`
	SatisfactionSubmittedAt int64  `json:"satisfaction_submitted_at"`
	SatisfactionLastError   string `json:"satisfaction_last_error"`
	SatisfactionLastErrorAt int64  `json:"satisfaction_last_error_at"`
	CreatedAt               int64  `json:"created_at"`
	UpdatedAt               int64  `json:"updated_at"`
}

type RollupFreezeStateResponse struct {
	Frozen    bool  `json:"frozen"`
	FrozenAt  int64 `json:"frozen_at"`
	RequestID int64 `json:"request_id"`
	UpdatedAt int64 `json:"updated_at"`
}

type CreateClaimPayoutRequest struct {
	UserID           int64  `json:"user_id" binding:"required"`
	WalletAddress    string `json:"wallet_address" binding:"required"`
	RecipientAddress string `json:"recipient_address" binding:"required"`
}

type ClaimPayoutRequest struct {
	EventID          string `json:"event_id"`
	UserID           int64  `json:"user_id"`
	MarketID         int64  `json:"market_id"`
	WalletAddress    string `json:"wallet_address"`
	RecipientAddress string `json:"recipient_address"`
	PayoutAsset      string `json:"payout_asset"`
	PayoutAmount     int64  `json:"payout_amount"`
}

func (r CreateClaimPayoutRequest) Normalize() CreateClaimPayoutRequest {
	r.WalletAddress = normalizeClaimAddress(r.WalletAddress)
	r.RecipientAddress = normalizeClaimAddress(r.RecipientAddress)
	return r
}

func (r CreateClaimPayoutRequest) ValidateAddresses() (CreateClaimPayoutRequest, error) {
	normalized := r.Normalize()

	walletAddress, err := validateClaimAddress("wallet_address", normalized.WalletAddress)
	if err != nil {
		return CreateClaimPayoutRequest{}, err
	}
	recipientAddress, err := validateClaimAddress("recipient_address", normalized.RecipientAddress)
	if err != nil {
		return CreateClaimPayoutRequest{}, err
	}

	normalized.WalletAddress = walletAddress
	normalized.RecipientAddress = recipientAddress
	return normalized, nil
}

func (r ClaimPayoutRequest) Normalize() ClaimPayoutRequest {
	r.EventID = strings.TrimSpace(r.EventID)
	r.WalletAddress = normalizeClaimAddress(r.WalletAddress)
	r.RecipientAddress = normalizeClaimAddress(r.RecipientAddress)
	r.PayoutAsset = strings.ToUpper(strings.TrimSpace(r.PayoutAsset))
	return r
}

func (r ClaimPayoutRequest) ValidateForQueuedClaim() (ClaimPayoutRequest, error) {
	normalized := r.Normalize()

	if normalized.EventID == "" {
		return ClaimPayoutRequest{}, fmt.Errorf("event_id is required")
	}
	walletAddress, err := validateClaimAddress("wallet_address", normalized.WalletAddress)
	if err != nil {
		return ClaimPayoutRequest{}, err
	}
	recipientAddress, err := validateClaimAddress("recipient_address", normalized.RecipientAddress)
	if err != nil {
		return ClaimPayoutRequest{}, err
	}
	if normalized.PayoutAmount <= 0 {
		return ClaimPayoutRequest{}, fmt.Errorf("payout_amount must be positive")
	}

	normalized.WalletAddress = walletAddress
	normalized.RecipientAddress = recipientAddress
	return normalized, nil
}

func normalizeClaimAddress(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validateClaimAddress(field, value string) (string, error) {
	normalized := normalizeClaimAddress(value)
	if normalized == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	if !common.IsHexAddress(normalized) {
		return "", fmt.Errorf("%s must be a valid EVM address", field)
	}

	address := common.HexToAddress(normalized)
	if address == (common.Address{}) {
		return "", fmt.Errorf("%s must not be zero address", field)
	}
	return strings.ToLower(address.Hex()), nil
}

type ChainTransactionResponse struct {
	ID            int64           `json:"id"`
	BizType       string          `json:"biz_type"`
	RefID         string          `json:"ref_id"`
	ChainName     string          `json:"chain_name"`
	NetworkName   string          `json:"network_name"`
	WalletAddress string          `json:"wallet_address"`
	TxHash        string          `json:"tx_hash"`
	Status        string          `json:"status"`
	Payload       json.RawMessage `json:"payload"`
	ErrorMessage  string          `json:"error_message"`
	AttemptCount  int64           `json:"attempt_count"`
	CreatedAt     int64           `json:"created_at"`
	UpdatedAt     int64           `json:"updated_at"`
}

type ListChainTransactionsRequest struct {
	BizType string `form:"biz_type"`
	RefID   string `form:"ref_id"`
	Status  string `form:"status"`
	Limit   int    `form:"limit"`
}
