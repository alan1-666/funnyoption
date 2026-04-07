package handler

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type NotificationRow struct {
	NotificationID int64  `json:"notification_id"`
	UserID         int64  `json:"user_id"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	Body           string `json:"body"`
	Metadata       string `json:"metadata"`
	IsRead         bool   `json:"is_read"`
	CreatedAt      int64  `json:"created_at"`
}

type NotificationHandler struct {
	db *sql.DB
}

func NewNotificationHandler(db *sql.DB) *NotificationHandler {
	return &NotificationHandler{db: db}
}

func (h *NotificationHandler) ListNotifications(ctx *gin.Context) {
	userID, err := strconv.ParseInt(strings.TrimSpace(ctx.Query("user_id")), 10, 64)
	if err != nil || userID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "valid user_id is required"})
		return
	}

	limit := 20
	if v := strings.TrimSpace(ctx.Query("limit")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	rows, err := h.db.QueryContext(ctx.Request.Context(), `
		SELECT notification_id, user_id, type, title, body, metadata::text, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query notifications"})
		return
	}
	defer rows.Close()

	items := make([]NotificationRow, 0, limit)
	for rows.Next() {
		var n NotificationRow
		if err := rows.Scan(&n.NotificationID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Metadata, &n.IsRead, &n.CreatedAt); err != nil {
			continue
		}
		items = append(items, n)
	}

	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *NotificationHandler) UnreadCount(ctx *gin.Context) {
	userID, err := strconv.ParseInt(strings.TrimSpace(ctx.Query("user_id")), 10, 64)
	if err != nil || userID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "valid user_id is required"})
		return
	}

	var count int64
	err = h.db.QueryRowContext(ctx.Request.Context(), `
		SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE
	`, userID).Scan(&count)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count notifications"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"count": count})
}

func (h *NotificationHandler) MarkRead(ctx *gin.Context) {
	notifID, err := strconv.ParseInt(ctx.Param("notification_id"), 10, 64)
	if err != nil || notifID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "valid notification_id is required"})
		return
	}

	authRaw, exists := ctx.Get("api.authenticated_user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
		return
	}
	authUserID := authRaw.(int64)

	result, err := h.db.ExecContext(ctx.Request.Context(), `
		UPDATE notifications SET is_read = TRUE
		WHERE notification_id = $1 AND user_id = $2 AND is_read = FALSE
	`, notifID, authUserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark notification"})
		return
	}

	affected, _ := result.RowsAffected()
	ctx.JSON(http.StatusOK, gin.H{"updated": affected})
}

func (h *NotificationHandler) MarkAllRead(ctx *gin.Context) {
	authRaw, exists := ctx.Get("api.authenticated_user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
		return
	}
	authUserID := authRaw.(int64)

	result, err := h.db.ExecContext(ctx.Request.Context(), `
		UPDATE notifications SET is_read = TRUE
		WHERE user_id = $1 AND is_read = FALSE
	`, authUserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark notifications"})
		return
	}

	affected, _ := result.RowsAffected()
	ctx.JSON(http.StatusOK, gin.H{"updated": affected})
}

func InsertNotification(ctx context.Context, db *sql.DB, userID int64, notifType, title, body string, metadata string) (int64, error) {
	if metadata == "" {
		metadata = "{}"
	}
	now := time.Now().Unix()
	var id int64
	err := db.QueryRowContext(ctx, `
		INSERT INTO notifications (user_id, type, title, body, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
		RETURNING notification_id
	`, userID, notifType, title, body, metadata, now).Scan(&id)
	return id, err
}
