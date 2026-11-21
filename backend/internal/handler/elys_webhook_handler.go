package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	elysModel "smart-space-management/backend/internal/model/elys"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type ElysWebhookHandler struct {
	ctx *Context
}

func NewElysWebhookHandler(ctx *Context) *ElysWebhookHandler {
	return &ElysWebhookHandler{ctx: ctx}
}

var elysRequestIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

func (h *ElysWebhookHandler) Handle(c *gin.Context) {
	requestID := c.GetHeader("X-Request-Id")
	if !validElysRequestID(requestID) {
		c.Status(http.StatusBadRequest)
		return
	}
	c.Header("X-Request-Id", requestID)

	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	ts := c.GetHeader("X-Event-Timestamp")
	sig := c.GetHeader("X-Event-Signature")
	if !verifyElysSignature(h.ctx.Cfg.Elys.APIKey, ts, sig, raw) {
		c.Status(http.StatusUnauthorized)
		return
	}

	var env struct {
		EventID string          `json:"eventId"`
		Event   string          `json:"event"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if env.EventID == "" || c.GetHeader("X-Event-Id") != env.EventID {
		c.Status(http.StatusBadRequest)
		return
	}

	if env.Event == "webhook.endpoint.verification" {
		var data struct {
			Challenge string `json:"challenge"`
		}
		if err := json.Unmarshal(env.Data, &data); err != nil ||
			data.Challenge == "" || !strings.HasPrefix(env.EventID, "evt_verify_") {
			c.Status(http.StatusBadRequest)
			return
		}
		c.JSON(http.StatusOK, gin.H{"challenge": data.Challenge})
		return
	}

	if env.Event == "merchant.authorization.succeeded" {
		var data struct {
			Platform         string `json:"platform"`
			BusinessCategory string `json:"businessCategory"`
			ShopID           uint64 `json:"shopId"`
			Status           string `json:"status"`
		}
		if err := json.Unmarshal(env.Data, &data); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		if data.BusinessCategory != "service_retail" ||
			(data.Platform != "meituan" && data.Platform != "douyin") ||
			data.ShopID == 0 {
			c.Status(http.StatusBadRequest)
			return
		}

		now := time.Now()
		err := h.ctx.DB.Transaction(func(tx *gorm.DB) error {
			event := elysModel.WebhookEvent{
				EventID:     env.EventID,
				Event:       env.Event,
				Platform:    data.Platform,
				ShopID:      data.ShopID,
				Payload:     string(raw),
				ProcessedAt: &now,
			}
			event.ID = utils.ShortID()
			if err := tx.Create(&event).Error; err != nil {
				return err
			}
			updates := map[string]interface{}{"last_auth_at": now}
			if data.Platform == "meituan" {
				updates["meituan_authorized"] = true
			} else {
				updates["douyin_authorized"] = true
			}
			return tx.Model(&elysModel.StoreBinding{}).
				Where("elys_shop_id = ?", data.ShopID).
				Updates(updates).Error
		})
		if err != nil && !isDuplicateDBError(err) {
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	c.Status(http.StatusNoContent)
}

func isDuplicateDBError(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

func validElysRequestID(v string) bool {
	return elysRequestIDPattern.MatchString(v)
}

func verifyElysSignature(apiKey, timestamp, signature string, raw []byte) bool {
	if apiKey == "" || timestamp == "" || signature == "" {
		return false
	}
	sec, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	now := time.Now().Unix()
	if sec < now-300 || sec > now+300 {
		return false
	}
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte(timestamp + "."))
	mac.Write(raw)
	expected := "v1=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
