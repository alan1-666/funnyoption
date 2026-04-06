package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"funnyoption/internal/api/dto"
	"funnyoption/internal/rollup"

	"github.com/lib/pq"
)

var ErrNotFound = errors.New("resource not found")
var ErrSessionNonceConflict = errors.New("session nonce conflict")
var ErrInvalidMarketCategory = errors.New("market category is invalid")
var ErrTradingKeyChallengeExpired = errors.New("trading key challenge expired")
var ErrTradingKeyChallengeConsumed = errors.New("trading key challenge already used")

const maxInt64Uint64 = uint64(^uint64(0) >> 1)

type QueryStore interface {
	CreateMarket(ctx context.Context, req dto.CreateMarketRequest) (dto.MarketResponse, error)
	CreateSession(ctx context.Context, req dto.CreateSessionRequest) (dto.SessionResponse, error)
	CreateTradingKeyChallenge(ctx context.Context, req dto.CreateTradingKeyChallengeRequest) (dto.TradingKeyChallengeResponse, error)
	RegisterTradingKey(ctx context.Context, req dto.RegisterTradingKeyRequest) (dto.SessionResponse, error)
	CreateClaimRequest(ctx context.Context, req dto.ClaimPayoutRequest) (dto.ChainTransactionResponse, error)
	GetUserProfile(ctx context.Context, req dto.GetUserProfileRequest) (dto.UserProfileResponse, error)
	UpsertUserProfile(ctx context.Context, req dto.UpdateUserProfileRequest, walletAddress string) (dto.UserProfileResponse, error)
	GetOrder(ctx context.Context, orderID string) (dto.OrderResponse, error)
	GetSession(ctx context.Context, sessionID string) (dto.SessionResponse, error)
	GetLatestFreezeByRef(ctx context.Context, refType, refID string) (dto.FreezeResponse, error)
	RevokeSession(ctx context.Context, sessionID string) (dto.SessionResponse, error)
	AdvanceSessionNonce(ctx context.Context, req dto.AdvanceSessionNonceRequest) (dto.SessionResponse, error)
	GetMarket(ctx context.Context, marketID int64) (dto.MarketResponse, error)
	GetMarketResolution(ctx context.Context, marketID int64) (MarketResolutionState, bool, error)
	ListMarkets(ctx context.Context, req dto.ListMarketsRequest) ([]dto.MarketResponse, error)
	ListSessions(ctx context.Context, req dto.ListSessionsRequest) ([]dto.SessionResponse, error)
	ListDeposits(ctx context.Context, req dto.ListDepositsRequest) ([]dto.DepositResponse, error)
	ListWithdrawals(ctx context.Context, req dto.ListWithdrawalsRequest) ([]dto.WithdrawalResponse, error)
	ListRollupForcedWithdrawals(ctx context.Context, req dto.ListRollupForcedWithdrawalsRequest) ([]dto.RollupForcedWithdrawalResponse, error)
	GetRollupFreezeState(ctx context.Context) (dto.RollupFreezeStateResponse, error)
	ListChainTransactions(ctx context.Context, req dto.ListChainTransactionsRequest) ([]dto.ChainTransactionResponse, error)
	ListOrders(ctx context.Context, req dto.ListOrdersRequest) ([]dto.OrderResponse, error)
	ListTrades(ctx context.Context, req dto.ListTradesRequest) ([]dto.TradeResponse, error)
	ListBalances(ctx context.Context, req dto.ListBalancesRequest) ([]dto.BalanceResponse, error)
	ListPositions(ctx context.Context, req dto.ListPositionsRequest) ([]dto.PositionResponse, error)
	ListPayouts(ctx context.Context, req dto.ListPayoutsRequest) ([]dto.PayoutResponse, error)
	ListFreezes(ctx context.Context, req dto.ListFreezesRequest) ([]dto.FreezeResponse, error)
	ListLedgerEntries(ctx context.Context, req dto.ListLedgerEntriesRequest) ([]dto.LedgerEntryResponse, error)
	ListLedgerPostings(ctx context.Context, entryID string) ([]dto.LedgerPostingResponse, error)
	BuildLiabilityReport(ctx context.Context) ([]dto.LiabilityReportLine, error)
}

type SQLStore struct {
	db     *sql.DB
	rollup *rollup.Store
}

type MarketResolutionState struct {
	Status          string
	ResolvedOutcome string
	ResolverType    string
	ResolverRef     string
}

type tradingKeyChallengeRecord struct {
	ChallengeID   string
	WalletAddress string
	ChainID       int64
	VaultAddress  string
	Challenge     string
	ExpiresAt     int64
	ConsumedAt    int64
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) WithRollup(store *rollup.Store) *SQLStore {
	s.rollup = store
	return s
}

