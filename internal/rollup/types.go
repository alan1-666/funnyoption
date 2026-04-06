package rollup

import (
	"encoding/json"

	sharedauth "funnyoption/internal/shared/auth"
)

const (
	EntryTypeTradingKeyAuthorized = "TRADING_KEY_AUTHORIZED"
	EntryTypeNonceAdvanced        = "NONCE_ADVANCED"
	EntryTypeOrderAccepted        = "ORDER_ACCEPTED"
	EntryTypeOrderCancelled       = "ORDER_CANCELLED"
	EntryTypeTradeMatched         = "TRADE_MATCHED"
	EntryTypeDepositCredited      = "DEPOSIT_CREDITED"
	EntryTypeWithdrawalRequested  = "WITHDRAWAL_REQUESTED"
	EntryTypeMarketResolved       = "MARKET_RESOLVED"
	EntryTypeSettlementPayout     = "SETTLEMENT_PAYOUT"

	SourceTypeAPIAuth          = "API_AUTH"
	SourceTypeMatchingOrder    = "MATCHING_ORDER"
	SourceTypeMatchingTrade    = "MATCHING_TRADE"
	SourceTypeChainDeposit     = "CHAIN_DEPOSIT"
	SourceTypeChainWithdraw    = "CHAIN_WITHDRAWAL"
	SourceTypeSettlementMarket = "SETTLEMENT_MARKET"
	SourceTypeSettlementOrder  = "SETTLEMENT_ORDER"
	SourceTypeSettlementPayout = "SETTLEMENT_PAYOUT"

	BatchEncodingVersion      = "shadow-batch-v1"
	SubmissionEncodingVersion = "shadow-submit-v1"

	VerifierAuthJoinSatisfied  = "JOINED"
	VerifierAuthJoinMissing    = "MISSING_TRADING_KEY_AUTHORIZED"
	VerifierAuthJoinIneligible = "NON_VERIFIER_ELIGIBLE"

	SubmissionStatusReady           = "READY"
	SubmissionStatusBlockedAuth     = "BLOCKED_AUTH"
	SubmissionStatusRecordSubmitted = "RECORD_SUBMITTED"
	SubmissionStatusAcceptSubmitted = "ACCEPT_SUBMITTED"
	SubmissionStatusAccepted        = "ACCEPTED"
	SubmissionStatusFailed          = "FAILED"

	FunnyRollupCoreContractName              = "FunnyRollupCore"
	FunnyRollupCoreContractPath              = "contracts/src/FunnyRollupCore.sol"
	FunnyRollupCoreAcceptVerifiedBatchMethod = "acceptVerifiedBatch"

	FunnyRollupBatchVerifierContractName          = "IFunnyRollupBatchVerifier"
	FunnyRollupBatchVerifierImplementationName    = "FunnyRollupVerifier"
	FunnyRollupBatchVerifierContractPath          = "contracts/src/FunnyRollupVerifier.sol"
	FunnyRollupBatchVerifierMethod                = "verifyBatch"
	FunnyRollupBatchVerifierProofSchemaVersion    = "funny-rollup-proof-envelope-v1"
	FunnyRollupBatchVerifierPublicSignalsV1       = "funny-rollup-public-signals-v1"
	FunnyRollupBatchVerifierProofDataVersion      = "funny-rollup-proof-data-v1"
	FunnyRollupBatchVerifierPlaceholderProofType  = "funny-rollup-proof-placeholder-v1"
	FunnyRollupBatchVerifierFirstGroth16ProofType = "funny-rollup-proof-groth16-bn254-2x128-shadow-state-root-gate-v1"
	FunnyRollupBatchVerifierProofVersion          = FunnyRollupBatchVerifierFirstGroth16ProofType
)

type SolidityAuthJoinStatus uint8

const (
	SolidityAuthJoinStatusUnspecified SolidityAuthJoinStatus = iota
	SolidityAuthJoinStatusJoined
	SolidityAuthJoinStatusMissingTradingKeyAuthorized
	SolidityAuthJoinStatusNonVerifierEligible
)

