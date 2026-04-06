package rollup

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"funnyoption/internal/shared/assets"
)

type balanceLeaf struct {
	AccountID           int64  `json:"account_id"`
	AssetID             string `json:"asset_id"`
	FreeBalance         int64  `json:"free_balance"`
	LockedBalance       int64  `json:"locked_balance"`
	LastAppliedSequence int64  `json:"last_applied_sequence"`
}

type nonceLeaf struct {
	AccountID           int64  `json:"account_id"`
	AuthKeyID           string `json:"auth_key_id"`
	NextNonce           uint64 `json:"next_nonce"`
	KeyStatus           string `json:"key_status"`
	Scope               string `json:"scope"`
	LastAppliedSequence int64  `json:"last_applied_sequence"`
}

type orderLeaf struct {
	OrderID             string `json:"order_id"`
	AccountID           int64  `json:"account_id"`
	MarketID            int64  `json:"market_id"`
	Outcome             string `json:"outcome"`
	Side                string `json:"side"`
	Price               int64  `json:"price"`
	OriginalQuantity    int64  `json:"original_quantity"`
	FilledQuantity      int64  `json:"filled_quantity"`
	RemainingQuantity   int64  `json:"remaining_quantity"`
	ReserveAsset        string `json:"reserve_asset"`
	ReservedCollateral  int64  `json:"reserved_collateral"`
	Status              string `json:"status"`
	LastAppliedSequence int64  `json:"last_applied_sequence"`
}

type positionLeaf struct {
	AccountID           int64  `json:"account_id"`
	MarketID            int64  `json:"market_id"`
	Outcome             string `json:"outcome"`
	Quantity            int64  `json:"quantity"`
	CostBasis           int64  `json:"cost_basis"`
	RealizedPnL         int64  `json:"realized_pnl"`
	FundingSnapshot     int64  `json:"funding_snapshot"`
	SettlementStatus    string `json:"settlement_status"`
	LastAppliedSequence int64  `json:"last_applied_sequence"`
}

type marketFundingLeaf struct {
	MarketID               int64  `json:"market_id"`
	CumulativeFundingIndex int64  `json:"cumulative_funding_index"`
	LastOracleRef          string `json:"last_oracle_ref"`
	MarketSettlementState  string `json:"market_settlement_state"`
	ResolvedOutcome        string `json:"resolved_outcome"`
	LastAppliedSequence    int64  `json:"last_applied_sequence"`
}

type withdrawalLeaf struct {
	WithdrawalID    string `json:"withdrawal_id"`
	AccountID       int64  `json:"account_id"`
	AssetID         string `json:"asset_id"`
	Amount          int64  `json:"amount"`
	Recipient       string `json:"recipient"`
	Lane            string `json:"lane"`
	Status          string `json:"status"`
	Beneficiary     string `json:"beneficiary"`
	RequestSequence int64  `json:"request_sequence"`
	ClaimNullifier  string `json:"claim_nullifier"`
}

type shadowState struct {
	balances      map[string]balanceLeaf
	nonces        map[string]nonceLeaf
	openOrders    map[string]orderLeaf
	positions     map[string]positionLeaf
	marketFunding map[string]marketFundingLeaf
	withdrawals   map[string]withdrawalLeaf
}

func ZeroBalancesRoot() string {
	return hashStrings("shadow", "balances", "empty")
}

func ZeroNonceRoot() string {
	return hashStrings("shadow", "orders", "nonce", "empty")
}

func ZeroOpenOrdersRoot() string {
	return hashStrings("shadow", "orders", "open", "empty")
}

func ZeroMarketFundingRoot() string {
	return hashStrings("shadow", "positions", "funding", "empty")
}

func ZeroInsuranceRoot() string {
	return hashStrings("shadow", "positions", "insurance", "empty")
}

func ZeroWithdrawalsRoot() string {
	return hashStrings("shadow", "withdrawals", "empty")
}

func ZeroConservationHash() string {
	return hashStrings("shadow", "conservation", "empty")
}

