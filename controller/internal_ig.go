package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type upsertIgTokenRequest struct {
	Group     string `json:"group"`
	UserID    int    `json:"user_id"`
	IGUserID  int    `json:"ig_user_id"`
	ExpiresAt string `json:"expires_at"`
	RotateKey bool   `json:"rotate_key"`
}

func InternalUpsertIgToken(c *gin.Context) {
	subscriptionID, err := strconv.ParseInt(c.Param("subscription_id"), 10, 64)
	if err != nil || subscriptionID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid subscription_id"})
		return
	}
	var req upsertIgTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid payload"})
		return
	}
	if req.UserID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "user_id is required"})
		return
	}
	if req.IGUserID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ig_user_id is required"})
		return
	}
	user, err := model.GetUserByID(req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "user not found"})
		return
	}
	var expiresAt *time.Time
	trimmedExpires := strings.TrimSpace(req.ExpiresAt)
	if trimmedExpires != "" {
		parsed, err := time.Parse(time.RFC3339, trimmedExpires)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid expires_at"})
			return
		}
		expiresAt = &parsed
	}
	tokenName := fmt.Sprintf("ig-sub-%d-%d", req.IGUserID, subscriptionID)
	token, err := model.UpsertIgToken(user.Id, subscriptionID, strings.TrimSpace(req.Group), expiresAt, req.RotateKey, tokenName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"token_id":   token.Id,
			"key":        formatInternalTokenKey(token.Key),
			"group":      token.Group,
			"expires_at": formatInternalExpiry(token.ExpiredTime),
		},
	})
}

func InternalRevokeIgToken(c *gin.Context) {
	subscriptionID, err := strconv.ParseInt(c.Param("subscription_id"), 10, 64)
	if err != nil || subscriptionID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid subscription_id"})
		return
	}
	updated, err := model.DisableTokenByIgSubscriptionID(subscriptionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"updated": updated,
		},
	})
}

func InternalListModels(c *gin.Context) {
	models, err := model.GetModelsForInternal()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    models,
	})
}

func formatInternalTokenKey(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "sk-") {
		return raw
	}
	return "sk-" + raw
}

func formatInternalExpiry(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}