type JournalAppend struct {
	EntryID          string
	EntryType        string
	SourceType       string
	SourceRef        string
	OccurredAtMillis int64
	Payload          any
}

type JournalEntry struct {
	Sequence         int64           `json:"sequence"`
	EntryID          string          `json:"entry_id"`
	EntryType        string          `json:"entry_type"`
	SourceType       string          `json:"source_type"`
	SourceRef        string          `json:"source_ref"`
	OccurredAtMillis int64           `json:"occurred_at_millis"`
	Payload          json.RawMessage `json:"payload"`
}

type BatchInput struct {
	EncodingVersion string         `json:"encoding_version"`
	Entries         []JournalEntry `json:"entries"`
}

type StoredBatch struct {
	BatchID              int64
	EncodingVersion      string
	FirstSequence        int64
	LastSequence         int64
	EntryCount           int
	InputData            string
	InputHash            string
	PrevStateRoot        string
	BalancesRoot         string
	OrdersRoot           string
	PositionsFundingRoot string
	WithdrawalsRoot      string
	StateRoot            string
	CreatedAt            int64
}

type StoredSubmission struct {
	SubmissionID      string
	BatchID           int64
	EncodingVersion   string
	Status            string
	BatchDataHash     string
	NextStateRoot     string
	AuthProofHash     string
	VerifierGateHash  string
	RecordCalldata    string
	AcceptCalldata    string
	SubmissionData    string
	SubmissionHash    string
	RecordTxHash      string
	AcceptTxHash      string
	RecordSubmittedAt int64
	AcceptSubmittedAt int64
	AcceptedAt        int64
	LastError         string
	LastErrorAt       int64
	CreatedAt         int64
	UpdatedAt         int64
}

type AcceptedBatchRecord struct {
	BatchID              int64  `json:"batch_id"`
	SubmissionID         string `json:"submission_id"`
	EncodingVersion      string `json:"encoding_version"`
	FirstSequence        int64  `json:"first_sequence_no"`
	LastSequence         int64  `json:"last_sequence_no"`
	EntryCount           int    `json:"entry_count"`
	BatchDataHash        string `json:"batch_data_hash"`
	PrevStateRoot        string `json:"prev_state_root"`
	BalancesRoot         string `json:"balances_root"`
	OrdersRoot           string `json:"orders_root"`
	PositionsFundingRoot string `json:"positions_funding_root"`
	WithdrawalsRoot      string `json:"withdrawals_root"`
	NextStateRoot        string `json:"next_state_root"`
	RecordTxHash         string `json:"record_tx_hash"`
	AcceptTxHash         string `json:"accept_tx_hash"`
	AcceptedAt           int64  `json:"accepted_at"`
	CreatedAt            int64  `json:"created_at"`
	UpdatedAt            int64  `json:"updated_at"`
}

type AcceptedWithdrawalRecord struct {
	WithdrawalID     string `json:"withdrawal_id"`
	BatchID          int64  `json:"batch_id"`
	AccountID        int64  `json:"account_id"`
	WalletAddress    string `json:"wallet_address"`
	RecipientAddress string `json:"recipient_address"`
	VaultAddress     string `json:"vault_address"`
	Asset            string `json:"asset"`
	Amount           int64  `json:"amount"`
	Lane             string `json:"lane"`
	ChainName        string `json:"chain_name"`
	NetworkName      string `json:"network_name"`
	RequestSequence  int64  `json:"request_sequence"`
	ClaimID          string `json:"claim_id"`
	ClaimStatus      string `json:"claim_status"`
	ClaimTxHash      string `json:"claim_tx_hash"`
	ClaimSubmittedAt int64  `json:"claim_submitted_at"`
	ClaimedAt        int64  `json:"claimed_at"`
	LastError        string `json:"last_error"`
	LastErrorAt      int64  `json:"last_error_at"`
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
}

