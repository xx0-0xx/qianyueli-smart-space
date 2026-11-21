package service

import (
	"fmt"
	"time"

	"smart-space-management/backend/internal/model/order"
	"smart-space-management/backend/internal/model/resource"
	"smart-space-management/backend/internal/model/store"

	"gorm.io/gorm"
)

// StoreService 门店业务逻辑
type StoreService struct {
	DB *gorm.DB
}

// NewStoreService 创建门店服务
func NewStoreService(db *gorm.DB) *StoreService {
	return &StoreService{DB: db}
}

// TimeSlotResult 时间段查询结果
type TimeSlotResult struct {
	Time       string
	StartMs    int64
	EndMs      int64
	PriceLabel string
	UnitPrice  int
	TimeUnit   int
	Available  int
	Total      int
	Types      []TypeAvailabilityResult
}

// TypeAvailabilityResult 资源类型可用性
type TypeAvailabilityResult struct {
	TypeName  string
	Available int
	Total     int
}

// LoadTypePrices 批量加载门店所有资源类型的专属价格
func (s *StoreService) LoadTypePrices(storeID uint64) map[uint64]int {
	type priceRow struct {
		ResourceTypeID uint64
		BasePrice      int
	}
	var rows []priceRow
	s.DB.Table("price_rules").
		Select("resource_type_id, base_price").
		Where("store_id = ? AND status = 1", storeID).
		Find(&rows)

	prices := make(map[uint64]int, len(rows))
	for _, r := range rows {
		prices[r.ResourceTypeID] = r.BasePrice
	}
	return prices
}

// GetTimeSlots 生成指定日期的营业时间段及可用性
func (s *StoreService) GetTimeSlots(storeID uint64, date time.Time, st store.Store) ([]TimeSlotResult, error) {
	// 解析营业时间
	openH, openM := parseTime(st.OpenTime)
	closeH, closeM := parseTime(st.CloseTime)

	openTs := date.UnixMilli() + int64(openH)*3600000 + int64(openM)*60000
	closeTs := date.UnixMilli() + int64(closeH)*3600000 + int64(closeM)*60000
	if closeTs <= openTs {
		closeTs += 24 * 3600000
	}

	// 加载资源（非维护中）
	var resources []resource.Resource
	s.DB.Where("store_id = ? AND status != ?", storeID, resource.StatusMaint).
		Preload("ResourceType").Find(&resources)

	// 按资源类型分组
	typeGroups := make(map[uint64][]resource.Resource)
	for _, r := range resources {
		typeGroups[r.ResourceTypeID] = append(typeGroups[r.ResourceTypeID], r)
	}

	// 加载当天订单
	dayStart := date.UnixMilli()
	dayEnd := date.Add(24 * time.Hour).UnixMilli()
	var orders []order.Order
	s.DB.Where("store_id = ? AND status IN ? AND start_time < ? AND end_time > ?",
		storeID, []int{1, 2}, dayEnd, dayStart).Find(&orders)

	// 预加载价格规则
	var rule struct {
		BasePrice     int
		TimeUnit      int
		PeakSurcharge string
	}
	s.DB.Table("price_rules").
		Where("store_id = ? AND status = 1", storeID).Order("is_default DESC").Limit(1).
		Scan(&rule)
	if rule.BasePrice <= 0 {
		rule.BasePrice = 5000
	}
	if rule.TimeUnit <= 0 {
		rule.TimeUnit = 30
	}

	// 批量预加载类型价格
	typePrices := s.LoadTypePrices(storeID)

	// 生成 30 分钟槽位
	slotMs := int64(30 * 60000)
	var slots []TimeSlotResult
	for ts := openTs; ts < closeTs; ts += slotMs {
		slotEnd := ts + slotMs

		occupiedIDs := make(map[uint64]bool)
		for _, o := range orders {
			if o.StartTime < slotEnd && o.EndTime > ts {
				occupiedIDs[o.ResourceID] = true
			}
		}

		totalAll, availAll := 0, 0
		var types []TypeAvailabilityResult
		showPrice := rule.BasePrice

		for tid, resList := range typeGroups {
			total := len(resList)
			avail := 0
			for _, r := range resList {
				if !occupiedIDs[r.ID] {
					avail++
				}
			}
			totalAll += total
			availAll += avail

			typeName := fmt.Sprintf("类型%d", tid)
			for _, r := range resList {
				if r.ResourceType != nil {
					typeName = r.ResourceType.Name
					break
				}
			}

			typePrice := rule.BasePrice
			if p, ok := typePrices[tid]; ok && p > 0 {
				typePrice = p
			}

			types = append(types, TypeAvailabilityResult{
				TypeName: typeName, Available: avail, Total: total,
			})

			if showPrice == rule.BasePrice && typePrice != rule.BasePrice {
				showPrice = typePrice
			}
		}

		priceLabel := checkPeak(ts, rule.PeakSurcharge)
		slotHour := (openH + (int(ts-openTs) / 3600000)) % 24
		slotMin := (int(ts-openTs) / 60000) % 60
		timeStr := fmt.Sprintf("%02d:%02d", slotHour, slotMin)
		if openH+(int(ts-openTs)/3600000) >= 24 {
			timeStr += " (次日)"
		}

		slots = append(slots, TimeSlotResult{
			Time:       timeStr,
			StartMs:    ts,
			EndMs:      slotEnd,
			PriceLabel: priceLabel,
			UnitPrice:  showPrice,
			TimeUnit:   rule.TimeUnit,
			Available:  availAll,
			Total:      totalAll,
			Types:      types,
		})
	}

	return slots, nil
}

func parseTime(t string) (h, m int) {
	fmt.Sscanf(t, "%d:%d", &h, &m)
	if h < 0 || h > 24 {
		h = 0
	}
	if m < 0 || m >= 60 {
		m = 0
	}
	return
}

func checkPeak(ts int64, surchargeJSON string) string {
	t := time.UnixMilli(ts)
	hour := t.Hour()
	if (hour >= 8 && hour < 12) || (hour >= 18 && hour < 22) {
		return "高峰"
	}
	if hour >= 12 && hour < 14 {
		return "优惠"
	}
	return "非高峰"
}
