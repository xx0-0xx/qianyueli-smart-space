package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	elysModel "smart-space-management/backend/internal/model/elys"
	"smart-space-management/backend/internal/model/payment"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/secretbox"
	"smart-space-management/backend/pkg/utils"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

const (
	elysPreparePrefix = "ssm:elys:prepare:"
	elysPrepareTTL    = 5 * time.Minute
)

const getDelScript = `
local v = redis.call("get", KEYS[1])
if v then redis.call("del", KEYS[1]) end
return v
`

type ElysAPI interface {
	AuthURL(ctx context.Context, shopID uint64, platform int) (string, error)
	Prepare(ctx context.Context, shopID uint64, platform int, code string) (*ElysPrepareResult, error)
	Consume(ctx context.Context, shopID uint64, platform int, code, ticketInfo string, num int) (*ElysConsumeResult, error)
}

type ElysVoucherService struct {
	DB      *gorm.DB
	Redis   *redis.Client
	Balance *BalanceService
	Client  ElysAPI
	Box     *secretbox.Box
	Enabled bool
}

func NewElysVoucherService(db *gorm.DB, rdb *redis.Client, balance *BalanceService, client ElysAPI, box *secretbox.Box, enabled bool) *ElysVoucherService {
	return &ElysVoucherService{DB: db, Redis: rdb, Balance: balance, Client: client, Box: box, Enabled: enabled}
}

type ElysPrepareView struct {
	PrepareToken    string `json:"prepare_token"`
	Platform        string `json:"platform"`
	TicketName      string `json:"ticket_name"`
	PayAmount       int64  `json:"pay_amount"`
	ActualPayAmount int64  `json:"actual_pay_amount"`
	ExpiresIn       int    `json:"expires_in"`
}

type prepareSession struct {
	UserID          uint64 `json:"user_id"`
	StoreID         uint64 `json:"store_id"`
	ShopID          uint64 `json:"shop_id"`
	Platform        string `json:"platform"`
	Code            string `json:"code"`
	TicketInfo      string `json:"ticket_info"`
	TicketName      string `json:"ticket_name"`
	DealID          string `json:"deal_id"`
	PayAmount       int64  `json:"pay_amount"`
	ActualPayAmount int64  `json:"actual_pay_amount"`
	ExpiresAt       int64  `json:"expires_at"`
}

type ElysConsumeView struct {
	Amount      int64  `json:"amount"`
	NewBalance  int64  `json:"new_balance"`
	Status      string `json:"status"`
	VoucherCode string `json:"voucher_code"`
}

func (s *ElysVoucherService) Prepare(ctx context.Context, userID, storeID uint64, platform, code string) (ElysPrepareView, error) {
	if !s.Enabled {
		return ElysPrepareView{}, errcode.ErrElysDisabled.New("团购核销未启用")
	}
	binding, err := s.binding(storeID)
	if err != nil {
		return ElysPrepareView{}, err
	}
	platformCode := elysPlatformCode(platform)
	if platformCode == 0 {
		return ElysPrepareView{}, errcode.ErrBadRequest.New("平台不合法")
	}
	if platform == "meituan" && !binding.MeituanAuthorized {
		return ElysPrepareView{}, errcode.ErrElysPlatformUnauthorized.New("美团未授权")
	}
	if platform == "douyin" && !binding.DouyinAuthorized {
		return ElysPrepareView{}, errcode.ErrElysPlatformUnauthorized.New("抖音未授权")
	}

	res, err := s.Client.Prepare(ctx, binding.ElysShopID, platformCode, code)
	if err != nil {
		return ElysPrepareView{}, mapElysError(err, errcode.ErrElysPrepareFailed)
	}

	token, err := randomToken()
	if err != nil {
		return ElysPrepareView{}, err
	}
	actual := int64(0)
	if res.ActualPayAmount != nil {
		actual = *res.ActualPayAmount
	}
	session := prepareSession{
		UserID: userID, StoreID: storeID, ShopID: binding.ElysShopID,
		Platform: platform, Code: code, TicketInfo: res.TicketInfo,
		TicketName: res.TicketName, DealID: res.DealID,
		PayAmount: res.PayAmount, ActualPayAmount: actual,
		ExpiresAt: time.Now().Add(elysPrepareTTL).Unix(),
	}
	raw, _ := json.Marshal(session)
	if err := s.Redis.Set(ctx, elysPreparePrefix+token, raw, elysPrepareTTL).Err(); err != nil {
		return ElysPrepareView{}, err
	}
	return ElysPrepareView{
		PrepareToken: token, Platform: platform, TicketName: res.TicketName,
		PayAmount: res.PayAmount, ActualPayAmount: actual,
		ExpiresIn: int(elysPrepareTTL.Seconds()),
	}, nil
}