type AcceptedSubmissionMaterialization struct {
	Batch               AcceptedBatchRecord        `json:"batch"`
	AcceptedWithdrawals []AcceptedWithdrawalRecord `json:"accepted_withdrawals"`
	QueuedClaimRefs     []string                   `json:"queued_claim_refs"`
}

type SubmissionBatchSummary struct {
	BatchID              int64  `json:"batch_id"`
	EncodingVersion      string `json:"encoding_version"`
	FirstSequence        int64  `json:"first_sequence_no"`
	LastSequence         int64  `json:"last_sequence_no"`
	EntryCount           int    `json:"entry_count"`
	InputHash            string `json:"input_hash"`
	PrevStateRoot        string `json:"prev_state_root"`
	BalancesRoot         string `json:"balances_root"`
	OrdersRoot           string `json:"orders_root"`
	PositionsFundingRoot string `json:"positions_funding_root"`
	WithdrawalsRoot      string `json:"withdrawals_root"`
	NextStateRoot        string `json:"next_state_root"`
}

type RollupContractCall struct {
	ContractName string `json:"contract_name"`
	ContractPath string `json:"contract_path"`
	FunctionName string `json:"function_name"`
	Selector     string `json:"selector"`
	Calldata     string `json:"calldata"`
}

type ShadowBatchSubmissionBundle struct {
	SubmissionVersion       string                 `json:"submission_version"`
	Status                  string                 `json:"status"`
	ReadyForAcceptance      bool                   `json:"ready_for_acceptance"`
	Batch                   SubmissionBatchSummary `json:"batch"`
	ShadowBatchContract     ShadowBatchContract    `json:"shadow_batch_contract"`
	VerifierArtifactBundle  VerifierArtifactBundle `json:"verifier_artifact_bundle"`
	RecordBatchMetadataCall RollupContractCall     `json:"record_batch_metadata_call"`
	AcceptVerifiedBatchCall RollupContractCall     `json:"accept_verified_batch_call"`
	Blockers                []string               `json:"blockers,omitempty"`
	Limitations             []string               `json:"limitations"`
}

type PreparedShadowSubmission struct {
	StoredSubmission StoredSubmission            `json:"stored_submission"`
	Bundle           ShadowBatchSubmissionBundle `json:"bundle"`
}

type RootSet struct {
	BalancesRoot         string
	OrdersRoot           string
	PositionsFundingRoot string
	WithdrawalsRoot      string
	StateRoot            string
}

type NamespaceTruth struct {
	Namespace string `json:"namespace"`
	Mode      string `json:"mode"`
	Detail    string `json:"detail"`
}

type ShadowBatchWitness struct {
	EncodingVersion string           `json:"encoding_version"`
	Entries         []JournalEntry   `json:"entries"`
	NamespaceTruth  []NamespaceTruth `json:"namespace_truth"`
	Limitations     []string         `json:"limitations"`
}

type ShadowBatchPublicInputs struct {
	EncodingVersion      string `json:"encoding_version"`
	BatchID              int64  `json:"batch_id"`
	FirstSequence        int64  `json:"first_sequence_no"`
	LastSequence         int64  `json:"last_sequence_no"`
	EntryCount           int    `json:"entry_count"`
	BatchDataHash        string `json:"batch_data_hash"`
	PrevStateRoot        string `json:"prev_state_root"`
	BalancesRoot         string `json:"balances_root"`
	OrdersRoot           string `json:"orders_root"`
	PositionsFundingRoot string `json:"positions_funding_root"`
	WithdrawalsRoot      string `json:"withdrawals_root"`
	NextStateRoot        string `json:"next_state_root"`
}

type L1BatchMetadata struct {
	BatchID       int64  `json:"batch_id"`
	BatchDataHash string `json:"batch_data_hash"`
	PrevStateRoot string `json:"prev_state_root"`
	NextStateRoot string `json:"next_state_root"`
}

