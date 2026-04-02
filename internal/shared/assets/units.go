package assets

import (
	"fmt"
	"math"
)

const DefaultCollateralDisplayDigits = 2

func AccountingDigits(asset string) int {
	switch NormalizeAsset(asset) {
	case DefaultCollateralAsset:
		return DefaultCollateralDisplayDigits
	default:
		return 0
	}
}

func ChainToAccountingAmount(rawAmount int64, chainDecimals int, accountingDigits int) (int64, error) {
	if rawAmount < 0 {
		return 0, fmt.Errorf("amount must not be negative")
	}
	if chainDecimals < 0 || accountingDigits < 0 {
		return 0, fmt.Errorf("decimals must not be negative")
	}
	if chainDecimals == accountingDigits {
		return rawAmount, nil
	}

	diff := chainDecimals - accountingDigits
	if diff > 0 {
		factor, err := pow10Int64(diff)
		if err != nil {
			return 0, err
		}
		if rawAmount%factor != 0 {
			return 0, fmt.Errorf("amount %d exceeds supported precision for %d accounting digits", rawAmount, accountingDigits)
		}
		return rawAmount / factor, nil
	}

	factor, err := pow10Int64(-diff)
	if err != nil {
		return 0, err
	}
	if rawAmount > math.MaxInt64/factor {
		return 0, fmt.Errorf("amount overflows int64")
	}
	return rawAmount * factor, nil
}

func AccountingToChainAmount(accountingAmount int64, chainDecimals int, accountingDigits int) (int64, error) {
	if accountingAmount < 0 {
		return 0, fmt.Errorf("amount must not be negative")
	}
	if chainDecimals < 0 || accountingDigits < 0 {
		return 0, fmt.Errorf("decimals must not be negative")
	}
	if chainDecimals == accountingDigits {
		return accountingAmount, nil
	}

	diff := chainDecimals - accountingDigits
	if diff > 0 {
		factor, err := pow10Int64(diff)
		if err != nil {
			return 0, err
		}
		if accountingAmount > math.MaxInt64/factor {
			return 0, fmt.Errorf("amount overflows int64")
		}
		return accountingAmount * factor, nil
	}

	factor, err := pow10Int64(-diff)
	if err != nil {
		return 0, err
	}
	if accountingAmount%factor != 0 {
		return 0, fmt.Errorf("amount %d exceeds supported precision for %d chain decimals", accountingAmount, chainDecimals)
	}
	return accountingAmount / factor, nil
}

func pow10Int64(exp int) (int64, error) {
	if exp < 0 {
		return 0, fmt.Errorf("negative exponent")
	}
	value := int64(1)
	for i := 0; i < exp; i++ {
		if value > math.MaxInt64/10 {
			return 0, fmt.Errorf("pow10 overflows int64")
		}
		value *= 10
	}
	return value, nil
}