func (s *ElysVoucherService) Consume(ctx context.Context, userID uint64, prepareToken string) (ElysConsumeView, error) {
	if !s.Enabled {
		return ElysConsumeView{}, errcode.ErrElysDisabled.New("团购核销未启用")
	}
	key := elysPreparePrefix + prepareToken
	rawAny, err := s.Redis.Eval(ctx, getDelScript, []string{key}).Result()
	if err == redis.Nil || rawAny == nil {
		return ElysConsumeView{}, errcode.ErrBadRequest.New("确认已过期，请重新扫码")
	}
	if err != nil {
		return ElysConsumeView{}, err
	}
	raw, ok := rawAny.(string)
	if !ok {
		return ElysConsumeView{}, errcode.ErrBadRequest.New("确认已过期，请重新扫码")
	}
	var session prepareSession
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return ElysConsumeView{}, err
	}
	if session.UserID != userID {
		return ElysConsumeView{}, errcode.ErrPermissionDenied.New("无权操作该会员码")
	}
	if time.Now().Unix() > session.ExpiresAt {
		return ElysConsumeView{}, errcode.ErrBadRequest.New("确认已过期，请重新扫码")
	}

	lockKey := "ssm:elys:lock:" + session.Platform + ":" + hashText(session.Code)
	ok, err = s.Redis.SetNX(ctx, lockKey, "1", 60*time.Second).Result()
	if err != nil {
		return ElysConsumeView{}, err
	}
	if !ok {
		return ElysConsumeView{}, errcode.ErrElysPendingCredit.New("该券正在处理中，请稍后")
	}
	defer s.Redis.Del(context.Background(), lockKey)

	var existing elysModel.VoucherVerification
	err = s.DB.Where("platform = ? AND voucher_code = ? AND status IN ?",
		session.Platform, session.Code,
		[]int{elysModel.VerificationConsuming, elysModel.VerificationConsumed, elysModel.VerificationCredited}).
		Order("id DESC").First(&existing).Error
	if err == nil {
		if existing.Status == elysModel.VerificationCredited {
			return ElysConsumeView{}, errcode.ErrElysAlreadyConsumed.New("该券已核销，请勿重复充值")
		}
		return ElysConsumeView{}, errcode.ErrElysPendingCredit.New("该券正在处理中，请稍后")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return ElysConsumeView{}, err
	}

	rec := elysModel.VoucherVerification{
		StoreID: session.StoreID, ElysShopID: session.ShopID,
		Platform: session.Platform, VoucherCode: session.Code,
		TicketName: session.TicketName, DealID: session.DealID,
		PayAmount: session.PayAmount, ActualPayAmount: session.ActualPayAmount,
		UserID: session.UserID, Status: elysModel.VerificationConsuming,
		PrepareToken: prepareToken,
	}
	rec.ID = utils.ShortID()
	if err := s.DB.Create(&rec).Error; err != nil {
		if isDuplicateKey(err) {
			return ElysConsumeView{}, errcode.ErrElysPendingCredit.New("该券正在处理中，请稍后")
		}
		return ElysConsumeView{}, err
	}

	platformCode := elysPlatformCode(session.Platform)
	consume, err := s.Client.Consume(ctx, session.ShopID, platformCode, session.Code, session.TicketInfo, 1)
	if err != nil {
		var elysErr *ElysError
		if errors.As(err, &elysErr) {
			switch {
			case elysErr.Code == 10500 || elysErr.ErrorCode == "ALREADY_CONSUMED":
				s.markStatus(rec.ID, elysModel.VerificationDuplicate, elysErr.ErrorCode, elysErr.Message)
				return ElysConsumeView{}, errcode.ErrElysAlreadyConsumed.New("该券已核销，请勿重复充值")
			case elysErr.ErrorCode == "RESULT_UNKNOWN":
				s.markStatus(rec.ID, elysModel.VerificationUnknown, elysErr.ErrorCode, elysErr.Message)
				return ElysConsumeView{}, errcode.ErrElysResultUnknown.New("核销结果待核实，请稍后查看")
			case elysErr.ErrorCode == "PLATFORM_UNAUTHORIZED":
				s.markStatus(rec.ID, elysModel.VerificationFailed, elysErr.ErrorCode, elysErr.Message)
				return ElysConsumeView{}, errcode.ErrElysPlatformUnauthorized.New("该门店尚未完成平台授权")
			case elysErr.ErrorCode == "QUOTA_EXHAUSTED":
				s.markStatus(rec.ID, elysModel.VerificationFailed, elysErr.ErrorCode, elysErr.Message)
				return ElysConsumeView{}, errcode.ErrElysQuotaExhausted.New("核销额度不足，请联系管理员")
			default:
				s.markStatus(rec.ID, elysModel.VerificationFailed, elysErr.ErrorCode, elysErr.Message)
				return ElysConsumeView{}, errcode.ErrElysConsumeFailed.New(elysErr.Message)
			}
		}
		s.markStatus(rec.ID, elysModel.VerificationUnknown, "NETWORK", err.Error())
		return ElysConsumeView{}, errcode.ErrElysResultUnknown.New("核销结果待核实，请稍后查看")
	}

	amount := sumPayAmount(consume.Data)
	if amount <= 0 {
		s.markStatus(rec.ID, elysModel.VerificationFailed, "EMPTY_AMOUNT", "核销返回金额为空")
		return ElysConsumeView{}, errcode.ErrElysConsumeFailed.New("核销返回金额为空")
	}
	credentialRaw, _ := json.Marshal(consume.ConsumeCredential)
	credentialEnc, err := s.Box.Encrypt(string(credentialRaw))
	if err != nil {
		s.markStatus(rec.ID, elysModel.VerificationFailed, "ENCRYPT", err.Error())
		return ElysConsumeView{}, err
	}
	ticketEnc, err := s.Box.Encrypt(session.TicketInfo)
	if err != nil {
		s.markStatus(rec.ID, elysModel.VerificationFailed, "ENCRYPT", err.Error())
		return ElysConsumeView{}, err
	}
	now := time.Now()
	orderID := ""
	receiptCode := ""
	if len(consume.Data) > 0 {
		orderID = consume.Data[0].OrderID
		receiptCode = consume.Data[0].ReceiptCode
	}
	var wxdg *string
	if consume.WxdgOrderNo != "" {
		wxdg = &consume.WxdgOrderNo
	}
	if err := s.DB.Model(&elysModel.VoucherVerification{}).Where("id = ?", rec.ID).Updates(map[string]interface{}{
		"status":                 elysModel.VerificationConsumed,
		"wxdg_order_no":          wxdg,
		"platform_order_id":      orderID,
		"receipt_code":           receiptCode,
		"pay_amount":             amount,
		"consume_credential_enc": credentialEnc,
		"ticket_info_enc":        ticketEnc,
		"trace_id":               consume.TraceID,
		"consumed_at":            now,
	}).Error; err != nil {
		return ElysConsumeView{}, err
	}

	newBalance, err := s.creditOne(ctx, rec.ID)
	if err != nil {
		s.markStatus(rec.ID, elysModel.VerificationPendingCredit, "CREDIT", err.Error())
		return ElysConsumeView{Amount: amount, Status: "pending_credit", VoucherCode: receiptCode}, nil
	}
	return ElysConsumeView{Amount: amount, NewBalance: newBalance, Status: "credited", VoucherCode: receiptCode}, nil
}

