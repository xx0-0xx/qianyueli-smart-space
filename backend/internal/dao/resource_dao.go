package dao

import (
	"smart-space-management/backend/internal/model/resource"

	"gorm.io/gorm"
)

// ResourceDAO 资源数据访问
type ResourceDAO struct {
	db *gorm.DB
}

func NewResourceDAO(db *gorm.DB) *ResourceDAO {
	return &ResourceDAO{db: db}
}

// FindByIDAndStore 按 ID + 门店查询
func (d *ResourceDAO) FindByIDAndStore(id, storeID uint64) (*resource.Resource, error) {
	var r resource.Resource
	err := d.db.Where("id = ? AND store_id = ?", id, storeID).First(&r).Error
	return &r, err
}

// FindFreeByType 查询某资源类型的空闲资源列表
func (d *ResourceDAO) FindFreeByType(storeID, resourceTypeID uint64) ([]resource.Resource, error) {
	var list []resource.Resource
	err := d.db.Where("store_id = ? AND resource_type_id = ? AND status = ?",
		storeID, resourceTypeID, resource.StatusFree).
		Order("sort ASC").Find(&list).Error
	return list, err
}

// UpdateStatus 原子更新资源状态
func (d *ResourceDAO) UpdateStatus(id uint64, status resource.ResourceStatus) error {
	return d.db.Model(&resource.Resource{}).Where("id = ?", id).
		Update("status", status).Error
}

// UpdateStatusCAS CAS-style update: 仅当当前状态为 oldStatus 时更新为 newStatus
func (d *ResourceDAO) UpdateStatusCAS(id uint64, oldStatus, newStatus resource.ResourceStatus) (bool, error) {
	result := d.db.Model(&resource.Resource{}).
		Where("id = ? AND status = ?", id, oldStatus).
		Update("status", newStatus)
	return result.RowsAffected > 0, result.Error
}
