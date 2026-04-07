package model

const (
	ForcedWithdrawalSatisfactionStatusNone      = "NONE"
	ForcedWithdrawalSatisfactionStatusReady     = "READY"
	ForcedWithdrawalSatisfactionStatusSubmitted = "SUBMITTED"
	ForcedWithdrawalSatisfactionStatusFailed    = "FAILED"
	ForcedWithdrawalSatisfactionStatusAmbiguous = "AMBIGUOUS"
	ForcedWithdrawalSatisfactionStatusSatisfied = "SATISFIED"
)

type RollupForcedWithdrawalRequest struct {
	RequestID               int64
	WalletAddress           string
	RecipientAddress        string
	Amount                  int64
	RequestedAt             int64
	DeadlineAt              int64
	SatisfiedClaimID        string
	SatisfiedAt             int64
	FrozenAt                int64
	Status                  string
	MatchedWithdrawalID     string
	MatchedClaimID          string
	SatisfactionStatus      string
	SatisfactionTxHash      string
	SatisfactionSubmittedAt int64
	SatisfactionLastError   string
	SatisfactionLastErrorAt int64
	CreatedAt               int64
	UpdatedAt               int64
}

type RollupFreezeState struct {
	Frozen    bool
	FrozenAt  int64
	RequestID int64
	UpdatedAt int64
}

type ForcedWithdrawalClaimMatch struct {
	WithdrawalID string
	ClaimID      string
	Amount       int64
	ClaimedAt    int64
}
