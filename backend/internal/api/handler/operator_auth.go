package handler

import (
	"math"
	"net/http"
	"strings"
	"time"

	"funnyoption/internal/api/dto"
	sharedauth "funnyoption/internal/shared/auth"

	"github.com/gin-gonic/gin"
)

const operatorSignatureWindow = 5 * time.Minute

type verifiedOperatorAction struct {
	WalletAddress string
	RequestedAt   int64
}

func normalizeOperatorWalletSet(wallets []string) map[string]struct{} {
	if len(wallets) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(wallets))
	for _, wallet := range wallets {
		normalized := sharedauth.NormalizeHex(wallet)
		if normalized != "" {
			set[normalized] = struct{}{}
		}
	}
	return set
}

func (h *OrderHandler) privilegedOperatorUserID() int64 {
	if h.operatorUserID > 0 {
		return h.operatorUserID
	}
	return 1001
}

func (h *OrderHandler) requirePrivilegedOperator(ctx *gin.Context, operator *dto.OperatorAction, message string) (*verifiedOperatorAction, bool) {
	if len(h.operatorWallets) == 0 {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "operator wallets are not configured for privileged actions"})
		return nil, false
	}
	if operator == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "operator proof is required for privileged actions"})
		return nil, false
	}

	walletAddress := sharedauth.NormalizeHex(operator.WalletAddress)
	signature := strings.TrimSpace(operator.Signature)
	if walletAddress == "" || signature == "" || operator.RequestedAt <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "operator wallet, signature, and requested_at are required"})
		return nil, false
	}

	ageMillis := math.Abs(float64(time.Now().UnixMilli() - operator.RequestedAt))
	if ageMillis > float64(operatorSignatureWindow/time.Millisecond) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "operator signature expired"})
		return nil, false
	}

	recoveredWallet, err := sharedauth.RecoverPersonalSignAddress(message, signature)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid operator signature"})
		return nil, false
	}
	if recoveredWallet != walletAddress {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "operator signature does not match wallet"})
		return nil, false
	}
	if _, ok := h.operatorWallets[recoveredWallet]; !ok {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "wallet is not authorized for operator actions"})
		return nil, false
	}

	return &verifiedOperatorAction{
		WalletAddress: recoveredWallet,
		RequestedAt:   operator.RequestedAt,
	}, true
}
