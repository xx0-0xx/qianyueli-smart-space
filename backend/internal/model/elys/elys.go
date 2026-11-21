package elys

import (
	"time"

	"smart-space-management/backend/internal/model/base"
)

const (
	BindingEnabled  = 1
	BindingDisabled = 0

	VerificationConsuming     = 10
	VerificationConsumed      = 20
	VerificationCredited      = 30
	VerificationPendingCredit = 40
	VerificationFailed        = 50
	VerificationUnknown       = 60
	VerificationDuplicate     = 70
)

type StoreBinding struct {
	base.Model
	StoreID           uint64     `gorm:"column:store_id;not null;uniqueIndex:uk_elys_binding_store" json:"store_id"`
	ElysShopID        uint64     `gorm:"column:elys_shop_id;not null;uniqueIndex:uk_elys_binding_shop" json:"elys_shop_id"`
	ElysMasterShopID  uint64     `gorm:"column:elys_master_shop_id;not null;default:0" json:"elys_master_shop_id"`
	MeituanAuthorized bool       `gorm:"column:meituan_authorized;not null;default:false" json:"meituan_authorized"`
	DouyinAuthorized  bool       `gorm:"column:douyin_authorized;not null;default:false" json:"douyin_authorized"`
	LastAuthAt        *time.Time `gorm:"column:last_auth_at" json:"last_auth_at"`
	Status            int        `gorm:"column:status;not null;default:1" json:"status"`
	LastError         string     `gorm:"column:last_error;type:varchar(255)" json:"last_error"`
}

func (StoreBinding) TableName() string { return "elys_store_bindings" }

type VoucherVerification struct {
	base.Model
	StoreID              uint64     `gorm:"column:store_id;not null" json:"store_id"`
	ElysShopID           uint64     `gorm:"column:elys_shop_id;not null" json:"elys_shop_id"`
	Platform             string     `gorm:"column:platform;type:varchar(16);not null" json:"platform"`
	VoucherCode          string     `gorm:"column:voucher_code;type:varchar(512);not null" json:"voucher_code"`
	TicketName           string     `gorm:"column:ticket_name;type:varchar(255)" json:"ticket_name"`
	DealID               string     `gorm:"column:deal_id;type:varchar(128)" json:"deal_id"`
	PayAmount            int64      `gorm:"column:pay_amount;not null;default:0" json:"pay_amount"`
	ActualPayAmount      int64      `gorm:"column:actual_pay_amount;not null;default:0" json:"actual_pay_amount"`
	WxdgOrderNo          *string    `gorm:"column:wxdg_order_no;type:varchar(128);uniqueIndex:uk_elys_wxdg_order" json:"wxdg_order_no"`
	PlatformOrderID      string     `gorm:"column:platform_order_id;type:varchar(128)" json:"platform_order_id"`
	ReceiptCode          string     `gorm:"column:receipt_code;type:varchar(128)" json:"receipt_code"`
	ConsumeCredentialEnc string     `gorm:"column:consume_credential_enc;type:text" json:"-"`
	TicketInfoEnc        string     `gorm:"column:ticket_info_enc;type:text" json:"-"`
	UserID               uint64     `gorm:"column:user_id;not null" json:"user_id"`
	Status               int        `gorm:"column:status;not null;default:10" json:"status"`
	ErrorCode            string     `gorm:"column:error_code;type:varchar(64)" json:"error_code"`
	ErrorMessage         string     `gorm:"column:error_message;type:varchar(512)" json:"error_message"`
	PrepareToken         string     `gorm:"column:prepare_token;type:varchar(64);not null;uniqueIndex:uk_elys_prepare_token" json:"-"`
	TraceID              string     `gorm:"column:trace_id;type:varchar(128)" json:"trace_id"`
	ConsumedAt           *time.Time `gorm:"column:consumed_at" json:"consumed_at"`
	CreditedAt           *time.Time `gorm:"column:credited_at" json:"credited_at"`
}

func (VoucherVerification) TableName() string { return "elys_voucher_verifications" }

type WebhookEvent struct {
	base.Model
	EventID     string     `gorm:"column:event_id;type:varchar(128);not null;uniqueIndex:uk_elys_event_id" json:"event_id"`
	Event       string     `gorm:"column:event;type:varchar(64);not null" json:"event"`
	Platform    string     `gorm:"column:platform;type:varchar(16)" json:"platform"`
	ShopID      uint64     `gorm:"column:shop_id;not null;default:0" json:"shop_id"`
	Payload     string     `gorm:"column:payload;type:text" json:"payload"`
	ProcessedAt *time.Time `gorm:"column:processed_at" json:"processed_at"`
}

func (WebhookEvent) TableName() string { return "elys_webhook_events" }
