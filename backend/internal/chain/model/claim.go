package model

type ClaimTask struct {
	ID               int64
	BizType          string
	RefID            string
	ChainName        string
	NetworkName      string
	WalletAddress    string
	RecipientAddress string
	TxHash           string
	Status           string
	PayoutAsset      string
	PayoutAmount     int64
	AttemptCount     int64
	ErrorMessage     string
	CreatedAt        int64
	UpdatedAt        int64
}