type ShadowBatchContract struct {
	Witness         ShadowBatchWitness      `json:"witness"`
	PublicInputs    ShadowBatchPublicInputs `json:"public_inputs"`
	L1BatchMetadata L1BatchMetadata         `json:"l1_batch_metadata"`
}

type VerifierTradingKeyAuthorization struct {
	BatchID             int64                          `json:"batch_id"`
	Sequence            int64                          `json:"sequence"`
	SourceRef           string                         `json:"source_ref"`
	Binding             sharedauth.VerifierAuthBinding `json:"binding"`
	WalletTypedDataHash string                         `json:"wallet_typed_data_hash"`
	WalletSignature     string                         `json:"wallet_signature"`
}

type VerifierNonceAuthorization struct {
	BatchID           int64                           `json:"batch_id"`
	Sequence          int64                           `json:"sequence"`
	SourceRef         string                          `json:"source_ref"`
	AuthVersion       string                          `json:"auth_version"`
	AuthorizationRef  string                          `json:"authorization_ref,omitempty"`
	JoinStatus        string                          `json:"join_status"`
	IneligibleReason  string                          `json:"ineligible_reason,omitempty"`
	Binding           *sharedauth.VerifierAuthBinding `json:"binding,omitempty"`
	IntentMessageHash string                          `json:"intent_message_hash,omitempty"`
	IntentSignature   string                          `json:"intent_signature,omitempty"`
}

type VerifierAuthProofContract struct {
	JoinKey                  string                            `json:"join_key"`
	ReadyForVerifier         bool                              `json:"ready_for_verifier"`
	TradingKeyAuthorizations []VerifierTradingKeyAuthorization `json:"trading_key_authorizations"`
	NonceAuthorizations      []VerifierNonceAuthorization      `json:"nonce_authorizations"`
	Limitations              []string                          `json:"limitations"`
}

type VerifierGateBatchContract struct {
	PublicInputs    ShadowBatchPublicInputs   `json:"public_inputs"`
	L1BatchMetadata L1BatchMetadata           `json:"l1_batch_metadata"`
	AuthProof       VerifierAuthProofContract `json:"auth_proof"`
	Limitations     []string                  `json:"limitations"`
}

type VerifierAcceptanceAuthStatus struct {
	Sequence   int64  `json:"sequence"`
	SourceRef  string `json:"source_ref"`
	JoinStatus string `json:"join_status"`
}

type VerifierAcceptanceSolidityComponent struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type VerifierAcceptanceSolidityArgument struct {
	Name       string                                `json:"name"`
	Type       string                                `json:"type"`
	Provided   bool                                  `json:"provided"`
	Components []VerifierAcceptanceSolidityComponent `json:"components,omitempty"`
}

type VerifierAcceptanceSolidityEnumValue struct {
	Name  string                 `json:"name"`
	Value SolidityAuthJoinStatus `json:"value"`
}

type VerifierAcceptanceSoliditySchema struct {
	ContractName         string                                `json:"contract_name"`
	ContractPath         string                                `json:"contract_path"`
	FunctionName         string                                `json:"function_name"`
	Arguments            []VerifierAcceptanceSolidityArgument  `json:"arguments"`
	AuthStatusEnumValues []VerifierAcceptanceSolidityEnumValue `json:"auth_status_enum_values"`
}

type SolidityVerifierPublicInputs struct {
	BatchID              uint64 `json:"batch_id"`
	FirstSequence        uint64 `json:"first_sequence_no"`
	LastSequence         uint64 `json:"last_sequence_no"`
	EntryCount           uint64 `json:"entry_count"`
	BatchDataHash        string `json:"batch_data_hash"`
	PrevStateRoot        string `json:"prev_state_root"`
	BalancesRoot         string `json:"balances_root"`
	OrdersRoot           string `json:"orders_root"`
	PositionsFundingRoot string `json:"positions_funding_root"`
	WithdrawalsRoot      string `json:"withdrawals_root"`
	NextStateRoot        string `json:"next_state_root"`
}

