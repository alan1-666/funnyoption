package model

type Withdrawal struct {
	WithdrawalID     string
	UserID           int64
	WalletAddress    string
	RecipientAddress string
	VaultAddress     string
	Asset            string
	Amount           int64
	ChainName        string
	NetworkName      string
	TxHash           string
	LogIndex         int64
	BlockNumber      int64
	Status           string
	DebitedAt        int64
	CreatedAt        int64
	UpdatedAt        int64
}
