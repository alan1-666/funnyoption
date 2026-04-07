package dto

import (
	"hash/fnv"
	"strings"
)

var allowedAvatarPresets = map[string]struct{}{
	"aurora": {},
	"ember":  {},
	"ocean":  {},
	"violet": {},
	"mono":   {},
	"forest": {},
}

var avatarPresetOrder = []string{
	"aurora",
	"ember",
	"ocean",
	"violet",
	"mono",
	"forest",
}

type GetUserProfileRequest struct {
	UserID        int64  `form:"user_id"`
	WalletAddress string `form:"wallet_address"`
}

type UpdateUserProfileRequest struct {
	UserID       int64  `json:"user_id" binding:"required"`
	SessionID    string `json:"session_id" binding:"required"`
	DisplayName  string `json:"display_name"`
	AvatarPreset string `json:"avatar_preset" binding:"required"`
}

type UserProfileResponse struct {
	UserID        int64  `json:"user_id"`
	WalletAddress string `json:"wallet_address"`
	DisplayName   string `json:"display_name"`
	AvatarPreset  string `json:"avatar_preset"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

func NormalizeUserDisplayName(value string) string {
	cleaned := strings.TrimSpace(value)
	if len(cleaned) > 32 {
		return cleaned[:32]
	}
	return cleaned
}

func NormalizeAvatarPreset(value string) (string, bool) {
	cleaned := strings.ToLower(strings.TrimSpace(value))
	_, ok := allowedAvatarPresets[cleaned]
	return cleaned, ok
}

func DefaultAvatarPreset(userID int64, walletAddress string) string {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(strings.ToLower(strings.TrimSpace(walletAddress))))
	sum := int(hasher.Sum32()) + int(userID%int64(len(avatarPresetOrder)+1))
	if sum < 0 {
		sum = -sum
	}
	return avatarPresetOrder[sum%len(avatarPresetOrder)]
}