type SolidityL1BatchMetadata struct {
	BatchID       uint64 `json:"batch_id"`
	BatchDataHash string `json:"batch_data_hash"`
	PrevStateRoot string `json:"prev_state_root"`
	NextStateRoot string `json:"next_state_root"`
}

type VerifierAcceptanceSolidityCalldata struct {
	PublicInputs   SolidityVerifierPublicInputs `json:"public_inputs"`
	MetadataSubset SolidityL1BatchMetadata      `json:"metadata_subset"`
	AuthStatuses   []SolidityAuthJoinStatus     `json:"auth_statuses"`
}

type VerifierAcceptanceSolidityExport struct {
	Schema   VerifierAcceptanceSoliditySchema   `json:"schema"`
	Calldata VerifierAcceptanceSolidityCalldata `json:"calldata"`
}

type VerifierGateDigestField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type VerifierProofSchemaField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type VerifierAuthProofDigestContract struct {
	HashFunction  string                   `json:"hash_function"`
	ArgumentType  string                   `json:"argument_type"`
	AuthStatuses  []SolidityAuthJoinStatus `json:"auth_statuses"`
	AuthProofHash string                   `json:"auth_proof_hash"`
}

type VerifierGateDigestContract struct {
	EncodingVersion     string                       `json:"encoding_version"`
	EncodingVersionHash string                       `json:"encoding_version_hash"`
	HashFunction        string                       `json:"hash_function"`
	FieldOrder          []VerifierGateDigestField    `json:"field_order"`
	PublicInputs        SolidityVerifierPublicInputs `json:"public_inputs"`
	AuthProofHash       string                       `json:"auth_proof_hash"`
	VerifierGateHash    string                       `json:"verifier_gate_hash"`
}

type SolidityVerifierGateContext struct {
	BatchEncodingHash string                       `json:"batch_encoding_hash"`
	PublicInputs      SolidityVerifierPublicInputs `json:"public_inputs"`
	AuthProofHash     string                       `json:"auth_proof_hash"`
	VerifierGateHash  string                       `json:"verifier_gate_hash"`
}

type VerifierProofPublicSignalsCalldata struct {
	BatchEncodingHash string `json:"batch_encoding_hash"`
	AuthProofHash     string `json:"auth_proof_hash"`
	VerifierGateHash  string `json:"verifier_gate_hash"`
}

type VerifierProofDataCalldata struct {
	ProofDataSchemaHash string `json:"proof_data_schema_hash"`
	ProofTypeHash       string `json:"proof_type_hash"`
	BatchEncodingHash   string `json:"batch_encoding_hash"`
	AuthProofHash       string `json:"auth_proof_hash"`
	VerifierGateHash    string `json:"verifier_gate_hash"`
	ProofBytes          string `json:"proof_bytes"`
}

type VerifierInterfaceSolidityCalldata struct {
	Context         SolidityVerifierGateContext        `json:"context"`
	PublicSignals   VerifierProofPublicSignalsCalldata `json:"public_signals"`
	ProofDataFields VerifierProofDataCalldata          `json:"proof_data_fields"`
	ProofData       string                             `json:"proof_data"`
	Proof           string                             `json:"proof"`
}

type VerifierGroth16ProofTuple struct {
	A [2]string    `json:"a"`
	B [2][2]string `json:"b"`
	C [2]string    `json:"c"`
}

type VerifierGroth16Fixture struct {
	ProofBytesEncoding    string                     `json:"proof_bytes_encoding"`
	PublicInputFieldOrder []VerifierProofSchemaField `json:"public_input_field_order"`
	PublicInputs          []string                   `json:"public_inputs"`
	ProofTuple            VerifierGroth16ProofTuple  `json:"proof_tuple"`
	ExpectedVerdict       bool                       `json:"expected_verdict"`
}

