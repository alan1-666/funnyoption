package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"funnyoption/internal/api/dto"
	sharedkafka "funnyoption/internal/shared/kafka"

	"github.com/gin-gonic/gin"
)

type ProposeMarketOption struct {
	Label      string `json:"label"`
	ShortLabel string `json:"short_label,omitempty"`
}

type ProposeMarketRequest struct {
	Title            string               `json:"title" binding:"required"`
	Description      string               `json:"description"`
	CategoryKey      string               `json:"category_key"`
	CloseAt          int64                `json:"close_at"`
	ResolveAt        int64                `json:"resolve_at"`
	ResolutionSource string               `json:"resolution_source"`
	Options          []ProposeMarketOption `json:"options"`
}

func (h *OrderHandler) ProposeMarket(ctx *gin.Context) {
	authRaw, exists := ctx.Get("api.authenticated_user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
		return
	}
	userID := authRaw.(int64)

	var req ProposeMarketRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	title := strings.TrimSpace(req.Title)
	if len(title) < 5 || len(title) > 200 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "title must be between 5 and 200 characters"})
		return
	}

	now := time.Now().Unix()
	if req.CloseAt > 0 && req.CloseAt <= now {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "close_at must be in the future"})
		return
	}
	if req.ResolveAt > 0 && req.CloseAt > 0 && req.ResolveAt < req.CloseAt {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "resolve_at must be >= close_at"})
		return
	}

	var options []dto.MarketOption
	for i, opt := range req.Options {
		label := strings.TrimSpace(opt.Label)
		if label == "" {
			continue
		}
		short := strings.TrimSpace(opt.ShortLabel)
		if short == "" {
			short = label
		}
		key := strings.ToUpper(strings.ReplaceAll(label, " ", "_"))
		if len(key) > 32 {
			key = key[:32]
		}
		options = append(options, dto.MarketOption{
			Key:        key,
			Label:      label,
			ShortLabel: short,
			SortOrder:  (i + 1) * 10,
			IsActive:   true,
		})
	}

	metadata := map[string]any{}
	if src := strings.TrimSpace(req.ResolutionSource); src != "" {
		metadata["resolution_source"] = src
	}
	metadataJSON, _ := json.Marshal(metadata)

	marketID := time.Now().UnixMilli()
	createReq := dto.CreateMarketRequest{
		MarketID:    marketID,
		Title:       title,
		Description: strings.TrimSpace(req.Description),
		CategoryKey: strings.TrimSpace(req.CategoryKey),
		Status:      "PENDING_REVIEW",
		CloseAt:     req.CloseAt,
		ResolveAt:   req.ResolveAt,
		CreatedBy:   userID,
		Metadata:    metadataJSON,
		Options:     options,
	}

	market, err := h.store.CreateMarket(ctx, createReq)
	if err != nil {
		h.logger.Error("propose market failed", "user_id", userID, "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create market proposal"})
		return
	}

	ctx.JSON(http.StatusCreated, market)
}

func (h *OrderHandler) ApproveMarket(ctx *gin.Context) {
	marketID, err := strconv.ParseInt(ctx.Param("market_id"), 10, 64)
	if err != nil || marketID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "valid market_id is required"})
		return
	}

	market, err := h.store.GetMarket(ctx, marketID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
		return
	}
	if market.Status != "PENDING_REVIEW" {
		ctx.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("market is %s, not PENDING_REVIEW", market.Status)})
		return
	}

	now := time.Now().Unix()

	_, err = h.store.(*SQLStore).db.ExecContext(ctx.Request.Context(), `
		UPDATE markets SET
			status = 'OPEN',
			open_at = $1,
			updated_at = $2
		WHERE market_id = $3 AND status = 'PENDING_REVIEW'
	`, now, now, marketID)
	if err != nil {
		h.logger.Error("approve market failed", "market_id", marketID, "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve market"})
		return
	}

	eventID := fmt.Sprintf("mkt-approve-%d-%d", marketID, now)
	_ = h.publisher.PublishJSON(ctx.Request.Context(), h.topics.MarketEvent, fmt.Sprintf("%d", marketID), sharedkafka.MarketEvent{
		EventID:          eventID,
		MarketID:         marketID,
		Status:           "OPEN",
		OccurredAtMillis: now * 1000,
	})

	if market.CreatedBy > 0 {
		_, _ = InsertNotification(ctx.Request.Context(), h.store.(*SQLStore).db,
			market.CreatedBy, "proposal_approved",
			fmt.Sprintf("Your market proposal \"%s\" has been approved", market.Title), "",
			fmt.Sprintf(`{"market_id":%d}`, marketID))
	}

	updated, _ := h.store.GetMarket(ctx, marketID)
	ctx.JSON(http.StatusOK, updated)
}

func (h *OrderHandler) RejectMarket(ctx *gin.Context) {
	marketID, err := strconv.ParseInt(ctx.Param("market_id"), 10, 64)
	if err != nil || marketID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "valid market_id is required"})
		return
	}

	market, err := h.store.GetMarket(ctx, marketID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
		return
	}
	if market.Status != "PENDING_REVIEW" {
		ctx.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("market is %s, not PENDING_REVIEW", market.Status)})
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = ctx.ShouldBindJSON(&body)

	now := time.Now().Unix()
	_, err = h.store.(*SQLStore).db.ExecContext(ctx.Request.Context(), `
		UPDATE markets SET status = 'REJECTED', updated_at = $1 WHERE market_id = $2 AND status = 'PENDING_REVIEW'
	`, now, marketID)
	if err != nil {
		h.logger.Error("reject market failed", "market_id", marketID, "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject market"})
		return
	}

	if market.CreatedBy > 0 {
		reason := strings.TrimSpace(body.Reason)
		notifBody := ""
		if reason != "" {
			notifBody = "Reason: " + reason
		}
		_, _ = InsertNotification(ctx.Request.Context(), h.store.(*SQLStore).db,
			market.CreatedBy, "proposal_rejected",
			fmt.Sprintf("Your market proposal \"%s\" has been rejected", market.Title), notifBody,
			fmt.Sprintf(`{"market_id":%d}`, marketID))
	}

	updated, _ := h.store.GetMarket(ctx, marketID)
	ctx.JSON(http.StatusOK, updated)
}

func (h *OrderHandler) getDBFromStore() *sql.DB {
	if store, ok := h.store.(*SQLStore); ok {
		return store.db
	}
	return nil
}