func (s *ElysVoucherService) creditOne(ctx context.Context, id uint64) (int64, error) {
	var newBalance int64
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&elysModel.VoucherVerification{}).
			Where("id = ? AND status IN ?", id, []int{elysModel.VerificationConsumed, elysModel.VerificationPendingCredit}).
			Updates(map[string]interface{}{"status": elysModel.VerificationCredited, "credited_at": time.Now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		var rec elysModel.VoucherVerification
		if err := tx.First(&rec, id).Error; err != nil {
			return err
		}
		bal, gid, err := s.Balance.Credit(tx, rec.UserID, rec.StoreID, rec.PayAmount)
		if err != nil {
			return err
		}
		log := payment.BalanceLog{
			UserID: rec.UserID, Amount: rec.PayAmount, Balance: bal,
			Type: "elys_voucher", RefID: rec.PrepareToken,
			Desc:    platformLabel(rec.Platform) + "团购券充值",
			StoreID: rec.StoreID, GroupID: gid,
		}
		log.ID = utils.ShortID()
		if err := tx.Create(&log).Error; err != nil {
			return err
		}
		newBalance = bal
		return nil
	})
	return newBalance, err
}

func (s *ElysVoucherService) RetryPendingCredits(ctx context.Context) {
	var rows []elysModel.VoucherVerification
	s.DB.Where("status = ?", elysModel.VerificationPendingCredit).Limit(100).Find(&rows)
	for _, row := range rows {
		_, _ = s.creditOne(ctx, row.ID)
	}
}

