package service

import (
	"sort"
	"time"

	"funnyoption/internal/ledger/model"
)

func BuildReport(liabilities []model.LiabilitySnapshot, chains []model.ChainSnapshot) model.Report {
	liabilityMap := make(map[string]model.LiabilitySnapshot)
	chainMap := make(map[string]model.ChainSnapshot)
	assets := make(map[string]struct{})

	for _, item := range liabilities {
		asset := normalizeAsset(item.Asset)
		item.Asset = asset
		liabilityMap[asset] = item
		assets[asset] = struct{}{}
	}
	for _, item := range chains {
		asset := normalizeAsset(item.Asset)
		item.Asset = asset
		chainMap[asset] = item
		assets[asset] = struct{}{}
	}

	keys := make([]string, 0, len(assets))
	for asset := range assets {
		keys = append(keys, asset)
	}
	sort.Strings(keys)

	report := model.Report{
		GeneratedAt: time.Now(),
		Lines:       make([]model.ReportLine, 0, len(keys)),
	}
	for _, asset := range keys {
		internal := liabilityMap[asset].InternalTotal()
		chain := chainMap[asset].ChainTotal()
		diff := chain - internal
		status := "BALANCED"
		if diff != 0 {
			status = "DRIFT"
		}
		report.Lines = append(report.Lines, model.ReportLine{
			Asset:         asset,
			InternalTotal: internal,
			ChainTotal:    chain,
			Difference:    diff,
			Status:        status,
		})
	}
	return report
}