func ZeroStateRoot() string {
	return hashStrings("shadow", "state", "v1", ZeroBalancesRoot(), hashStrings("shadow", "orders", ZeroNonceRoot(), ZeroOpenOrdersRoot()), hashStrings("shadow", "positions_funding", hashStrings("shadow", "positions", "leafs", "empty"), ZeroMarketFundingRoot(), ZeroInsuranceRoot()), ZeroWithdrawalsRoot())
}

func EncodeBatchInput(entries []JournalEntry) (string, string, error) {
	if entries == nil {
		entries = []JournalEntry{}
	}
	input := BatchInput{
		EncodingVersion: BatchEncodingVersion,
		Entries:         entries,
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", "", err
	}
	inputData := string(encoded)
	return inputData, hashStrings("shadow", "batch_input", inputData), nil
}

func DecodeBatchInput(input string) (BatchInput, error) {
	var batchInput BatchInput
	if err := json.Unmarshal([]byte(input), &batchInput); err != nil {
		return BatchInput{}, err
	}
	if strings.TrimSpace(batchInput.EncodingVersion) != BatchEncodingVersion {
		return BatchInput{}, fmt.Errorf("unsupported rollup batch encoding version: %s", batchInput.EncodingVersion)
	}
	if batchInput.Entries == nil {
		batchInput.Entries = []JournalEntry{}
	}
	return batchInput, nil
}

func ReplayStoredBatches(batches []StoredBatch) (RootSet, error) {
	state := shadowState{
		balances:      make(map[string]balanceLeaf),
		nonces:        make(map[string]nonceLeaf),
		openOrders:    make(map[string]orderLeaf),
		positions:     make(map[string]positionLeaf),
		marketFunding: make(map[string]marketFundingLeaf),
		withdrawals:   make(map[string]withdrawalLeaf),
	}
	roots := state.roots()
	for _, batch := range batches {
		if batch.InputHash != "" && hashStrings("shadow", "batch_input", batch.InputData) != batch.InputHash {
			return RootSet{}, fmt.Errorf("rollup batch input hash mismatch for batch %d", batch.BatchID)
		}
		expectedPrev := strings.TrimSpace(batch.PrevStateRoot)
		if expectedPrev == "" {
			expectedPrev = ZeroStateRoot()
		}
		if roots.StateRoot != expectedPrev {
			return RootSet{}, fmt.Errorf("rollup batch prev_state_root mismatch: have %s want %s", roots.StateRoot, expectedPrev)
		}
		input, err := DecodeBatchInput(batch.InputData)
		if err != nil {
			return RootSet{}, err
		}
		for _, entry := range input.Entries {
			if err := state.apply(entry); err != nil {
				return RootSet{}, err
			}
		}
		roots = state.roots()
	}
	return roots, nil
}

func (s *shadowState) apply(entry JournalEntry) error {
	switch entry.EntryType {
	case EntryTypeNonceAdvanced:
		var payload NonceAdvancedPayload
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			return err
		}
		return s.applyNonceAdvanced(entry.Sequence, payload)
	case EntryTypeTradingKeyAuthorized:
		var payload TradingKeyAuthorizedPayload
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			return err
		}
		return validateTradingKeyAuthorizedPayload(payload)
	case EntryTypeOrderAccepted:
		var payload OrderAcceptedPayload
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			return err
		}
		return s.applyOrderAccepted(entry.Sequence, payload)
	case EntryTypeOrderCancelled:
		var payload OrderCancelledPayload
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			return err
		}
		return s.applyOrderCancelled(entry.Sequence, payload)
	case EntryTypeTradeMatched:
		var payload TradeMatchedPayload
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			return err
		}
		return s.applyTradeMatched(entry.Sequence, payload)
	case EntryTypeDepositCredited:
		var payload DepositCreditedPayload
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			return err
		}
		s.adjustBalance(payload.AccountID, payload.Asset, payload.Amount, 0, entry.Sequence)
		return nil
	case EntryTypeWithdrawalRequested:
		var payload WithdrawalRequestedPayload
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			return err
		}
		s.adjustBalance(payload.AccountID, payload.Asset, -payload.Amount, 0, entry.Sequence)
		s.withdrawals[payload.WithdrawalID] = withdrawalLeaf{
			WithdrawalID:    payload.WithdrawalID,
			AccountID:       payload.AccountID,
			AssetID:         normalizeAsset(payload.Asset),
			Amount:          payload.Amount,
			Recipient:       normalizeText(payload.RecipientAddress),
			Lane:            normalizeText(payload.Lane),
			Status:          "REQUESTED",
			Beneficiary:     "USER",
			RequestSequence: entry.Sequence,
			ClaimNullifier:  "",
		}
		return nil
	case EntryTypeMarketResolved:
		var payload MarketResolvedPayload
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			return err
		}
		return s.applyMarketResolved(entry.Sequence, payload)
	case EntryTypeSettlementPayout:
		var payload SettlementPayoutPayload
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			return err
		}
		return s.applySettlementPayout(entry.Sequence, payload)
	default:
		return fmt.Errorf("unsupported rollup entry type: %s", entry.EntryType)
	}
}

