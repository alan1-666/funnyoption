package model

type Deposit struct {
	DepositID     string
	UserID        int64
	WalletAddress string
	VaultAddress  string
	Asset         string
	Amount        int64
	ChainName     string
	NetworkName   string
	TxHash        string
	LogIndex      int64
	BlockNumber   int64
	Status        string
	CreditedAt    int64
	CreatedAt     int64
	UpdatedAt     int64
}
