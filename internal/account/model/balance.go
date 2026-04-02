package model

type Balance struct {
	UserID    int64
	Asset     string
	Available int64
	Frozen    int64
}

func (b Balance) Total() int64 {
	return b.Available + b.Frozen
}

type FreezeRecord struct {
	FreezeID       string
	UserID         int64
	Asset          string
	RefType        string
	RefID          string
	OriginalAmount int64
	Amount         int64
	Released       bool
	Consumed       bool
}