func (s *shadowState) applyNonceAdvanced(sequence int64, payload NonceAdvancedPayload) error {
	if payload.AccountID <= 0 {
		return fmt.Errorf("account_id is required for nonce advance")
	}
	authKeyID := normalizeText(payload.AuthKeyID)
	if authKeyID == "" {
		return fmt.Errorf("auth_key_id is required for nonce advance")
	}
	if payload.AcceptedNonce == 0 {
		return fmt.Errorf("accepted_nonce must be positive")
	}
	if payload.NextNonce == 0 || payload.NextNonce != payload.AcceptedNonce+1 {
		return fmt.Errorf("next_nonce must equal accepted_nonce + 1")
	}

	key := nonceKey(payload.AccountID, authKeyID)
	current, ok := s.nonces[key]
	if ok && payload.AcceptedNonce < current.NextNonce {
		return fmt.Errorf("nonce advance regressed for account %d auth_key %s", payload.AccountID, authKeyID)
	}

	s.nonces[key] = nonceLeaf{
		AccountID:           payload.AccountID,
		AuthKeyID:           authKeyID,
		NextNonce:           payload.NextNonce,
		KeyStatus:           normalizeStatus(payload.KeyStatus),
		Scope:               normalizeScope(payload.Scope),
		LastAppliedSequence: sequence,
	}
	return nil
}

func validateTradingKeyAuthorizedPayload(payload TradingKeyAuthorizedPayload) error {
	witness := payload.AuthorizationWitness
	if strings.TrimSpace(witness.TradingKeyID) == "" {
		return fmt.Errorf("trading_key_id is required for trading key auth witness")
	}
	if strings.TrimSpace(witness.AuthorizationRef) == "" {
		return fmt.Errorf("authorization_ref is required for trading key auth witness")
	}
	if witness.AccountID <= 0 {
		return fmt.Errorf("account_id is required for trading key auth witness")
	}
	if strings.TrimSpace(witness.WalletAddress) == "" {
		return fmt.Errorf("wallet_address is required for trading key auth witness")
	}
	if witness.ChainID <= 0 {
		return fmt.Errorf("chain_id is required for trading key auth witness")
	}
	if strings.TrimSpace(witness.VaultAddress) == "" {
		return fmt.Errorf("vault_address is required for trading key auth witness")
	}
	if strings.TrimSpace(witness.TradingPublicKey) == "" {
		return fmt.Errorf("trading_public_key is required for trading key auth witness")
	}
	if strings.TrimSpace(witness.WalletSignature) == "" {
		return fmt.Errorf("wallet_signature is required for trading key auth witness")
	}
	return nil
}