func (s *SQLStore) CreateMarket(ctx context.Context, req dto.CreateMarketRequest) (dto.MarketResponse, error) {
	categoryKey := dto.NormalizeMarketCategoryKey(req.CategoryKey, req.Metadata)
	options, err := dto.NormalizeMarketOptions(req.Options)
	if err != nil {
		return dto.MarketResponse{}, dto.ErrInvalidMarketOptions
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return dto.MarketResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	category, err := lookupMarketCategoryTx(ctx, tx, categoryKey)
	if err != nil {
		return dto.MarketResponse{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO markets (
			market_id, title, description, category_id, collateral_asset, status,
			open_at, close_at, resolve_at, resolved_outcome, created_by, metadata, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, '', $10, $11, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
	`,
		req.MarketID,
		strings.TrimSpace(req.Title),
		strings.TrimSpace(req.Description),
		category.CategoryID,
		normalizeAsset(req.CollateralAsset),
		normalizeMarketStatus(req.Status),
		req.OpenAt,
		req.CloseAt,
		req.ResolveAt,
		req.CreatedBy,
		metadataOrDefault(req.Metadata),
	); err != nil {
		return dto.MarketResponse{}, err
	}

	encodedOptions, err := json.Marshal(options)
	if err != nil {
		return dto.MarketResponse{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO market_option_sets (
			market_id, option_schema, version, created_at, updated_at
		)
		VALUES ($1, $2, 1, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
	`, req.MarketID, encodedOptions); err != nil {
		return dto.MarketResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return dto.MarketResponse{}, err
	}
	return s.GetMarket(ctx, req.MarketID)
}

func (s *SQLStore) CreateSession(ctx context.Context, req dto.CreateSessionRequest) (dto.SessionResponse, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return dto.SessionResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	row := tx.QueryRowContext(ctx, `
		INSERT INTO wallet_sessions (
			session_id, user_id, wallet_address, session_public_key, scope, chain_id,
			vault_address, session_nonce, last_order_nonce, status, issued_at, expires_at, revoked_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, '', $7, 0, 'ACTIVE', $8, $9, 0,
		        EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (session_id) DO UPDATE
		SET user_id = EXCLUDED.user_id,
			wallet_address = EXCLUDED.wallet_address,
			session_public_key = EXCLUDED.session_public_key,
			scope = EXCLUDED.scope,
			chain_id = EXCLUDED.chain_id,
			vault_address = EXCLUDED.vault_address,
			session_nonce = EXCLUDED.session_nonce,
			status = 'ACTIVE',
			issued_at = EXCLUDED.issued_at,
			expires_at = EXCLUDED.expires_at,
			revoked_at = 0,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		RETURNING session_id, user_id, wallet_address, session_public_key, scope, chain_id, vault_address,
		          session_nonce, last_order_nonce, status, issued_at, expires_at, revoked_at, created_at, updated_at
	`,
		req.SessionID,
		req.UserID,
		strings.ToLower(strings.TrimSpace(req.WalletAddress)),
		strings.ToLower(strings.TrimSpace(req.SessionPublicKey)),
		normalizeSessionScope(req.Scope),
		req.ChainID,
		strings.TrimSpace(req.Nonce),
		req.IssuedAtMillis,
		req.ExpiresAtMillis,
	)
	session, err := scanSession(row)
	if err != nil {
		return dto.SessionResponse{}, err
	}
	if err := ensureUserProfileTx(ctx, tx, session.UserID, session.WalletAddress); err != nil {
		return dto.SessionResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return dto.SessionResponse{}, err
	}
	return session, nil
}

func (s *SQLStore) CreateTradingKeyChallenge(ctx context.Context, req dto.CreateTradingKeyChallengeRequest) (dto.TradingKeyChallengeResponse, error) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO trading_key_challenges (
			challenge_id, wallet_address, chain_id, vault_address, challenge,
			expires_at, consumed_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, 0,
		        EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
	`,
		strings.TrimSpace(req.ChallengeID),
		normalizeWalletAddress(req.WalletAddress),
		req.ChainID,
		normalizeVaultAddress(req.VaultAddress),
		normalizeChallengeValue(req.Challenge),
		req.ChallengeExpiresAt,
	)
	if err != nil {
		return dto.TradingKeyChallengeResponse{}, err
	}
	return dto.TradingKeyChallengeResponse{
		ChallengeID:        strings.TrimSpace(req.ChallengeID),
		Challenge:          formatChallengeValue(req.Challenge),
		ChallengeExpiresAt: req.ChallengeExpiresAt,
	}, nil
}

func (s *SQLStore) RegisterTradingKey(ctx context.Context, req dto.RegisterTradingKeyRequest) (dto.SessionResponse, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return dto.SessionResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	nowMillis := time.Now().UnixMilli()
	normalizedChallenge := normalizeChallengeValue(req.Challenge)
	normalizedWallet := normalizeWalletAddress(req.WalletAddress)
	normalizedVault := normalizeVaultAddress(req.VaultAddress)
	normalizedPublicKey := normalizePublicKey(req.TradingPublicKey)

	existing, err := getSessionTx(ctx, tx, req.SessionID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return dto.SessionResponse{}, err
	}

	challenge, err := getTradingKeyChallengeTx(ctx, tx, req.ChallengeID)
	if err != nil {
		if errors.Is(err, ErrNotFound) && isIdempotentTradingKeyRetry(existing, req, normalizedChallenge) {
			if err := tx.Commit(); err != nil {
				return dto.SessionResponse{}, err
			}
			return existing, nil
		}
		return dto.SessionResponse{}, err
	}
	if challenge.WalletAddress != normalizedWallet ||
		challenge.ChainID != req.ChainID ||
		challenge.VaultAddress != normalizedVault ||
		challenge.Challenge != normalizedChallenge {
		return dto.SessionResponse{}, ErrNotFound
	}
	if challenge.ExpiresAt < nowMillis {
		if isIdempotentTradingKeyRetry(existing, req, normalizedChallenge) {
			if err := tx.Commit(); err != nil {
				return dto.SessionResponse{}, err
			}
			return existing, nil
		}
		return dto.SessionResponse{}, ErrTradingKeyChallengeExpired
	}
	if challenge.ConsumedAt > 0 {
		if isIdempotentTradingKeyRetry(existing, req, normalizedChallenge) {
			if err := tx.Commit(); err != nil {
				return dto.SessionResponse{}, err
			}
			return existing, nil
		}
		return dto.SessionResponse{}, ErrTradingKeyChallengeConsumed
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE trading_key_challenges
		SET consumed_at = $2,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE challenge_id = $1
		  AND consumed_at = 0
	`, strings.TrimSpace(req.ChallengeID), nowMillis)
	if err != nil {
		return dto.SessionResponse{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dto.SessionResponse{}, err
	}
	if rowsAffected == 0 {
		if isIdempotentTradingKeyRetry(existing, req, normalizedChallenge) {
			if err := tx.Commit(); err != nil {
				return dto.SessionResponse{}, err
			}
			return existing, nil
		}
		return dto.SessionResponse{}, ErrTradingKeyChallengeConsumed
	}

	userID, err := resolveOrCreateUserIDTx(ctx, tx, normalizedWallet)
	if err != nil {
		return dto.SessionResponse{}, err
	}

	sameActive, err := getActiveTradingKeyTx(ctx, tx, req.SessionID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return dto.SessionResponse{}, err
	}

	if sameActive.SessionID != "" &&
		sameActive.WalletAddress == normalizedWallet &&
		sameActive.ChainID == req.ChainID &&
		sameActive.VaultAddress == normalizedVault &&
		(sameActive.ExpiresAtMillis == 0 || sameActive.ExpiresAtMillis >= nowMillis) &&
		strings.EqualFold(sameActive.SessionPublicKey, normalizedPublicKey) {
		if err := tx.Commit(); err != nil {
			return dto.SessionResponse{}, err
		}
		return sameActive, nil
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE wallet_sessions
		SET status = 'ROTATED',
			revoked_at = $3,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE wallet_address = $1
		  AND chain_id = $2
		  AND vault_address = $5
		  AND status = 'ACTIVE'
		  AND session_id <> $4
	`, normalizedWallet, req.ChainID, nowMillis, req.SessionID, normalizedVault); err != nil {
		return dto.SessionResponse{}, err
	}

	row := tx.QueryRowContext(ctx, `
		INSERT INTO wallet_sessions (
			session_id, user_id, wallet_address, session_public_key, scope, chain_id,
			vault_address,
			session_nonce, last_order_nonce, status, issued_at, expires_at, revoked_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, 'ACTIVE', $9, $10, 0,
		        EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (session_id) DO UPDATE
		SET user_id = EXCLUDED.user_id,
			wallet_address = EXCLUDED.wallet_address,
			session_public_key = EXCLUDED.session_public_key,
			scope = EXCLUDED.scope,
			chain_id = EXCLUDED.chain_id,
			vault_address = EXCLUDED.vault_address,
			session_nonce = EXCLUDED.session_nonce,
			status = 'ACTIVE',
			issued_at = EXCLUDED.issued_at,
			expires_at = EXCLUDED.expires_at,
			revoked_at = 0,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		RETURNING session_id, user_id, wallet_address, session_public_key, scope, chain_id, vault_address,
		          session_nonce, last_order_nonce, status, issued_at, expires_at, revoked_at, created_at, updated_at
	`,
		req.SessionID,
		userID,
		normalizedWallet,
		normalizedPublicKey,
		normalizeSessionScope(req.Scope),
		req.ChainID,
		normalizedVault,
		normalizedChallenge,
		nowMillis,
		req.KeyExpiresAtMillis,
	)
	session, err := scanSession(row)
	if err != nil {
		return dto.SessionResponse{}, err
	}
	if err := ensureUserProfileTx(ctx, tx, session.UserID, session.WalletAddress); err != nil {
		return dto.SessionResponse{}, err
	}
	if s.rollup != nil {
		entry, buildErr := buildTradingKeyAuthorizedEntry(session, req, req.WalletSignature)
		if buildErr != nil {
			return dto.SessionResponse{}, buildErr
		}
		if err := s.rollup.AppendEntriesTx(ctx, tx, []rollup.JournalAppend{entry}); err != nil {
			return dto.SessionResponse{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return dto.SessionResponse{}, err
	}
	return session, nil
}

func (s *SQLStore) GetUserProfile(ctx context.Context, req dto.GetUserProfileRequest) (dto.UserProfileResponse, error) {
	query := `
		SELECT user_id, wallet_address, display_name, avatar_preset, created_at, updated_at
		FROM user_profiles
	`
	var (
		args    []any
		filters []string
	)

	if req.UserID > 0 {
		args = append(args, req.UserID)
		filters = append(filters, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if walletAddress := normalizeWalletAddress(req.WalletAddress); walletAddress != "" {
		args = append(args, walletAddress)
		filters = append(filters, fmt.Sprintf("wallet_address = $%d", len(args)))
	}
	if len(filters) == 0 {
		return dto.UserProfileResponse{}, ErrNotFound
	}

	query += " WHERE " + strings.Join(filters, " AND ")
	query += " ORDER BY updated_at DESC LIMIT 1"
	row := s.db.QueryRowContext(ctx, query, args...)
	return scanUserProfile(row)
}

func (s *SQLStore) UpsertUserProfile(ctx context.Context, req dto.UpdateUserProfileRequest, walletAddress string) (dto.UserProfileResponse, error) {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO user_profiles (
			user_id, wallet_address, display_name, avatar_preset, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (user_id) DO UPDATE
		SET wallet_address = EXCLUDED.wallet_address,
			display_name = EXCLUDED.display_name,
			avatar_preset = EXCLUDED.avatar_preset,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		RETURNING user_id, wallet_address, display_name, avatar_preset, created_at, updated_at
	`,
		req.UserID,
		normalizeWalletAddress(walletAddress),
		dto.NormalizeUserDisplayName(req.DisplayName),
		req.AvatarPreset,
	)
	return scanUserProfile(row)
}

func (s *SQLStore) CreateClaimRequest(ctx context.Context, req dto.ClaimPayoutRequest) (dto.ChainTransactionResponse, error) {
	existing, err := s.getChainTransactionByRef(ctx, "CLAIM", req.EventID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return dto.ChainTransactionResponse{}, err
	}

	var (
		marketID     int64
		payoutAsset  string
		payoutAmount int64
	)
	err = s.db.QueryRowContext(ctx, `
		SELECT market_id, payout_asset, payout_amount
		FROM settlement_payouts
		WHERE event_id = $1
		  AND user_id = $2
	`, strings.TrimSpace(req.EventID), req.UserID).Scan(&marketID, &payoutAsset, &payoutAmount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ChainTransactionResponse{}, ErrNotFound
		}
		return dto.ChainTransactionResponse{}, err
	}

	req.MarketID = marketID
	req.PayoutAsset = strings.ToUpper(strings.TrimSpace(payoutAsset))
	req.PayoutAmount = payoutAmount

	payload, err := json.Marshal(req)
	if err != nil {
		return dto.ChainTransactionResponse{}, err
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO chain_transactions (
			biz_type, ref_id, chain_name, network_name, wallet_address, tx_hash,
			status, payload, error_message, attempt_count, created_at, updated_at
		)
		VALUES ('CLAIM', $1, 'bsc', 'testnet', $2, '', 'PENDING', $3, '', 0,
		        EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		RETURNING id, biz_type, ref_id, chain_name, network_name, wallet_address, tx_hash,
		          status, payload, error_message, attempt_count, created_at, updated_at
	`, strings.TrimSpace(req.EventID), normalizeWalletAddress(req.WalletAddress), payload)
	return scanChainTransaction(row)
}

func (s *SQLStore) GetOrder(ctx context.Context, orderID string) (dto.OrderResponse, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT order_id, client_order_id, command_id, user_id, market_id, outcome, side, order_type,
		       time_in_force, collateral_asset, freeze_id, freeze_asset, freeze_amount, price, quantity,
		       filled_quantity, remaining_quantity, status, cancel_reason, created_at, updated_at
		FROM orders
		WHERE order_id = $1
	`, strings.TrimSpace(orderID))
	return scanOrder(row)
}

func (s *SQLStore) GetSession(ctx context.Context, sessionID string) (dto.SessionResponse, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT session_id, user_id, wallet_address, session_public_key, scope, chain_id, vault_address,
		       session_nonce, last_order_nonce, status, issued_at, expires_at, revoked_at, created_at, updated_at
		FROM wallet_sessions
		WHERE session_id = $1
	`, strings.TrimSpace(sessionID))
	return scanSession(row)
}

func (s *SQLStore) RevokeSession(ctx context.Context, sessionID string) (dto.SessionResponse, error) {
	row := s.db.QueryRowContext(ctx, `
		UPDATE wallet_sessions
		SET status = 'REVOKED',
			revoked_at = EXTRACT(EPOCH FROM NOW())::BIGINT * 1000,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE session_id = $1
		RETURNING session_id, user_id, wallet_address, session_public_key, scope, chain_id, vault_address,
		          session_nonce, last_order_nonce, status, issued_at, expires_at, revoked_at, created_at, updated_at
	`, strings.TrimSpace(sessionID))
	return scanSession(row)
}

func (s *SQLStore) AdvanceSessionNonce(ctx context.Context, req dto.AdvanceSessionNonceRequest) (dto.SessionResponse, error) {
	if req.Nonce > maxInt64Uint64 {
		return dto.SessionResponse{}, fmt.Errorf("order nonce exceeds supported range")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return dto.SessionResponse{}, err
	}
	defer tx.Rollback()

	now := time.Now()
	row := tx.QueryRowContext(ctx, `
		UPDATE wallet_sessions
		SET last_order_nonce = $2,
			updated_at = $3
		WHERE session_id = $1
		  AND status = 'ACTIVE'
		  AND last_order_nonce < $2
		RETURNING session_id, user_id, wallet_address, session_public_key, scope, chain_id, vault_address,
		          session_nonce, last_order_nonce, status, issued_at, expires_at, revoked_at, created_at, updated_at
	`, strings.TrimSpace(req.SessionID), int64(req.Nonce), now.Unix())

	session, err := scanSession(row)
	if err == nil {
		if s.rollup != nil {
			entry, buildErr := buildNonceAdvanceEntry(session, req.Nonce, now.UnixMilli(), req.AuthorizationWitness)
			if buildErr != nil {
				return dto.SessionResponse{}, buildErr
			}
			if err := s.rollup.AppendEntriesTx(ctx, tx, []rollup.JournalAppend{entry}); err != nil {
				return dto.SessionResponse{}, err
			}
		}
		if err := tx.Commit(); err != nil {
			return dto.SessionResponse{}, err
		}
		return session, nil
	}
	if errors.Is(err, ErrNotFound) {
		lookupRow := tx.QueryRowContext(ctx, `
			SELECT session_id, user_id, wallet_address, session_public_key, scope, chain_id, vault_address,
			       session_nonce, last_order_nonce, status, issued_at, expires_at, revoked_at, created_at, updated_at
			FROM wallet_sessions
			WHERE session_id = $1
		`, strings.TrimSpace(req.SessionID))
		_, lookupErr := scanSession(lookupRow)
		if lookupErr == nil {
			return dto.SessionResponse{}, ErrSessionNonceConflict
		}
		if errors.Is(lookupErr, ErrNotFound) {
			return dto.SessionResponse{}, ErrNotFound
		}
		return dto.SessionResponse{}, lookupErr
	}
	return dto.SessionResponse{}, err
}

func (s *SQLStore) GetMarket(ctx context.Context, marketID int64) (dto.MarketResponse, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT market_id, title, description, collateral_asset, status, open_at, close_at, resolve_at,
		       resolved_outcome, created_by, metadata, created_at, updated_at
		FROM markets
		WHERE market_id = $1
	`, marketID)
	item, err := scanMarket(row)
	if err != nil {
		return dto.MarketResponse{}, err
	}
	items := []dto.MarketResponse{item}
	if err := s.attachMarketRuntimeAt(ctx, items, time.Now().Unix()); err != nil {
		return dto.MarketResponse{}, err
	}
	return items[0], nil
}

func (s *SQLStore) GetMarketResolution(ctx context.Context, marketID int64) (MarketResolutionState, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT status, resolved_outcome, resolver_type, resolver_ref
		FROM market_resolutions
		WHERE market_id = $1
	`, marketID)
	var state MarketResolutionState
	if err := row.Scan(&state.Status, &state.ResolvedOutcome, &state.ResolverType, &state.ResolverRef); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MarketResolutionState{}, false, nil
		}
		return MarketResolutionState{}, false, err
	}
	return state, true, nil
}

func (s *SQLStore) GetLatestFreezeByRef(ctx context.Context, refType, refID string) (dto.FreezeResponse, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT freeze_id, user_id, asset, ref_type, ref_id, original_amount, remaining_amount, status, created_at, updated_at
		FROM freeze_records
		WHERE ref_type = $1 AND ref_id = $2
		ORDER BY created_at DESC, freeze_id DESC
		LIMIT 1
	`, strings.ToUpper(strings.TrimSpace(refType)), strings.TrimSpace(refID))
	return scanFreeze(row)
}

func (s *SQLStore) ListMarkets(ctx context.Context, req dto.ListMarketsRequest) ([]dto.MarketResponse, error) {
	var (
		args    []any
		filters []string
	)
	nowUnix := time.Now().Unix()
	oracleResolutionFilter := "UPPER(COALESCE(metadata->'resolution'->>'mode', '')) = 'ORACLE_PRICE'"
	query := `
		SELECT market_id, title, description, collateral_asset, status, open_at, close_at, resolve_at,
		       resolved_outcome, created_by, metadata, created_at, updated_at
		FROM markets
	`

	if status := normalizeOptional(req.Status); status != "" {
		args = append(args, nowUnix)
		nowPlaceholder := len(args)
		switch status {
		case "OPEN":
			filters = append(filters, fmt.Sprintf("status = 'OPEN' AND (close_at <= 0 OR close_at > $%d)", nowPlaceholder))
		case "CLOSED":
			filters = append(filters, fmt.Sprintf(`(
				status = 'CLOSED'
				OR (
					status = 'OPEN'
					AND close_at > 0
					AND close_at <= $%d
					AND (
						%s
						OR (resolve_at > 0 AND resolve_at > $%d)
					)
				)
			)`, nowPlaceholder, oracleResolutionFilter, nowPlaceholder))
		case "WAITING_RESOLUTION":
			filters = append(filters, fmt.Sprintf(`(
				status = 'WAITING_RESOLUTION'
				OR (
					status = 'OPEN'
					AND close_at > 0
					AND close_at <= $%d
					AND NOT (%s)
					AND (resolve_at <= 0 OR resolve_at <= $%d)
				)
			)`, nowPlaceholder, oracleResolutionFilter, nowPlaceholder))
		default:
			args = args[:len(args)-1]
			args = append(args, status)
			filters = append(filters, fmt.Sprintf("status = $%d", len(args)))
		}
	}
	if req.CreatedBy > 0 {
		args = append(args, req.CreatedBy)
		filters = append(filters, fmt.Sprintf("created_by = $%d", len(args)))
	}
	if categoryKey := dto.NormalizeMarketCategoryFilter(req.CategoryKey); categoryKey != "" {
		args = append(args, categoryKey)
		filters = append(filters, fmt.Sprintf("category_id = (SELECT category_id FROM market_categories WHERE category_key = $%d)", len(args)))
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}

	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY created_at DESC, market_id DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var markets []dto.MarketResponse
	for rows.Next() {
		item, err := scanMarket(rows)
		if err != nil {
			return nil, err
		}
		markets = append(markets, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.attachMarketRuntimeAt(ctx, markets, nowUnix); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(markets), nil
}

func (s *SQLStore) ListSessions(ctx context.Context, req dto.ListSessionsRequest) ([]dto.SessionResponse, error) {
	var (
		args    []any
		filters []string
	)
	query := `
		SELECT session_id, user_id, wallet_address, session_public_key, scope, chain_id, vault_address,
		       session_nonce, last_order_nonce, status, issued_at, expires_at, revoked_at, created_at, updated_at
		FROM wallet_sessions
	`

	if req.UserID > 0 {
		args = append(args, req.UserID)
		filters = append(filters, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if wallet := normalizeWalletAddress(req.WalletAddress); wallet != "" {
		args = append(args, wallet)
		filters = append(filters, fmt.Sprintf("wallet_address = $%d", len(args)))
	}
	if vault := normalizeVaultAddress(req.VaultAddress); vault != "" {
		args = append(args, vault)
		filters = append(filters, fmt.Sprintf("vault_address = $%d", len(args)))
	}
	if status := normalizeOptional(req.Status); status != "" {
		args = append(args, status)
		filters = append(filters, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}

	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY created_at DESC, session_id DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.SessionResponse
	for rows.Next() {
		item, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(items), nil
}

func (s *SQLStore) ListDeposits(ctx context.Context, req dto.ListDepositsRequest) ([]dto.DepositResponse, error) {
	var (
		args    []any
		filters []string
	)
	query := `
		SELECT deposit_id, user_id, wallet_address, vault_address, asset, amount,
		       chain_name, network_name, tx_hash, log_index, block_number, status,
		       credited_at, created_at, updated_at
		FROM chain_deposits
	`

	if req.UserID > 0 {
		args = append(args, req.UserID)
		filters = append(filters, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if wallet := normalizeWalletAddress(req.WalletAddress); wallet != "" {
		args = append(args, wallet)
		filters = append(filters, fmt.Sprintf("wallet_address = $%d", len(args)))
	}
	if status := normalizeOptional(req.Status); status != "" {
		args = append(args, status)
		filters = append(filters, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}

	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY created_at DESC, deposit_id DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.DepositResponse
	for rows.Next() {
		item, err := scanDeposit(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(items), nil
}

func (s *SQLStore) ListWithdrawals(ctx context.Context, req dto.ListWithdrawalsRequest) ([]dto.WithdrawalResponse, error) {
	var (
		args    []any
		filters []string
	)
	query := `
		SELECT w.withdrawal_id,
		       w.user_id,
		       w.wallet_address,
		       w.recipient_address,
		       w.vault_address,
		       w.asset,
		       w.amount,
		       w.chain_name,
		       w.network_name,
		       w.tx_hash,
		       w.log_index,
		       w.block_number,
		       CASE
		           WHEN aw.claim_status = 'CLAIMED' THEN 'CLAIMED'
		           WHEN aw.claim_status = 'CLAIM_SUBMITTED' THEN 'CLAIM_SUBMITTED'
		           WHEN aw.claim_status = 'CLAIMABLE' THEN 'CLAIMABLE'
		           WHEN aw.claim_status = 'FAILED' THEN 'CLAIM_FAILED'
		           ELSE w.status
		       END AS effective_status,
		       COALESCE(aw.claim_status, ''),
		       COALESCE(aw.claim_tx_hash, ''),
		       COALESCE(aw.claim_submitted_at, 0),
		       COALESCE(aw.claimed_at, 0),
		       COALESCE(aw.last_error, ''),
		       w.debited_at,
		       w.created_at,
		       w.updated_at
		FROM chain_withdrawals w
		LEFT JOIN rollup_accepted_withdrawals aw
		  ON aw.withdrawal_id = w.withdrawal_id
	`

	if req.UserID > 0 {
		args = append(args, req.UserID)
		filters = append(filters, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if wallet := normalizeWalletAddress(req.WalletAddress); wallet != "" {
		args = append(args, wallet)
		filters = append(filters, fmt.Sprintf("wallet_address = $%d", len(args)))
	}
	if status := normalizeOptional(req.Status); status != "" {
		args = append(args, status)
		filters = append(filters, fmt.Sprintf(`(
			CASE
			    WHEN aw.claim_status = 'CLAIMED' THEN 'CLAIMED'
			    WHEN aw.claim_status = 'CLAIM_SUBMITTED' THEN 'CLAIM_SUBMITTED'
			    WHEN aw.claim_status = 'CLAIMABLE' THEN 'CLAIMABLE'
			    WHEN aw.claim_status = 'FAILED' THEN 'CLAIM_FAILED'
			    ELSE w.status
			END
		) = $%d`, len(args)))
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}

	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY w.created_at DESC, w.withdrawal_id DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.WithdrawalResponse
	for rows.Next() {
		item, err := scanWithdrawal(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(items), nil
}

func (s *SQLStore) ListRollupForcedWithdrawals(ctx context.Context, req dto.ListRollupForcedWithdrawalsRequest) ([]dto.RollupForcedWithdrawalResponse, error) {
	var (
		args    []any
		filters []string
	)
	query := `
		SELECT request_id, wallet_address, recipient_address, amount,
		       requested_at, deadline_at, satisfied_claim_id, satisfied_at,
		       frozen_at, status, matched_withdrawal_id, matched_claim_id,
		       satisfaction_status, satisfaction_tx_hash, satisfaction_submitted_at,
		       satisfaction_last_error, satisfaction_last_error_at, created_at, updated_at
		FROM rollup_forced_withdrawal_requests
	`

	if wallet := normalizeWalletAddress(req.WalletAddress); wallet != "" {
		args = append(args, wallet)
		filters = append(filters, fmt.Sprintf("wallet_address = $%d", len(args)))
	}
	if status := normalizeOptional(req.Status); status != "" {
		args = append(args, status)
		filters = append(filters, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}

	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY request_id DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]dto.RollupForcedWithdrawalResponse, 0)
	for rows.Next() {
		var item dto.RollupForcedWithdrawalResponse
		if err := rows.Scan(
			&item.RequestID,
			&item.WalletAddress,
			&item.RecipientAddress,
			&item.Amount,
			&item.RequestedAt,
			&item.DeadlineAt,
			&item.SatisfiedClaimID,
			&item.SatisfiedAt,
			&item.FrozenAt,
			&item.Status,
			&item.MatchedWithdrawalID,
			&item.MatchedClaimID,
			&item.SatisfactionStatus,
			&item.SatisfactionTxHash,
			&item.SatisfactionSubmittedAt,
			&item.SatisfactionLastError,
			&item.SatisfactionLastErrorAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(items), nil
}

func (s *SQLStore) GetRollupFreezeState(ctx context.Context) (dto.RollupFreezeStateResponse, error) {
	var item dto.RollupFreezeStateResponse
	err := s.db.QueryRowContext(ctx, `
		SELECT frozen, frozen_at, request_id, updated_at
		FROM rollup_freeze_state
		WHERE id = TRUE
	`).Scan(&item.Frozen, &item.FrozenAt, &item.RequestID, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.RollupFreezeStateResponse{}, ErrNotFound
		}
		return dto.RollupFreezeStateResponse{}, err
	}
	return item, nil
}

func (s *SQLStore) ListChainTransactions(ctx context.Context, req dto.ListChainTransactionsRequest) ([]dto.ChainTransactionResponse, error) {
	var (
		args    []any
		filters []string
	)
	query := `
		SELECT id, biz_type, ref_id, chain_name, network_name, wallet_address, tx_hash,
		       status, payload, error_message, attempt_count, created_at, updated_at
		FROM chain_transactions
	`

	if bizType := normalizeOptional(req.BizType); bizType != "" {
		args = append(args, bizType)
		filters = append(filters, fmt.Sprintf("biz_type = $%d", len(args)))
	}
	if refID := strings.TrimSpace(req.RefID); refID != "" {
		args = append(args, refID)
		filters = append(filters, fmt.Sprintf("ref_id = $%d", len(args)))
	}
	if status := normalizeOptional(req.Status); status != "" {
		args = append(args, status)
		filters = append(filters, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.ChainTransactionResponse
	for rows.Next() {
		item, err := scanChainTransaction(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(items), nil
}

func (s *SQLStore) ListOrders(ctx context.Context, req dto.ListOrdersRequest) ([]dto.OrderResponse, error) {
	var (
		args    []any
		filters []string
	)
	query := `
		SELECT order_id, client_order_id, command_id, user_id, market_id, outcome, side, order_type,
		       time_in_force, collateral_asset, freeze_id, freeze_asset, freeze_amount, price, quantity,
		       filled_quantity, remaining_quantity, status, cancel_reason, created_at, updated_at
		FROM orders
	`

	if req.UserID > 0 {
		args = append(args, req.UserID)
		filters = append(filters, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if req.MarketID > 0 {
		args = append(args, req.MarketID)
		filters = append(filters, fmt.Sprintf("market_id = $%d", len(args)))
	}
	if status := normalizeOptional(req.Status); status != "" {
		args = append(args, status)
		filters = append(filters, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}

	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY created_at DESC, order_id DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []dto.OrderResponse
	for rows.Next() {
		item, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(orders), nil
}

func (s *SQLStore) ListTrades(ctx context.Context, req dto.ListTradesRequest) ([]dto.TradeResponse, error) {
	var (
		args    []any
		filters []string
	)
	query := `
		SELECT trade_id, sequence_no, market_id, outcome, collateral_asset, price, quantity,
		       taker_order_id, maker_order_id, taker_user_id, maker_user_id, taker_side, maker_side, occurred_at
		FROM trades
	`

	if req.MarketID > 0 {
		args = append(args, req.MarketID)
		filters = append(filters, fmt.Sprintf("market_id = $%d", len(args)))
	}
	if req.UserID > 0 {
		args = append(args, req.UserID)
		filters = append(filters, fmt.Sprintf("(taker_user_id = $%d OR maker_user_id = $%d)", len(args), len(args)))
	}
	if outcome := normalizeOptional(req.Outcome); outcome != "" {
		args = append(args, outcome)
		filters = append(filters, fmt.Sprintf("outcome = $%d", len(args)))
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}

	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY sequence_no DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []dto.TradeResponse
	for rows.Next() {
		item, err := scanTrade(rows)
		if err != nil {
			return nil, err
		}
		trades = append(trades, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(trades), nil
}

func (s *SQLStore) ListBalances(ctx context.Context, req dto.ListBalancesRequest) ([]dto.BalanceResponse, error) {
	acceptedVisible, err := s.acceptedReadTruthVisible(ctx)
	if err != nil {
		return nil, err
	}
	if acceptedVisible {
		return s.listAcceptedBalances(ctx, req)
	}

	var (
		args    []any
		filters []string
	)
	query := `
		SELECT user_id, asset, available, frozen, created_at, updated_at
		FROM account_balances
	`

	args = append(args, req.UserID)
	filters = append(filters, fmt.Sprintf("user_id = $%d", len(args)))
	if asset := normalizeOptional(req.Asset); asset != "" {
		args = append(args, asset)
		filters = append(filters, fmt.Sprintf("asset = $%d", len(args)))
	}
	query += " WHERE " + strings.Join(filters, " AND ")
	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY asset ASC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []dto.BalanceResponse
	for rows.Next() {
		var item dto.BalanceResponse
		if err := rows.Scan(&item.UserID, &item.Asset, &item.Available, &item.Frozen, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		balances = append(balances, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(balances), nil
}

func (s *SQLStore) ListPositions(ctx context.Context, req dto.ListPositionsRequest) ([]dto.PositionResponse, error) {
	acceptedVisible, err := s.acceptedReadTruthVisible(ctx)
	if err != nil {
		return nil, err
	}
	if acceptedVisible {
		return s.listAcceptedPositions(ctx, req)
	}

	var (
		args    []any
		filters []string
	)
	query := `
		SELECT market_id, user_id, outcome, position_asset, quantity, settled_quantity, created_at, updated_at
		FROM positions
	`

	args = append(args, req.UserID)
	filters = append(filters, fmt.Sprintf("user_id = $%d", len(args)))
	if req.MarketID > 0 {
		args = append(args, req.MarketID)
		filters = append(filters, fmt.Sprintf("market_id = $%d", len(args)))
	}
	if outcome := normalizeOptional(req.Outcome); outcome != "" {
		args = append(args, outcome)
		filters = append(filters, fmt.Sprintf("outcome = $%d", len(args)))
	}
	query += " WHERE " + strings.Join(filters, " AND ")
	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY updated_at DESC, market_id DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []dto.PositionResponse
	for rows.Next() {
		var item dto.PositionResponse
		if err := rows.Scan(
			&item.MarketID,
			&item.UserID,
			&item.Outcome,
			&item.PositionAsset,
			&item.Quantity,
			&item.SettledQuantity,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		positions = append(positions, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(positions), nil
}

func (s *SQLStore) ListPayouts(ctx context.Context, req dto.ListPayoutsRequest) ([]dto.PayoutResponse, error) {
	acceptedVisible, err := s.acceptedReadTruthVisible(ctx)
	if err != nil {
		return nil, err
	}
	if acceptedVisible {
		return s.listAcceptedPayouts(ctx, req)
	}

	var (
		args    []any
		filters []string
	)
	query := `
		SELECT event_id, market_id, user_id, winning_outcome, position_asset, settled_quantity,
		       payout_asset, payout_amount, status, created_at, updated_at
		FROM settlement_payouts
	`

	args = append(args, req.UserID)
	filters = append(filters, fmt.Sprintf("user_id = $%d", len(args)))
	if req.MarketID > 0 {
		args = append(args, req.MarketID)
		filters = append(filters, fmt.Sprintf("market_id = $%d", len(args)))
	}
	query += " WHERE " + strings.Join(filters, " AND ")
	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY created_at DESC, event_id DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payouts []dto.PayoutResponse
	for rows.Next() {
		var item dto.PayoutResponse
		if err := rows.Scan(
			&item.EventID,
			&item.MarketID,
			&item.UserID,
			&item.WinningOutcome,
			&item.PositionAsset,
			&item.SettledQuantity,
			&item.PayoutAsset,
			&item.PayoutAmount,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		payouts = append(payouts, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(payouts), nil
}

func (s *SQLStore) acceptedReadTruthVisible(ctx context.Context) (bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM rollup_accepted_batches
			LIMIT 1
		)
	`)
	var visible bool
	if err := row.Scan(&visible); err != nil {
		return false, err
	}
	return visible, nil
}

func (s *SQLStore) listAcceptedBalances(ctx context.Context, req dto.ListBalancesRequest) ([]dto.BalanceResponse, error) {
	var (
		args    []any
		filters []string
	)
	query := `
		SELECT account_id, asset, available, frozen, created_at, updated_at
		FROM rollup_accepted_balances
	`

	args = append(args, req.UserID)
	filters = append(filters, fmt.Sprintf("account_id = $%d", len(args)))
	if asset := normalizeOptional(req.Asset); asset != "" {
		args = append(args, asset)
		filters = append(filters, fmt.Sprintf("asset = $%d", len(args)))
	}
	query += " WHERE " + strings.Join(filters, " AND ")
	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY asset ASC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []dto.BalanceResponse
	for rows.Next() {
		var item dto.BalanceResponse
		if err := rows.Scan(&item.UserID, &item.Asset, &item.Available, &item.Frozen, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		balances = append(balances, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(balances), nil
}

func (s *SQLStore) listAcceptedPositions(ctx context.Context, req dto.ListPositionsRequest) ([]dto.PositionResponse, error) {
	var (
		args    []any
		filters []string
	)
	query := `
		SELECT market_id, account_id, outcome, position_asset, quantity, settled_quantity, created_at, updated_at
		FROM rollup_accepted_positions
	`

	args = append(args, req.UserID)
	filters = append(filters, fmt.Sprintf("account_id = $%d", len(args)))
	if req.MarketID > 0 {
		args = append(args, req.MarketID)
		filters = append(filters, fmt.Sprintf("market_id = $%d", len(args)))
	}
	if outcome := normalizeOptional(req.Outcome); outcome != "" {
		args = append(args, outcome)
		filters = append(filters, fmt.Sprintf("outcome = $%d", len(args)))
	}
	query += " WHERE " + strings.Join(filters, " AND ")
	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY updated_at DESC, market_id DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []dto.PositionResponse
	for rows.Next() {
		var item dto.PositionResponse
		if err := rows.Scan(
			&item.MarketID,
			&item.UserID,
			&item.Outcome,
			&item.PositionAsset,
			&item.Quantity,
			&item.SettledQuantity,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		positions = append(positions, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(positions), nil
}

func (s *SQLStore) listAcceptedPayouts(ctx context.Context, req dto.ListPayoutsRequest) ([]dto.PayoutResponse, error) {
	var (
		args    []any
		filters []string
	)
	query := `
		SELECT event_id, market_id, user_id, winning_outcome, position_asset, settled_quantity,
		       payout_asset, payout_amount, status, created_at, updated_at
		FROM rollup_accepted_payouts
	`

	args = append(args, req.UserID)
	filters = append(filters, fmt.Sprintf("user_id = $%d", len(args)))
	if req.MarketID > 0 {
		args = append(args, req.MarketID)
		filters = append(filters, fmt.Sprintf("market_id = $%d", len(args)))
	}
	query += " WHERE " + strings.Join(filters, " AND ")
	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY created_at DESC, event_id DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payouts []dto.PayoutResponse
	for rows.Next() {
		var item dto.PayoutResponse
		if err := rows.Scan(
			&item.EventID,
			&item.MarketID,
			&item.UserID,
			&item.WinningOutcome,
			&item.PositionAsset,
			&item.SettledQuantity,
			&item.PayoutAsset,
			&item.PayoutAmount,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		payouts = append(payouts, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(payouts), nil
}

func (s *SQLStore) ListFreezes(ctx context.Context, req dto.ListFreezesRequest) ([]dto.FreezeResponse, error) {
	var (
		args    []any
		filters []string
	)
	query := `
		SELECT freeze_id, user_id, asset, ref_type, ref_id, original_amount, remaining_amount, status, created_at, updated_at
		FROM freeze_records
	`
	if req.UserID > 0 {
		args = append(args, req.UserID)
		filters = append(filters, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if status := normalizeOptional(req.Status); status != "" {
		args = append(args, status)
		filters = append(filters, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY created_at DESC, freeze_id DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var freezes []dto.FreezeResponse
	for rows.Next() {
		var item dto.FreezeResponse
		if err := rows.Scan(
			&item.FreezeID,
			&item.UserID,
			&item.Asset,
			&item.RefType,
			&item.RefID,
			&item.OriginalAmount,
			&item.RemainingAmount,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		freezes = append(freezes, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(freezes), nil
}

func (s *SQLStore) ListLedgerEntries(ctx context.Context, req dto.ListLedgerEntriesRequest) ([]dto.LedgerEntryResponse, error) {
	var (
		args    []any
		filters []string
	)
	query := `
		SELECT e.entry_id, e.biz_type, e.ref_id, e.status, COUNT(p.id) AS posting_count, e.created_at
		FROM ledger_entries e
		LEFT JOIN ledger_postings p ON p.entry_id = e.entry_id
	`
	if bizType := normalizeOptional(req.BizType); bizType != "" {
		args = append(args, bizType)
		filters = append(filters, fmt.Sprintf("e.biz_type = $%d", len(args)))
	}
	if refID := strings.TrimSpace(req.RefID); refID != "" {
		args = append(args, refID)
		filters = append(filters, fmt.Sprintf("e.ref_id = $%d", len(args)))
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	query += " GROUP BY e.entry_id, e.biz_type, e.ref_id, e.status, e.created_at"
	args = append(args, normalizeLimit(req.Limit))
	query += fmt.Sprintf(" ORDER BY e.created_at DESC, e.entry_id DESC LIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []dto.LedgerEntryResponse
	for rows.Next() {
		var item dto.LedgerEntryResponse
		if err := rows.Scan(&item.EntryID, &item.BizType, &item.RefID, &item.Status, &item.PostingCnt, &item.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(entries), nil
}

func (s *SQLStore) ListLedgerPostings(ctx context.Context, entryID string) ([]dto.LedgerPostingResponse, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, entry_id, account_ref, asset, direction, amount, created_at
		FROM ledger_postings
		WHERE entry_id = $1
		ORDER BY id ASC
	`, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var postings []dto.LedgerPostingResponse
	for rows.Next() {
		var item dto.LedgerPostingResponse
		if err := rows.Scan(&item.ID, &item.EntryID, &item.AccountRef, &item.Asset, &item.Direction, &item.Amount, &item.CreatedAt); err != nil {
			return nil, err
		}
		postings = append(postings, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(postings), nil
}

func (s *SQLStore) BuildLiabilityReport(ctx context.Context) ([]dto.LiabilityReportLine, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT asset, SUM(available) AS user_available, SUM(frozen) AS user_frozen
		FROM account_balances
		GROUP BY asset
		ORDER BY asset ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []dto.LiabilityReportLine
	for rows.Next() {
		var item dto.LiabilityReportLine
		if err := rows.Scan(&item.Asset, &item.UserAvailable, &item.UserFrozen); err != nil {
			return nil, err
		}
		item.InternalTotal = item.UserAvailable + item.UserFrozen
		lines = append(lines, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	indexByAsset := make(map[string]int)
	for i := range lines {
		indexByAsset[lines[i].Asset] = i
	}

	withdrawRows, err := s.db.QueryContext(ctx, `
		SELECT asset, SUM(amount) AS pending_withdraw
		FROM chain_withdrawals
		WHERE status IN ('QUEUED', 'DEBITED')
		GROUP BY asset
	`)
	if err != nil {
		return nil, err
	}
	defer withdrawRows.Close()

	for withdrawRows.Next() {
		var (
			asset   string
			pending int64
		)
		if err := withdrawRows.Scan(&asset, &pending); err != nil {
			return nil, err
		}
		idx, ok := indexByAsset[asset]
		if !ok {
			lines = append(lines, dto.LiabilityReportLine{Asset: asset})
			idx = len(lines) - 1
			indexByAsset[asset] = idx
		}
		lines[idx].PendingWithdraw = pending
		lines[idx].InternalTotal = lines[idx].UserAvailable + lines[idx].UserFrozen + lines[idx].PendingWithdraw + lines[idx].PendingSettlement + lines[idx].PlatformFee
	}

	if err := withdrawRows.Err(); err != nil {
		return nil, err
	}
	return normalizeCollectionItems(lines), nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanMarket(row scanner) (dto.MarketResponse, error) {
	var (
		item     dto.MarketResponse
		metadata []byte
	)
	if err := row.Scan(
		&item.MarketID,
		&item.Title,
		&item.Description,
		&item.CollateralAsset,
		&item.Status,
		&item.OpenAt,
		&item.CloseAt,
		&item.ResolveAt,
		&item.ResolvedOutcome,
		&item.CreatedBy,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.MarketResponse{}, ErrNotFound
		}
		return dto.MarketResponse{}, err
	}
	item.Metadata = json.RawMessage(metadata)
	return item, nil
}

func scanOrder(row scanner) (dto.OrderResponse, error) {
	var item dto.OrderResponse
	if err := row.Scan(
		&item.OrderID,
		&item.ClientOrderID,
		&item.CommandID,
		&item.UserID,
		&item.MarketID,
		&item.Outcome,
		&item.Side,
		&item.OrderType,
		&item.TimeInForce,
		&item.CollateralAsset,
		&item.FreezeID,
		&item.FreezeAsset,
		&item.FreezeAmount,
		&item.Price,
		&item.Quantity,
		&item.FilledQuantity,
		&item.RemainingQuantity,
		&item.Status,
		&item.CancelReason,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.OrderResponse{}, ErrNotFound
		}
		return dto.OrderResponse{}, err
	}
	return item, nil
}

func scanFreeze(row scanner) (dto.FreezeResponse, error) {
	var item dto.FreezeResponse
	if err := row.Scan(
		&item.FreezeID,
		&item.UserID,
		&item.Asset,
		&item.RefType,
		&item.RefID,
		&item.OriginalAmount,
		&item.RemainingAmount,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.FreezeResponse{}, ErrNotFound
		}
		return dto.FreezeResponse{}, err
	}
	return item, nil
}

func scanTrade(row scanner) (dto.TradeResponse, error) {
	var item dto.TradeResponse
	if err := row.Scan(
		&item.TradeID,
		&item.SequenceNo,
		&item.MarketID,
		&item.Outcome,
		&item.CollateralAsset,
		&item.Price,
		&item.Quantity,
		&item.TakerOrderID,
		&item.MakerOrderID,
		&item.TakerUserID,
		&item.MakerUserID,
		&item.TakerSide,
		&item.MakerSide,
		&item.OccurredAt,
	); err != nil {
		return dto.TradeResponse{}, err
	}
	return item, nil
}

func scanSession(row scanner) (dto.SessionResponse, error) {
	var item dto.SessionResponse
	if err := row.Scan(
		&item.SessionID,
		&item.UserID,
		&item.WalletAddress,
		&item.SessionPublicKey,
		&item.Scope,
		&item.ChainID,
		&item.VaultAddress,
		&item.SessionNonce,
		&item.LastOrderNonce,
		&item.Status,
		&item.IssuedAtMillis,
		&item.ExpiresAtMillis,
		&item.RevokedAtMillis,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.SessionResponse{}, ErrNotFound
		}
		return dto.SessionResponse{}, err
	}
	return item, nil
}

func scanUserProfile(row scanner) (dto.UserProfileResponse, error) {
	var item dto.UserProfileResponse
	if err := row.Scan(
		&item.UserID,
		&item.WalletAddress,
		&item.DisplayName,
		&item.AvatarPreset,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.UserProfileResponse{}, ErrNotFound
		}
		return dto.UserProfileResponse{}, err
	}
	return item, nil
}

func scanDeposit(row scanner) (dto.DepositResponse, error) {
	var item dto.DepositResponse
	if err := row.Scan(
		&item.DepositID,
		&item.UserID,
		&item.WalletAddress,
		&item.VaultAddress,
		&item.Asset,
		&item.Amount,
		&item.ChainName,
		&item.NetworkName,
		&item.TxHash,
		&item.LogIndex,
		&item.BlockNumber,
		&item.Status,
		&item.CreditedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return dto.DepositResponse{}, err
	}
	return item, nil
}

func scanWithdrawal(row scanner) (dto.WithdrawalResponse, error) {
	var item dto.WithdrawalResponse
	if err := row.Scan(
		&item.WithdrawalID,
		&item.UserID,
		&item.WalletAddress,
		&item.RecipientAddress,
		&item.VaultAddress,
		&item.Asset,
		&item.Amount,
		&item.ChainName,
		&item.NetworkName,
		&item.TxHash,
		&item.LogIndex,
		&item.BlockNumber,
		&item.Status,
		&item.ClaimStatus,
		&item.ClaimTxHash,
		&item.ClaimSubmittedAt,
		&item.ClaimedAt,
		&item.LastError,
		&item.DebitedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return dto.WithdrawalResponse{}, err
	}
	return item, nil
}

func scanChainTransaction(row scanner) (dto.ChainTransactionResponse, error) {
	var (
		item    dto.ChainTransactionResponse
		payload []byte
	)
	if err := row.Scan(
		&item.ID,
		&item.BizType,
		&item.RefID,
		&item.ChainName,
		&item.NetworkName,
		&item.WalletAddress,
		&item.TxHash,
		&item.Status,
		&payload,
		&item.ErrorMessage,
		&item.AttemptCount,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ChainTransactionResponse{}, ErrNotFound
		}
		return dto.ChainTransactionResponse{}, err
	}
	item.Payload = json.RawMessage(payload)
	return item, nil
}

func (s *SQLStore) attachMarketRuntime(ctx context.Context, markets []dto.MarketResponse) error {
	return s.attachMarketRuntimeAt(ctx, markets, time.Now().Unix())
}

func (s *SQLStore) attachMarketRuntimeAt(ctx context.Context, markets []dto.MarketResponse, nowUnix int64) error {
	if len(markets) == 0 {
		return nil
	}

	categoriesByMarket, err := s.loadMarketCategories(ctx, marketIDs(markets))
	if err != nil {
		return err
	}
	optionsByMarket, err := s.loadMarketOptions(ctx, marketIDs(markets))
	if err != nil {
		return err
	}
	runtimeByMarket, err := s.loadMarketRuntime(ctx, marketIDs(markets))
	if err != nil {
		return err
	}

	for index := range markets {
		applyEffectiveMarketStatus(&markets[index], nowUnix)
		runtime := runtimeByMarket[markets[index].MarketID]
		markets[index].Category = categoriesByMarket[markets[index].MarketID]
		if options, ok := optionsByMarket[markets[index].MarketID]; ok {
			markets[index].Options = options
		} else {
			markets[index].Options = dto.DefaultBinaryMarketOptions()
		}
		markets[index].Runtime = runtime
		markets[index].Metadata = mergeMarketMetadata(markets[index].Metadata, markets[index].Category, markets[index].Status, markets[index].ResolvedOutcome, runtime)
	}
	return nil
}

func (s *SQLStore) loadMarketCategories(ctx context.Context, marketIDs []int64) (map[int64]*dto.MarketCategory, error) {
	items := make(map[int64]*dto.MarketCategory, len(marketIDs))
	if len(marketIDs) == 0 {
		return items, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT m.market_id, c.category_id, c.category_key, c.display_name, c.description, c.sort_order, c.metadata
		FROM markets m
		LEFT JOIN market_categories c ON c.category_id = m.category_id
		WHERE m.market_id = ANY($1)
	`, pq.Array(marketIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			marketID    int64
			item        dto.MarketCategory
			metadata    []byte
			categoryID  sql.NullInt64
			categoryKey sql.NullString
			displayName sql.NullString
			description sql.NullString
			sortOrder   sql.NullInt64
		)
		if err := rows.Scan(&marketID, &categoryID, &categoryKey, &displayName, &description, &sortOrder, &metadata); err != nil {
			return nil, err
		}
		if !categoryID.Valid {
			continue
		}
		item.CategoryID = categoryID.Int64
		item.CategoryKey = categoryKey.String
		item.DisplayName = displayName.String
		item.Description = description.String
		if sortOrder.Valid {
			item.SortOrder = int(sortOrder.Int64)
		}
		item.Metadata = normalizeJSONRaw(metadata)
		items[marketID] = &item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SQLStore) loadMarketOptions(ctx context.Context, marketIDs []int64) (map[int64][]dto.MarketOption, error) {
	items := make(map[int64][]dto.MarketOption, len(marketIDs))
	if len(marketIDs) == 0 {
		return items, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT market_id, option_schema
		FROM market_option_sets
		WHERE market_id = ANY($1)
	`, pq.Array(marketIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			marketID int64
			raw      []byte
		)
		if err := rows.Scan(&marketID, &raw); err != nil {
			return nil, err
		}
		var options []dto.MarketOption
		if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			options = dto.DefaultBinaryMarketOptions()
		} else if err := json.Unmarshal(raw, &options); err != nil {
			return nil, err
		}
		normalized, err := dto.NormalizeMarketOptions(options)
		if err != nil {
			return nil, err
		}
		items[marketID] = normalized
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SQLStore) loadMarketRuntime(ctx context.Context, marketIDs []int64) (map[int64]dto.MarketRuntime, error) {
	runtimeByMarket := make(map[int64]dto.MarketRuntime, len(marketIDs))
	if len(marketIDs) == 0 {
		return runtimeByMarket, nil
	}
	for _, marketID := range marketIDs {
		runtimeByMarket[marketID] = dto.MarketRuntime{}
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT market_id,
		       COUNT(*) AS trade_count,
		       COALESCE(SUM(quantity), 0) AS matched_quantity,
		       COALESCE(SUM(price * quantity), 0) AS matched_notional,
		       COALESCE(MAX(occurred_at), 0) AS last_trade_at
		FROM trades
		WHERE market_id = ANY($1)
		GROUP BY market_id
	`, pq.Array(marketIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			marketID int64
			runtime  dto.MarketRuntime
		)
		if err := rows.Scan(&marketID, &runtime.TradeCount, &runtime.MatchedQuantity, &runtime.MatchedNotional, &runtime.LastTradeAt); err != nil {
			return nil, err
		}
		runtimeByMarket[marketID] = runtime
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	priceRows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT ON (market_id, outcome) market_id, outcome, price
		FROM trades
		WHERE market_id = ANY($1)
		ORDER BY market_id, outcome, sequence_no DESC
	`, pq.Array(marketIDs))
	if err != nil {
		return nil, err
	}
	defer priceRows.Close()

	for priceRows.Next() {
		var (
			marketID int64
			outcome  string
			price    int64
			runtime  dto.MarketRuntime
		)
		if err := priceRows.Scan(&marketID, &outcome, &price); err != nil {
			return nil, err
		}
		runtime = runtimeByMarket[marketID]
		switch normalizeOptional(outcome) {
		case "YES":
			runtime.LastPriceYes = price
		case "NO":
			runtime.LastPriceNo = price
		}
		runtimeByMarket[marketID] = runtime
	}
	if err := priceRows.Err(); err != nil {
		return nil, err
	}

	orderRows, err := s.db.QueryContext(ctx, `
		SELECT market_id, COUNT(*) AS active_order_count
		FROM orders
		WHERE market_id = ANY($1)
		  AND status IN ('NEW', 'PARTIALLY_FILLED')
		GROUP BY market_id
	`, pq.Array(marketIDs))
	if err != nil {
		return nil, err
	}
	defer orderRows.Close()

	for orderRows.Next() {
		var (
			marketID         int64
			activeOrderCount int64
			runtime          dto.MarketRuntime
		)
		if err := orderRows.Scan(&marketID, &activeOrderCount); err != nil {
			return nil, err
		}
		runtime = runtimeByMarket[marketID]
		runtime.ActiveOrderCount = activeOrderCount
		runtimeByMarket[marketID] = runtime
	}
	if err := orderRows.Err(); err != nil {
		return nil, err
	}

	payoutRows, err := s.db.QueryContext(ctx, `
		SELECT market_id,
		       COUNT(*) AS payout_count,
		       COUNT(*) FILTER (WHERE status = 'COMPLETED') AS completed_payout_count
		FROM settlement_payouts
		WHERE market_id = ANY($1)
		GROUP BY market_id
	`, pq.Array(marketIDs))
	if err != nil {
		return nil, err
	}
	defer payoutRows.Close()

	for payoutRows.Next() {
		var (
			marketID             int64
			payoutCount          int64
			completedPayoutCount int64
			runtime              dto.MarketRuntime
		)
		if err := payoutRows.Scan(&marketID, &payoutCount, &completedPayoutCount); err != nil {
			return nil, err
		}
		runtime = runtimeByMarket[marketID]
		runtime.PayoutCount = payoutCount
		runtime.CompletedPayoutCount = completedPayoutCount
		runtimeByMarket[marketID] = runtime
	}
	if err := payoutRows.Err(); err != nil {
		return nil, err
	}

	claimRows, err := s.db.QueryContext(ctx, `
		SELECT sp.market_id,
		       COUNT(*) FILTER (WHERE ct.status = 'PENDING') AS pending_claim_count,
		       COUNT(*) FILTER (WHERE ct.status = 'SUBMITTED') AS submitted_claim_count,
		       COUNT(*) FILTER (WHERE ct.status = 'FAILED') AS failed_claim_count
		FROM chain_transactions ct
		INNER JOIN settlement_payouts sp ON sp.event_id = ct.ref_id
		WHERE ct.biz_type = 'CLAIM'
		  AND sp.market_id = ANY($1)
		GROUP BY sp.market_id
	`, pq.Array(marketIDs))
	if err != nil {
		return nil, err
	}
	defer claimRows.Close()

	for claimRows.Next() {
		var (
			marketID            int64
			pendingClaimCount   int64
			submittedClaimCount int64
			failedClaimCount    int64
			runtime             dto.MarketRuntime
		)
		if err := claimRows.Scan(&marketID, &pendingClaimCount, &submittedClaimCount, &failedClaimCount); err != nil {
			return nil, err
		}
		runtime = runtimeByMarket[marketID]
		runtime.PendingClaimCount = pendingClaimCount
		runtime.SubmittedClaimCount = submittedClaimCount
		runtime.FailedClaimCount = failedClaimCount
		runtimeByMarket[marketID] = runtime
	}
	if err := claimRows.Err(); err != nil {
		return nil, err
	}

	return runtimeByMarket, nil
}

func marketIDs(markets []dto.MarketResponse) []int64 {
	ids := make([]int64, 0, len(markets))
	seen := make(map[int64]struct{}, len(markets))
	for _, market := range markets {
		if _, ok := seen[market.MarketID]; ok {
			continue
		}
		seen[market.MarketID] = struct{}{}
		ids = append(ids, market.MarketID)
	}
	return ids
}

func mergeMarketMetadata(raw json.RawMessage, category *dto.MarketCategory, status, resolvedOutcome string, runtime dto.MarketRuntime) json.RawMessage {
	metadata := make(map[string]any)
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
		if err := json.Unmarshal(trimmed, &metadata); err != nil {
			metadata = make(map[string]any)
		}
	}

	if category != nil {
		metadata["category"] = category.DisplayName
		metadata["categoryKey"] = category.CategoryKey
		metadata["category_key"] = category.CategoryKey
	}

	switch normalizeOptional(status) {
	case "RESOLVED":
		switch normalizeOptional(resolvedOutcome) {
		case "YES":
			metadata["yesOdds"] = 1.0
			metadata["noOdds"] = 0.0
		case "NO":
			metadata["yesOdds"] = 0.0
			metadata["noOdds"] = 1.0
		}
	default:
		switch {
		case runtime.LastPriceYes > 0:
			yesOdds := float64(runtime.LastPriceYes) / 100
			metadata["yesOdds"] = yesOdds
			metadata["noOdds"] = 1 - yesOdds
		case runtime.LastPriceNo > 0:
			noOdds := float64(runtime.LastPriceNo) / 100
			metadata["noOdds"] = noOdds
			metadata["yesOdds"] = 1 - noOdds
		}
	}

	metadata["volume"] = runtime.MatchedNotional
	metadata["matchedQuantity"] = runtime.MatchedQuantity
	metadata["matchedNotional"] = runtime.MatchedNotional
	metadata["tradeCount"] = runtime.TradeCount
	metadata["lastTradeAt"] = runtime.LastTradeAt

	encoded, err := json.Marshal(metadata)
	if err != nil {
		return []byte(`{}`)
	}
	return encoded
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func ensureUserProfileTx(ctx context.Context, tx *sql.Tx, userID int64, walletAddress string) error {
	if userID <= 0 || strings.TrimSpace(walletAddress) == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO user_profiles (
			user_id, wallet_address, display_name, avatar_preset, created_at, updated_at
		)
		VALUES ($1, $2, '', $3, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (user_id) DO UPDATE
		SET wallet_address = EXCLUDED.wallet_address,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
	`, userID, normalizeWalletAddress(walletAddress), dto.DefaultAvatarPreset(userID, walletAddress))
	return err
}

func resolveOrCreateUserIDTx(ctx context.Context, tx *sql.Tx, walletAddress string) (int64, error) {
	walletAddress = normalizeWalletAddress(walletAddress)
	if walletAddress == "" {
		return 0, ErrNotFound
	}

	var userID int64
	err := tx.QueryRowContext(ctx, `
		SELECT user_id
		FROM user_profiles
		WHERE wallet_address = $1
		LIMIT 1
	`, walletAddress).Scan(&userID)
	if err == nil {
		return userID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	if _, err := tx.ExecContext(ctx, `LOCK TABLE user_profiles IN EXCLUSIVE MODE`); err != nil {
		return 0, err
	}

	err = tx.QueryRowContext(ctx, `
		SELECT user_id
		FROM user_profiles
		WHERE wallet_address = $1
		LIMIT 1
	`, walletAddress).Scan(&userID)
	if err == nil {
		return userID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(user_id), 1000) + 1
		FROM user_profiles
	`).Scan(&userID)
	if err != nil {
		return 0, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO user_profiles (
			user_id, wallet_address, display_name, avatar_preset, created_at, updated_at
		)
		VALUES ($1, $2, '', $3, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
	`, userID, walletAddress, dto.DefaultAvatarPreset(userID, walletAddress)); err != nil {
		return 0, err
	}
	return userID, nil
}

func normalizeOptional(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeWalletAddress(walletAddress string) string {
	return strings.ToLower(strings.TrimSpace(walletAddress))
}

func normalizeVaultAddress(vaultAddress string) string {
	return strings.ToLower(strings.TrimSpace(vaultAddress))
}

func normalizeChallengeValue(challenge string) string {
	normalized := strings.ToLower(strings.TrimSpace(challenge))
	return strings.TrimPrefix(normalized, "0x")
}

func formatChallengeValue(challenge string) string {
	normalized := normalizeChallengeValue(challenge)
	if normalized == "" {
		return ""
	}
	return "0x" + normalized
}

func normalizePublicKey(publicKey string) string {
	return strings.ToLower(strings.TrimSpace(publicKey))
}

func normalizeAsset(asset string) string {
	if strings.TrimSpace(asset) == "" {
		return "USDT"
	}
	return strings.ToUpper(strings.TrimSpace(asset))
}

func normalizeMarketStatus(status string) string {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	if normalized == "" {
		return "OPEN"
	}
	switch normalized {
	case "DRAFT", "OPEN", "PAUSED", "CLOSED":
		return normalized
	default:
		return "OPEN"
	}
}

func normalizeSessionScope(scope string) string {
	normalized := strings.ToUpper(strings.TrimSpace(scope))
	if normalized == "" {
		return "TRADE"
	}
	return normalized
}

func getSessionTx(ctx context.Context, tx *sql.Tx, sessionID string) (dto.SessionResponse, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT session_id, user_id, wallet_address, session_public_key, scope, chain_id, vault_address,
		       session_nonce, last_order_nonce, status, issued_at, expires_at, revoked_at, created_at, updated_at
		FROM wallet_sessions
		WHERE session_id = $1
	`, strings.TrimSpace(sessionID))
	return scanSession(row)
}

func getActiveTradingKeyTx(ctx context.Context, tx *sql.Tx, sessionID string) (dto.SessionResponse, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT session_id, user_id, wallet_address, session_public_key, scope, chain_id, vault_address,
		       session_nonce, last_order_nonce, status, issued_at, expires_at, revoked_at, created_at, updated_at
		FROM wallet_sessions
		WHERE session_id = $1
		  AND status = 'ACTIVE'
		LIMIT 1
	`, strings.TrimSpace(sessionID))
	return scanSession(row)
}

func getTradingKeyChallengeTx(ctx context.Context, tx *sql.Tx, challengeID string) (tradingKeyChallengeRecord, error) {
	var item tradingKeyChallengeRecord
	err := tx.QueryRowContext(ctx, `
		SELECT challenge_id, wallet_address, chain_id, vault_address, challenge, expires_at, consumed_at
		FROM trading_key_challenges
		WHERE challenge_id = $1
		LIMIT 1
	`, strings.TrimSpace(challengeID)).Scan(
		&item.ChallengeID,
		&item.WalletAddress,
		&item.ChainID,
		&item.VaultAddress,
		&item.Challenge,
		&item.ExpiresAt,
		&item.ConsumedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tradingKeyChallengeRecord{}, ErrNotFound
		}
		return tradingKeyChallengeRecord{}, err
	}
	return item, nil
}

func isIdempotentTradingKeyRetry(existing dto.SessionResponse, req dto.RegisterTradingKeyRequest, normalizedChallenge string) bool {
	if existing.SessionID == "" || strings.ToUpper(existing.Status) != "ACTIVE" {
		return false
	}
	if existing.SessionID != strings.TrimSpace(req.SessionID) {
		return false
	}
	if existing.WalletAddress != normalizeWalletAddress(req.WalletAddress) {
		return false
	}
	if existing.ChainID != req.ChainID {
		return false
	}
	if existing.VaultAddress != normalizeVaultAddress(req.VaultAddress) {
		return false
	}
	if !strings.EqualFold(existing.SessionPublicKey, normalizePublicKey(req.TradingPublicKey)) {
		return false
	}
	return strings.EqualFold(existing.SessionNonce, normalizedChallenge)
}

func (s *SQLStore) getChainTransactionByRef(ctx context.Context, bizType, refID string) (dto.ChainTransactionResponse, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, biz_type, ref_id, chain_name, network_name, wallet_address, tx_hash,
		       status, payload, error_message, attempt_count, created_at, updated_at
		FROM chain_transactions
		WHERE biz_type = $1
		  AND ref_id = $2
		ORDER BY id DESC
		LIMIT 1
	`, strings.ToUpper(strings.TrimSpace(bizType)), strings.TrimSpace(refID))
	return scanChainTransaction(row)
}

func metadataOrDefault(raw json.RawMessage) []byte {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return []byte(`{}`)
	}
	return raw
}

func normalizeJSONRaw(raw []byte) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	return json.RawMessage(trimmed)
}

func lookupMarketCategoryTx(ctx context.Context, tx *sql.Tx, categoryKey string) (dto.MarketCategory, error) {
	var (
		item     dto.MarketCategory
		metadata []byte
	)
	row := tx.QueryRowContext(ctx, `
		SELECT category_id, category_key, display_name, description, sort_order, metadata
		FROM market_categories
		WHERE category_key = $1
		  AND status = 'ACTIVE'
		LIMIT 1
	`, categoryKey)
	if err := row.Scan(&item.CategoryID, &item.CategoryKey, &item.DisplayName, &item.Description, &item.SortOrder, &metadata); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.MarketCategory{}, ErrInvalidMarketCategory
		}
		return dto.MarketCategory{}, err
	}
	item.Metadata = normalizeJSONRaw(metadata)
	return item, nil
}
