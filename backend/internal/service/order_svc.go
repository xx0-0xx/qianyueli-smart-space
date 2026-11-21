package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"smart-space-management/backend/internal/model/order"
	"smart-space-management/backend/internal/model/payment"
	"smart-space-management/backend/internal/model/resource"
	"smart-space-management/backend/pkg/utils"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// OrderService 订单业务逻辑
type OrderService struct {
	DB    *gorm.DB
	Redis *redis.Client
}

// NewOrderService 创建订单服务
func NewOrderService(db *gorm.DB, rdb *redis.Client) *OrderService {
	return &OrderService{DB: db, Redis: rdb}
}

// CreateOrderParams 创建订单参数
type CreateOrderParams struct {
	UserID     uint64
	ResourceID uint64
	StoreID    uint64
	StartTime  int64 // unix ms
	EndTime    int64 // unix ms
	LockTTL    time.Duration
	PayMethod  string // wechat=微信(默认) balance=余额
}

// CreateOrderResult 创建订单结果
type CreateOrderResult struct {
	Order       *order.Order
	OrderNo     string
	TotalAmount int
}

// CreateOrder 创建订单核心逻辑
func (s *OrderService) CreateOrder(ctx context.Context, params CreateOrderParams) (*CreateOrderResult, error) {
	now := time.Now().UnixMilli()

	// 校验时间：允许当前时段（开始时间不早于当前时间 5 分钟）
	if params.StartTime < now-5*60*1000 {
		return nil, fmt.Errorf("预约时间必须晚于当前时间")
	}
	if params.EndTime <= params.StartTime {
		return nil, fmt.Errorf("结束时间必须晚于开始时间")
	}

	durationMinutes := int((params.EndTime - params.StartTime) / 60000)

	// ── Redis 分布式锁 ──
	lockKey := fmt.Sprintf("ssm:resource:lock:%d", params.ResourceID)
	lockVal := fmt.Sprintf("%d:%d", params.UserID, now)

	ok, err := s.Redis.SetNX(ctx, lockKey, lockVal, params.LockTTL).Result()
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}
	if !ok {
		return nil, fmt.Errorf("resource locked")
	}

	// 锁释放函数（调用方负责 defer）
	deferFunc := func() {
		script := `
			if redis.call("GET", KEYS[1]) == ARGV[1] then
				return redis.call("DEL", KEYS[1])
			else
				return 0
			end
		`
		s.Redis.Eval(ctx, script, []string{lockKey}, lockVal)
	}
	defer deferFunc()

	// ── 校验资源状态 ──
	var res resource.Resource
	if err := s.DB.First(&res, params.ResourceID).Error; err != nil {
		return nil, fmt.Errorf("resource not found")
	}
	if res.Status == resource.StatusMaint {
		return nil, fmt.Errorf("resource in maintenance")
	}
	if res.StoreID != params.StoreID {
		return nil, fmt.Errorf("资源不属于该门店")
	}

	// ── 时间冲突校验 ──
	var conflictCount int64
	s.DB.Model(&order.Order{}).
		Where("resource_id = ? AND status IN (0,1,2) AND start_time < ? AND end_time > ?",
			params.ResourceID, params.EndTime, params.StartTime).
		Count(&conflictCount)
	if conflictCount > 0 {
		return nil, fmt.Errorf("order time conflict")
	}

	// ── 查询价格 ──
	unitPrice := s.getUnitPrice(params.StoreID, res.ResourceTypeID)
	timeUnit := s.getTimeUnit(params.StoreID, res.ResourceTypeID)
	if timeUnit <= 0 {
		timeUnit = 60
	}
	totalAmount := unitPrice * durationMinutes / timeUnit
	if totalAmount <= 0 {
		totalAmount = unitPrice
	}

	// ── 会员折扣 ──
	originalAmount := totalAmount
	memberDiscount := 0
	var memberLevelID *uint64
	var memberLevelName string
	active, err := FindActiveMember(s.DB, params.StoreID, params.UserID, now)
	if err != nil {
		log.Printf("[ORDER] find active member failed store=%d user=%d: %v", params.StoreID, params.UserID, err)
		return nil, fmt.Errorf("会员信息读取失败，请稍后重试")
	}
	if active != nil {
		totalAmount, memberDiscount = ApplyMemberDiscount(originalAmount, active.Level.EffectiveDiscountBP())
		id := active.Level.ID
		memberLevelID = &id
		memberLevelName = active.Level.Name
	}

	// ── 余额支付 ──
	payByBalance := params.PayMethod == "balance"

	// ── 创建订单 ──
	orderNo := fmt.Sprintf("SS%d%04d", time.Now().UnixMilli(), utils.ShortID()%10000)
	status := order.StatusPendingPay
	var paidAt *int64
	if payByBalance {
		status = order.StatusPaid
		nowMs := time.Now().UnixMilli()
		paidAt = &nowMs
	}
	o := order.Order{
		OrderNo:         orderNo,
		UserID:          params.UserID,
		StoreID:         params.StoreID,
		ResourceID:      params.ResourceID,
		ResourceName:    res.Name,
		StartTime:       params.StartTime,
		EndTime:         params.EndTime,
		Duration:        durationMinutes,
		UnitPrice:       unitPrice,
		TotalAmount:     totalAmount,
		OriginalAmount:  originalAmount,
		MemberDiscount:  memberDiscount,
		MemberLevelID:   memberLevelID,
		MemberLevelName: memberLevelName,
		Status:          status,
		PaidAt:          paidAt,
	}
	o.ID = utils.ShortID()

	tx := s.DB.Begin()
	if err := tx.Create(&o).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("create order failed")
	}
	// 余额扣款 + 流水（同一事务）
	if payByBalance {
		newBal, gid, derr := (&BalanceService{DB: s.DB}).Debit(tx, params.UserID, params.StoreID, int64(totalAmount))
		if derr != nil {
			tx.Rollback()
			if errors.Is(derr, ErrInsufficientBalance) {
				return nil, fmt.Errorf("余额不足")
			}
			return nil, fmt.Errorf("扣款失败")
		}
		balLog := payment.BalanceLog{
			UserID: params.UserID, Amount: -int64(totalAmount), Balance: newBal,
			Type: "order", RefID: orderNo, Desc: "预约订单扣款",
			StoreID: params.StoreID, GroupID: gid,
		}
		balLog.ID = utils.ShortID()
		if err := tx.Create(&balLog).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("记录余额流水失败")
		}
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("提交订单失败")
	}

	return &CreateOrderResult{
		Order:       &o,
		OrderNo:     o.OrderNo,
		TotalAmount: o.TotalAmount,
	}, nil
}

func (s *OrderService) getUnitPrice(storeID, resourceTypeID uint64) int {
	return BasePrice(s.DB, storeID, resourceTypeID)
}

// getTimeUnit 获取计费单位 (分钟)
func (s *OrderService) getTimeUnit(storeID, resourceTypeID uint64) int {
	var result struct{ TimeUnit int }
	s.DB.Table("price_rules").
		Select("time_unit").
		Where("store_id = ? AND resource_type_id IN ? AND status = 1", storeID, []uint64{resourceTypeID, 0}).
		Order("resource_type_id DESC, is_default DESC").
		Limit(1).
		Scan(&result)
	if result.TimeUnit > 0 {
		return result.TimeUnit
	}
	return 60
}