func (s *shadowState) applyOrderAccepted(sequence int64, payload OrderAcceptedPayload) error {
	reserveAsset := reserveAsset(payload)
	s.adjustBalance(payload.AccountID, reserveAsset, -payload.ReserveAmount, payload.ReserveAmount, sequence)
	s.openOrders[payload.OrderID] = orderLeaf{
		OrderID:             payload.OrderID,
		AccountID:           payload.AccountID,
		MarketID:            payload.MarketID,
		Outcome:             normalizeOutcome(payload.Outcome),
		Side:                normalizeSide(payload.Side),
		Price:               payload.Price,
		OriginalQuantity:    payload.Quantity,
		FilledQuantity:      0,
		RemainingQuantity:   payload.Quantity,
		ReserveAsset:        reserveAsset,
		ReservedCollateral:  payload.ReserveAmount,
		Status:              "OPEN",
		LastAppliedSequence: sequence,
	}
	return nil
}

func (s *shadowState) applyOrderCancelled(sequence int64, payload OrderCancelledPayload) error {
	order, ok := s.openOrders[payload.OrderID]
	if !ok {
		remaining := payload.RemainingQuantity
		if remaining < 0 {
			remaining = 0
		}
		order = orderLeaf{
			OrderID:            payload.OrderID,
			AccountID:          payload.AccountID,
			MarketID:           payload.MarketID,
			Outcome:            normalizeOutcome(payload.Outcome),
			Side:               normalizeSide(payload.Side),
			Price:              payload.Price,
			RemainingQuantity:  remaining,
			ReserveAsset:       reserveAssetFromCancellation(payload),
			ReservedCollateral: expectedReserveAmount(payload.Side, payload.Price, remaining),
		}
	}
	if order.ReservedCollateral > 0 {
		s.adjustBalance(order.AccountID, order.ReserveAsset, order.ReservedCollateral, -order.ReservedCollateral, sequence)
	}
	delete(s.openOrders, payload.OrderID)
	return nil
}

func (s *shadowState) applyTradeMatched(sequence int64, payload TradeMatchedPayload) error {
	notional, err := multiply(payload.Price, payload.Quantity)
	if err != nil {
		return err
	}

	if err := s.fillOrder(sequence, payload.TakerOrderID, payload.Quantity, payload.Price); err != nil {
		return err
	}
	if err := s.fillOrder(sequence, payload.MakerOrderID, payload.Quantity, payload.Price); err != nil {
		return err
	}

	buyerAccountID, sellerAccountID := buyerSellerAccounts(payload)
	positionAsset := assets.PositionAsset(payload.MarketID, payload.Outcome)
	if buyerAccountID > 0 {
		s.adjustBalance(buyerAccountID, positionAsset, payload.Quantity, 0, sequence)
		s.adjustPosition(buyerAccountID, payload.MarketID, payload.Outcome, payload.Quantity, sequence)
	}
	if sellerAccountID > 0 {
		s.adjustBalance(sellerAccountID, payload.CollateralAsset, notional, 0, sequence)
		s.adjustPosition(sellerAccountID, payload.MarketID, payload.Outcome, -payload.Quantity, sequence)
	}
	return nil
}