func (s *ElysVoucherService) markStatus(id uint64, status int, code, msg string) {
	s.DB.Model(&elysModel.VoucherVerification{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": status, "error_code": code, "error_message": msg,
	})
}

func (s *ElysVoucherService) binding(storeID uint64) (*elysModel.StoreBinding, error) {
	var row elysModel.StoreBinding
	err := s.DB.Where("store_id = ? AND status = ?", storeID, elysModel.BindingEnabled).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrElysStoreNotBound.New("该门店暂未接入团购核销")
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashText(v string) string {
	h := sha256.Sum256([]byte(v))
	return hex.EncodeToString(h[:])
}

func sumPayAmount(items []ElysConsumeItem) int64 {
	var total int64
	for _, item := range items {
		total += item.PayAmount
	}
	return total
}

func elysPlatformCode(platform string) int {
	switch platform {
	case "meituan":
		return 1
	case "douyin":
		return 2
	default:
		return 0
	}
}

func platformLabel(platform string) string {
	if platform == "meituan" {
		return "美团"
	}
	if platform == "douyin" {
		return "抖音"
	}
	return platform
}

func mapElysError(err error, fallback errcode.Code) error {
	var e *ElysError
	if errors.As(err, &e) {
		switch e.ErrorCode {
		case "PLATFORM_UNAUTHORIZED":
			return errcode.ErrElysPlatformUnauthorized.New("该门店尚未完成平台授权")
		case "QUOTA_EXHAUSTED":
			return errcode.ErrElysQuotaExhausted.New("核销额度不足，请联系管理员")
		case "ALREADY_CONSUMED":
			return errcode.ErrElysAlreadyConsumed.New("该券已核销，请勿重复充值")
		}
		return fallback.New(e.Message)
	}
	return fallback.New("验券失败")
}

func isDuplicateKey(err error) bool {
	return strings.Contains(err.Error(), "Duplicate entry")
}

type ElysVerificationFilter struct {
	StoreID  uint64
	Platform string
	Status   int
	Page     int
	PageSize int
}

func (s *ElysVoucherService) ListVerifications(f ElysVerificationFilter) ([]elysModel.VoucherVerification, int64, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	query := func() *gorm.DB {
		q := s.DB.Model(&elysModel.VoucherVerification{})
		if f.StoreID > 0 {
			q = q.Where("store_id = ?", f.StoreID)
		}
		if f.Platform == "meituan" || f.Platform == "douyin" {
			q = q.Where("platform = ?", f.Platform)
		}
		if f.Status > 0 {
			q = q.Where("status = ?", f.Status)
		}
		return q
	}
	var total int64
	if err := query().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []elysModel.VoucherVerification
	err := query().Order("id DESC").
		Offset((f.Page - 1) * f.PageSize).
		Limit(f.PageSize).
		Find(&rows).Error
	return rows, total, err
}

func (s *ElysVoucherService) RetryCreditByID(ctx context.Context, id uint64) error {
	var rec elysModel.VoucherVerification
	if err := s.DB.First(&rec, id).Error; err != nil {
		return err
	}
	if rec.Status != elysModel.VerificationConsumed && rec.Status != elysModel.VerificationPendingCredit {
		return errcode.ErrBadRequest.New("当前状态不可重试入账")
	}
	_, err := s.creditOne(ctx, id)
	return err
}

func (s *ElysVoucherService) ListUserVerifications(userID uint64, page, size int) ([]elysModel.VoucherVerification, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}
	var rows []elysModel.VoucherVerification
	err := s.DB.Where("user_id = ? AND status IN ?", userID,
		[]int{elysModel.VerificationConsumed, elysModel.VerificationCredited, elysModel.VerificationPendingCredit}).
		Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, err
}
