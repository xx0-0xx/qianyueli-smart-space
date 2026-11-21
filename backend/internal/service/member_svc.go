package service

import (
	"errors"
	"log"

	"smart-space-management/backend/internal/model/member"

	"gorm.io/gorm"
)

type ActiveMember struct {
	Member member.StoreMember
	Level  member.Level
}

// FindActiveMember 查询门店下有效的会员和等级；无会员/已过期/等级停用返回 nil, nil。
func FindActiveMember(db *gorm.DB, storeID, userID uint64, nowMs int64) (*ActiveMember, error) {
	var m member.StoreMember
	err := db.Where("store_id = ? AND user_id = ? AND status = 1", storeID, userID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !m.IsActive(nowMs) {
		return nil, nil
	}
	var level member.Level
	if err := db.Where("id = ? AND store_id = ? AND status = 1", m.LevelID, storeID).First(&level).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ActiveMember{Member: m, Level: level}, nil
}

// BasePrice 返回门店房型的基础价格（分），未配置回退 5000。
func BasePrice(db *gorm.DB, storeID, resourceTypeID uint64) int {
	var result struct{ BasePrice int }
	err := db.Table("price_rules").
		Select("base_price").
		Where("store_id = ? AND (resource_type_id = ? OR resource_type_id = 0) AND status = 1 AND deleted_at IS NULL", storeID, resourceTypeID).
		Order("resource_type_id DESC, is_default DESC, id DESC").
		Limit(1).
		Scan(&result).Error
	if err != nil {
		log.Printf("[MEMBER] query base price failed store=%d resource_type=%d: %v", storeID, resourceTypeID, err)
		return 5000
	}
	if result.BasePrice > 0 {
		return result.BasePrice
	}
	return 5000
}