func (s *shadowState) fillOrder(sequence int64, orderID string, fillQuantity, executionPrice int64) error {
	order, ok := s.openOrders[orderID]
	if !ok {
		return fmt.Errorf("rollup trade references unknown order: %s", orderID)
	}
	if fillQuantity <= 0 {
		return fmt.Errorf("fill quantity must be positive")
	}
	if order.RemainingQuantity < fillQuantity {
		return fmt.Errorf("fill quantity exceeds remaining quantity for order %s", orderID)
	}

	releasedFree := int64(0)
	switch order.Side {
	case "BUY":
		notional, err := multiply(executionPrice, fillQuantity)
		if err != nil {
			return err
		}
		s.adjustBalance(order.AccountID, order.ReserveAsset, 0, -notional, sequence)
		order.ReservedCollateral -= notional
		expectedReserve := expectedReserveAmount(order.Side, order.Price, order.RemainingQuantity-fillQuantity)
		releasedFree = order.ReservedCollateral - expectedReserve
		if releasedFree < 0 {
			return fmt.Errorf("negative reserve release for buy order %s", order.OrderID)
		}
		if releasedFree > 0 {
			s.adjustBalance(order.AccountID, order.ReserveAsset, releasedFree, -releasedFree, sequence)
		}
		order.ReservedCollateral = expectedReserve
	case "SELL":
		s.adjustBalance(order.AccountID, order.ReserveAsset, 0, -fillQuantity, sequence)
		order.ReservedCollateral -= fillQuantity
		expectedReserve := expectedReserveAmount(order.Side, order.Price, order.RemainingQuantity-fillQuantity)
		releasedFree = order.ReservedCollateral - expectedReserve
		if releasedFree < 0 {
			return fmt.Errorf("negative reserve release for sell order %s", order.OrderID)
		}
		if releasedFree > 0 {
			s.adjustBalance(order.AccountID, order.ReserveAsset, releasedFree, -releasedFree, sequence)
		}
		order.ReservedCollateral = expectedReserve
	default:
		return fmt.Errorf("unsupported order side %s", order.Side)
	}

	order.FilledQuantity += fillQuantity
	order.RemainingQuantity -= fillQuantity
	order.LastAppliedSequence = sequence
	if order.RemainingQuantity == 0 {
		delete(s.openOrders, orderID)
		return nil
	}
	order.Status = "OPEN"
	s.openOrders[orderID] = order
	return nil
}

func (s *shadowState) adjustBalance(accountID int64, asset string, freeDelta, lockedDelta int64, sequence int64) {
	if accountID <= 0 {
		return
	}
	key := balanceKey(accountID, asset)
	leaf := s.balances[key]
	leaf.AccountID = accountID
	leaf.AssetID = normalizeAsset(asset)
	leaf.FreeBalance += freeDelta
	leaf.LockedBalance += lockedDelta
	leaf.LastAppliedSequence = sequence
	if leaf.FreeBalance == 0 && leaf.LockedBalance == 0 {
		delete(s.balances, key)
		return
	}
	s.balances[key] = leaf
}

func (s *shadowState) adjustPosition(accountID, marketID int64, outcome string, delta, sequence int64) {
	if accountID <= 0 || delta == 0 {
		return
	}
	key := positionKey(accountID, marketID, outcome)
	leaf := s.positions[key]
	leaf.AccountID = accountID
	leaf.MarketID = marketID
	leaf.Outcome = normalizeOutcome(outcome)
	leaf.Quantity += delta
	leaf.CostBasis = 0
	leaf.RealizedPnL = 0
	leaf.FundingSnapshot = 0
	leaf.SettlementStatus = "OPEN"
	leaf.LastAppliedSequence = sequence
	if leaf.Quantity == 0 {
		delete(s.positions, key)
		return
	}
	s.positions[key] = leaf
}

func (s *shadowState) applyMarketResolved(sequence int64, payload MarketResolvedPayload) error {
	if payload.MarketID <= 0 {
		return fmt.Errorf("market_id is required for market resolution")
	}
	outcome := normalizeOutcome(payload.ResolvedOutcome)
	if outcome == "" {
		return fmt.Errorf("resolved_outcome is required for market resolution")
	}
	key := marketFundingKey(payload.MarketID)
	s.marketFunding[key] = marketFundingLeaf{
		MarketID:               payload.MarketID,
		CumulativeFundingIndex: 0,
		LastOracleRef:          resolutionRef(payload),
		MarketSettlementState:  "RESOLVED",
		ResolvedOutcome:        outcome,
		LastAppliedSequence:    sequence,
	}
	return nil
}

