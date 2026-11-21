package service

import (
	"context"
	"errors"
	"time"

	"smart-space-management/backend/internal/model/elys"
	storeModel "smart-space-management/backend/internal/model/store"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/utils"

	"gorm.io/gorm"
)

type ElysBindingService struct {
	DB     *gorm.DB
	Client *ElysClient
}

func NewElysBindingService(db *gorm.DB, client *ElysClient) *ElysBindingService {
	return &ElysBindingService{DB: db, Client: client}
}

type ElysBindingView struct {
	ID                uint64     `json:"id,string"`
	StoreID           uint64     `json:"store_id,string"`
	StoreName         string     `json:"store_name"`
	ElysShopID        uint64     `json:"elys_shop_id"`
	ElysMasterShopID  uint64     `json:"elys_master_shop_id"`
	MeituanAuthorized bool       `json:"meituan_authorized"`
	DouyinAuthorized  bool       `json:"douyin_authorized"`
	LastAuthAt        *time.Time `json:"last_auth_at"`
	Status            int        `json:"status"`
	LastError         string     `json:"last_error"`
}

func (s *ElysBindingService) List() ([]ElysBindingView, error) {
	var rows []ElysBindingView
	err := s.DB.Table("elys_store_bindings AS b").
		Joins("LEFT JOIN stores AS s ON s.id = b.store_id").
		Select("b.id, b.store_id, COALESCE(s.name,'') AS store_name, b.elys_shop_id, b.elys_master_shop_id, b.meituan_authorized, b.douyin_authorized, b.last_auth_at, b.status, b.last_error").
		Where("b.deleted_at IS NULL").
		Order("b.id DESC").
		Scan(&rows).Error
	return rows, err
}

func (s *ElysBindingService) Get(storeID uint64) (*elys.StoreBinding, error) {
	var row elys.StoreBinding
	err := s.DB.Where("store_id = ?", storeID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrElysStoreNotBound.New("该门店暂未接入团购核销")
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *ElysBindingService) Upsert(storeID, shopID, masterShopID uint64) error {
	if storeID == 0 || shopID == 0 {
		return errcode.ErrBadRequest.New("门店 ID 和 Elys shopId 必填")
	}
	var storeCount int64
	if err := s.DB.Model(&storeModel.Store{}).Where("id = ?", storeID).Count(&storeCount).Error; err != nil {
		return err
	}
	if storeCount == 0 {
		return errcode.ErrStoreNotFound.New("门店不存在")
	}
	var row elys.StoreBinding
	err := s.DB.Where("store_id = ?", storeID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = elys.StoreBinding{
			StoreID:          storeID,
			ElysShopID:       shopID,
			ElysMasterShopID: masterShopID,
			Status:           elys.BindingEnabled,
		}
		row.ID = utils.ShortID()
		return s.DB.Create(&row).Error
	}
	if err != nil {
		return err
	}
	return s.DB.Model(&row).Updates(map[string]interface{}{
		"elys_shop_id":        shopID,
		"elys_master_shop_id": masterShopID,
	}).Error
}

func (s *ElysBindingService) Delete(storeID uint64) error {
	return s.DB.Where("store_id = ?", storeID).Delete(&elys.StoreBinding{}).Error
}

func (s *ElysBindingService) AuthURL(ctx context.Context, storeID uint64, platform int) (string, error) {
	row, err := s.Get(storeID)
	if err != nil {
		return "", err
	}
	return s.Client.AuthURL(ctx, row.ElysShopID, platform)
}

func (s *ElysBindingService) MarkAuthorized(shopID uint64, platform string, at time.Time) error {
	updates := map[string]interface{}{
		"last_auth_at": at,
	}
	if platform == "meituan" {
		updates["meituan_authorized"] = true
	}
	if platform == "douyin" {
		updates["douyin_authorized"] = true
	}
	return s.DB.Model(&elys.StoreBinding{}).
		Where("elys_shop_id = ?", shopID).
		Updates(updates).Error
}
