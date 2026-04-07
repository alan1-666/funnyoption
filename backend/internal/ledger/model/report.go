package model

import "time"

type LiabilitySnapshot struct {
	Asset                string
	UserAvailable        int64
	UserFrozen           int64
	PendingSettlement    int64
	PendingWithdraw      int64
	PlatformFeeLiability int64
}

func (s LiabilitySnapshot) InternalTotal() int64 {
	return s.UserAvailable + s.UserFrozen + s.PendingSettlement + s.PendingWithdraw + s.PlatformFeeLiability
}

type ChainSnapshot struct {
	Asset          string
	HotWallet      int64
	ColdWallet     int64
	ContractLocked int64
}

func (s ChainSnapshot) ChainTotal() int64 {
	return s.HotWallet + s.ColdWallet + s.ContractLocked
}

type ReportLine struct {
	Asset         string
	InternalTotal int64
	ChainTotal    int64
	Difference    int64
	Status        string
}

type Report struct {
	GeneratedAt time.Time
	Lines       []ReportLine
}