type VerifierInterfaceSolidityExport struct {
	ContractName             string                            `json:"contract_name"`
	ImplementationName       string                            `json:"implementation_name"`
	ContractPath             string                            `json:"contract_path"`
	FunctionName             string                            `json:"function_name"`
	ContextType              string                            `json:"context_type"`
	PublicInputsType         string                            `json:"public_inputs_type"`
	PublicSignalsType        string                            `json:"public_signals_type"`
	PublicSignalsVersion     string                            `json:"public_signals_version"`
	PublicSignalsVersionHash string                            `json:"public_signals_version_hash"`
	PublicSignalsFieldOrder  []VerifierProofSchemaField        `json:"public_signals_field_order"`
	ProofType                string                            `json:"proof_type"`
	ProofSchemaVersion       string                            `json:"proof_schema_version"`
	ProofSchemaHash          string                            `json:"proof_schema_hash"`
	ProofFieldOrder          []VerifierProofSchemaField        `json:"proof_field_order"`
	ProofDataSchemaVersion   string                            `json:"proof_data_schema_version"`
	ProofDataSchemaHash      string                            `json:"proof_data_schema_hash"`
	ProofDataFieldOrder      []VerifierProofSchemaField        `json:"proof_data_field_order"`
	ProofEncoding            string                            `json:"proof_encoding"`
	ProofDataEncoding        string                            `json:"proof_data_encoding"`
	ProofVersion             string                            `json:"proof_version"`
	ProofVersionHash         string                            `json:"proof_version_hash"`
	Groth16Fixture           VerifierGroth16Fixture            `json:"groth16_fixture"`
	Calldata                 VerifierInterfaceSolidityCalldata `json:"calldata"`
}

type VerifierArtifactBundle struct {
	AcceptanceContract VerifierStateRootAcceptanceContract `json:"acceptance_contract"`
	AuthProofDigest    VerifierAuthProofDigestContract     `json:"auth_proof_digest"`
	VerifierGateDigest VerifierGateDigestContract          `json:"verifier_gate_digest"`
	VerifierInterface  VerifierInterfaceSolidityExport     `json:"verifier_interface"`
	Limitations        []string                            `json:"limitations"`
}

type VerifierStateRootAcceptanceContract struct {
	PublicInputs       ShadowBatchPublicInputs          `json:"public_inputs"`
	L1BatchMetadata    L1BatchMetadata                  `json:"l1_batch_metadata"`
	ReadyForAcceptance bool                             `json:"ready_for_acceptance"`
	AuthStatuses       []VerifierAcceptanceAuthStatus   `json:"auth_statuses"`
	SolidityExport     VerifierAcceptanceSolidityExport `json:"solidity_export"`
	Limitations        []string                         `json:"limitations"`
}

type OrderAcceptedPayload struct {
	OrderID           string `json:"order_id"`
	CommandID         string `json:"command_id,omitempty"`
	ClientOrderID     string `json:"client_order_id,omitempty"`
	AccountID         int64  `json:"account_id"`
	MarketID          int64  `json:"market_id"`
	Outcome           string `json:"outcome"`
	Side              string `json:"side"`
	OrderType         string `json:"order_type"`
	TimeInForce       string `json:"time_in_force"`
	CollateralAsset   string `json:"collateral_asset"`
	ReserveAsset      string `json:"reserve_asset"`
	ReserveAmount     int64  `json:"reserve_amount"`
	Price             int64  `json:"price"`
	Quantity          int64  `json:"quantity"`
	RequestedAtMillis int64  `json:"requested_at_millis"`
}

type NonceAdvancedPayload struct {
	AccountID          int64                                 `json:"account_id"`
	AuthKeyID          string                                `json:"auth_key_id"`
	Scope              string                                `json:"scope"`
	KeyStatus          string                                `json:"key_status"`
	AcceptedNonce      uint64                                `json:"accepted_nonce"`
	NextNonce          uint64                                `json:"next_nonce"`
	OccurredAtMillis   int64                                 `json:"occurred_at_millis"`
	OrderAuthorization *sharedauth.OrderAuthorizationWitness `json:"order_authorization,omitempty"`
}