func (s *shadowState) applySettlementPayout(sequence int64, payload SettlementPayoutPayload) error {
	if payload.AccountID <= 0 {
		return fmt.Errorf("account_id is required for settlement payout")
	}
	if payload.MarketID <= 0 {
		return fmt.Errorf("market_id is required for settlement payout")
	}
	if payload.SettledQuantity <= 0 {
		return fmt.Errorf("settled_quantity must be positive")
	}
	if payload.PayoutAmount < 0 {
		return fmt.Errorf("payout_amount cannot be negative")
	}

	positionAsset := normalizeAsset(payload.PositionAsset)
	if positionAsset == "" {
		positionAsset = assets.PositionAsset(payload.MarketID, payload.WinningOutcome)
	}
	if balance := s.balanceLeaf(payload.AccountID, positionAsset); balance.FreeBalance < payload.SettledQuantity {
		return fmt.Errorf("settlement payout exceeds shadow position balance for account %d market %d", payload.AccountID, payload.MarketID)
	}
	s.adjustBalance(payload.AccountID, positionAsset, -payload.SettledQuantity, 0, sequence)

	if err := s.settlePosition(sequence, payload.AccountID, payload.MarketID, payload.WinningOutcome, payload.SettledQuantity); err != nil {
		return err
	}
	if payload.PayoutAmount > 0 {
		payoutAsset := normalizeAsset(payload.PayoutAsset)
		if payoutAsset == "" {
			payoutAsset = assets.DefaultCollateralAsset
		}
		s.adjustBalance(payload.AccountID, payoutAsset, payload.PayoutAmount, 0, sequence)
	}
	return nil
}

func (s *shadowState) settlePosition(sequence, accountID, marketID int64, outcome string, settledQuantity int64) error {
	key := positionKey(accountID, marketID, outcome)
	leaf, ok := s.positions[key]
	if !ok {
		return fmt.Errorf("settlement payout references unknown position %s", key)
	}
	if leaf.Quantity < settledQuantity {
		return fmt.Errorf("settlement quantity exceeds position quantity for %s", key)
	}
	leaf.Quantity -= settledQuantity
	leaf.LastAppliedSequence = sequence
	switch {
	case leaf.Quantity == 0:
		delete(s.positions, key)
	case leaf.Quantity > 0:
		leaf.SettlementStatus = "PARTIALLY_SETTLED"
		s.positions[key] = leaf
	default:
		return fmt.Errorf("settlement produced negative position quantity for %s", key)
	}
	return nil
}

func (s *shadowState) balanceLeaf(accountID int64, asset string) balanceLeaf {
	return s.balances[balanceKey(accountID, asset)]
}

func (s *shadowState) roots() RootSet {
	balancesRoot := leafRoot("balances", s.balances, ZeroBalancesRoot())
	nonceRoot := nonceLeafRoot(s.nonces)
	openOrdersRoot := leafRoot("open_orders", s.openOrders, ZeroOpenOrdersRoot())
	positionRoot := leafRoot("positions", s.positions, hashStrings("shadow", "positions", "leafs", "empty"))
	marketFundingRoot := leafRoot("market_funding", s.marketFunding, ZeroMarketFundingRoot())
	withdrawalsRoot := leafRoot("withdrawals", s.withdrawals, ZeroWithdrawalsRoot())
	ordersRoot := hashStrings("shadow", "orders", nonceRoot, openOrdersRoot)
	positionsFundingRoot := hashStrings("shadow", "positions_funding", positionRoot, marketFundingRoot, ZeroInsuranceRoot())
	return RootSet{
		BalancesRoot:         balancesRoot,
		OrdersRoot:           ordersRoot,
		PositionsFundingRoot: positionsFundingRoot,
		WithdrawalsRoot:      withdrawalsRoot,
		StateRoot:            hashStrings("shadow", "state", "v1", balancesRoot, ordersRoot, positionsFundingRoot, withdrawalsRoot),
	}
}

