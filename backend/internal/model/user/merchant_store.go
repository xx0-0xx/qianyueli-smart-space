package user

import "smart-space-management/backend/internal/model/base"

// MerchantStore 商家-门店关联（多对多，一个商家可管理多个门店）
type MerchantStore struct {
	base.Model
	MerchantID uint64 `gorm:"column:merchant_id;index;not null" json:"merchant_id"`
	StoreID    uint64 `gorm:"column:store_id;index;not null" json:"store_id"`
}

func (MerchantStore) TableName() string { return "merchant_stores" }