type TradingKeyAuthorizedPayload struct {
	AuthorizationWitness sharedauth.TradingKeyAuthorizationWitness `json:"authorization_witness"`
}

type OrderCancelledPayload struct {
	OrderID           string `json:"order_id"`
	AccountID         int64  `json:"account_id"`
	MarketID          int64  `json:"market_id"`
	Outcome           string `json:"outcome"`
	Side              string `json:"side"`
	ReserveAsset      string `json:"reserve_asset"`
	Price             int64  `json:"price"`
	RemainingQuantity int64  `json:"remaining_quantity"`
	CancelReason      string `json:"cancel_reason,omitempty"`
}

type TradeMatchedPayload struct {
	TradeID          string `json:"trade_id"`
	Sequence         uint64 `json:"sequence"`
	CollateralAsset  string `json:"collateral_asset"`
	MarketID         int64  `json:"market_id"`
	Outcome          string `json:"outcome"`
	Price            int64  `json:"price"`
	Quantity         int64  `json:"quantity"`
	TakerOrderID     string `json:"taker_order_id"`
	MakerOrderID     string `json:"maker_order_id"`
	TakerAccountID   int64  `json:"taker_account_id"`
	MakerAccountID   int64  `json:"maker_account_id"`
	TakerSide        string `json:"taker_side"`
	MakerSide        string `json:"maker_side"`
	OccurredAtMillis int64  `json:"occurred_at_millis"`
}

type DepositCreditedPayload struct {
	DepositID        string `json:"deposit_id"`
	AccountID        int64  `json:"account_id"`
	WalletAddress    string `json:"wallet_address"`
	VaultAddress     string `json:"vault_address"`
	Asset            string `json:"asset"`
	Amount           int64  `json:"amount"`
	ChainName        string `json:"chain_name"`
	NetworkName      string `json:"network_name"`
	TxHash           string `json:"tx_hash"`
	LogIndex         int64  `json:"log_index"`
	BlockNumber      int64  `json:"block_number"`
	OccurredAtMillis int64  `json:"occurred_at_millis"`
}

type WithdrawalRequestedPayload struct {
	WithdrawalID     string `json:"withdrawal_id"`
	AccountID        int64  `json:"account_id"`
	WalletAddress    string `json:"wallet_address"`
	RecipientAddress string `json:"recipient_address"`
	VaultAddress     string `json:"vault_address"`
	Asset            string `json:"asset"`
	Amount           int64  `json:"amount"`
	Lane             string `json:"lane"`
	ChainName        string `json:"chain_name"`
	NetworkName      string `json:"network_name"`
	TxHash           string `json:"tx_hash"`
	LogIndex         int64  `json:"log_index"`
	BlockNumber      int64  `json:"block_number"`
	OccurredAtMillis int64  `json:"occurred_at_millis"`
}

type MarketResolvedPayload struct {
	MarketID         int64  `json:"market_id"`
	ResolvedOutcome  string `json:"resolved_outcome"`
	ResolverType     string `json:"resolver_type"`
	ResolverRef      string `json:"resolver_ref"`
	EvidenceHash     string `json:"evidence_hash"`
	OccurredAtMillis int64  `json:"occurred_at_millis"`
}

type SettlementPayoutPayload struct {
	EventID          string `json:"event_id"`
	MarketID         int64  `json:"market_id"`
	AccountID        int64  `json:"account_id"`
	WinningOutcome   string `json:"winning_outcome"`
	PositionAsset    string `json:"position_asset"`
	SettledQuantity  int64  `json:"settled_quantity"`
	PayoutAsset      string `json:"payout_asset"`
	PayoutAmount     int64  `json:"payout_amount"`
	OccurredAtMillis int64  `json:"occurred_at_millis"`
}