func nonceLeafRoot(leaves map[string]nonceLeaf) string {
	if len(leaves) == 0 {
		return ZeroNonceRoot()
	}
	keys := make([]string, 0, len(leaves))
	for key := range leaves {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hashes := make([]string, 0, len(keys))
	for _, key := range keys {
		sum, err := hashStruct("orders_nonce", key, leaves[key])
		if err != nil {
			return hashStrings("shadow", "orders", "nonce", "encode_error", key)
		}
		hashes = append(hashes, sum)
	}
	return hashStrings(append([]string{"shadow", "orders", "nonce"}, hashes...)...)
}

func leafRoot[T any](namespace string, leaves map[string]T, zero string) string {
	if len(leaves) == 0 {
		return zero
	}
	keys := make([]string, 0, len(leaves))
	for key := range leaves {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hashes := make([]string, 0, len(keys))
	for _, key := range keys {
		sum, err := hashStruct(namespace, key, leaves[key])
		if err != nil {
			return hashStrings(namespace, "encode_error", key)
		}
		hashes = append(hashes, sum)
	}
	return hashStrings(append([]string{"shadow", namespace}, hashes...)...)
}

func balanceKey(accountID int64, asset string) string {
	return fmt.Sprintf("%d:%s", accountID, normalizeAsset(asset))
}

func nonceKey(accountID int64, authKeyID string) string {
	return fmt.Sprintf("%d:%s", accountID, normalizeText(authKeyID))
}

func positionKey(accountID, marketID int64, outcome string) string {
	return fmt.Sprintf("%d:%d:%s", accountID, marketID, normalizeOutcome(outcome))
}

func marketFundingKey(marketID int64) string {
	return fmt.Sprintf("%d", marketID)
}

func reserveAsset(payload OrderAcceptedPayload) string {
	if asset := normalizeAsset(payload.ReserveAsset); asset != "" {
		return asset
	}
	return normalizeAsset(payload.CollateralAsset)
}

func reserveAssetFromCancellation(payload OrderCancelledPayload) string {
	if asset := normalizeAsset(payload.ReserveAsset); asset != "" {
		return asset
	}
	if normalizeSide(payload.Side) == "SELL" {
		return assets.PositionAsset(payload.MarketID, payload.Outcome)
	}
	return assets.DefaultCollateralAsset
}

func expectedReserveAmount(side string, price, quantity int64) int64 {
	if quantity <= 0 {
		return 0
	}
	if normalizeSide(side) == "SELL" {
		return quantity
	}
	value, err := multiply(price, quantity)
	if err != nil {
		return 0
	}
	return value
}

func buyerSellerAccounts(payload TradeMatchedPayload) (buyerAccountID, sellerAccountID int64) {
	if normalizeSide(payload.TakerSide) == "BUY" {
		return payload.TakerAccountID, payload.MakerAccountID
	}
	return payload.MakerAccountID, payload.TakerAccountID
}

func normalizeAsset(asset string) string {
	return strings.ToUpper(strings.TrimSpace(asset))
}

func normalizeOutcome(outcome string) string {
	return strings.ToUpper(strings.TrimSpace(outcome))
}

func normalizeSide(side string) string {
	return strings.ToUpper(strings.TrimSpace(side))
}

func normalizeText(value string) string {
	return strings.TrimSpace(value)
}

func normalizeScope(scope string) string {
	normalized := strings.ToUpper(strings.TrimSpace(scope))
	if normalized == "" {
		return "TRADE"
	}
	return normalized
}

func normalizeStatus(status string) string {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	if normalized == "" {
		return "ACTIVE"
	}
	return normalized
}

func resolutionRef(payload MarketResolvedPayload) string {
	ref := strings.TrimSpace(payload.ResolverRef)
	switch {
	case ref != "" && strings.TrimSpace(payload.ResolverType) != "":
		return strings.ToUpper(strings.TrimSpace(payload.ResolverType)) + ":" + ref
	case ref != "":
		return ref
	default:
		return strings.ToUpper(strings.TrimSpace(payload.ResolverType))
	}
}

func multiply(left, right int64) (int64, error) {
	if left < 0 || right < 0 {
		return 0, fmt.Errorf("negative multiply is not supported")
	}
	if left == 0 || right == 0 {
		return 0, nil
	}
	if left > (1<<63-1)/right {
		return 0, fmt.Errorf("multiply overflow")
	}
	return left * right, nil
}
