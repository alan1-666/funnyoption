package model

import "time"

type BizType string

const (
	BizTypeTrade      BizType = "TRADE"
	BizTypeFee        BizType = "FEE"
	BizTypeDeposit    BizType = "DEPOSIT"
	BizTypeWithdraw   BizType = "WITHDRAW"
	BizTypeTransfer   BizType = "TRANSFER"
	BizTypeSettlement BizType = "SETTLEMENT"
)

type Direction string

const (
	DirectionDebit  Direction = "DEBIT"
	DirectionCredit Direction = "CREDIT"
)

type EntryStatus string

const (
	EntryStatusConfirmed EntryStatus = "CONFIRMED"
)

type Posting struct {
	Account   string
	Asset     string
	Direction Direction
	Amount    int64
}

type Entry struct {
	EntryID   string
	BizType   BizType
	RefID     string
	Postings  []Posting
	CreatedAt time.Time
	Status    EntryStatus
}
