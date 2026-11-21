// Package base 提供所有 GORM 模型的公共字段
package base

import (
	"time"

	"gorm.io/gorm"
)

// Model 内嵌到所有业务模型中
type Model struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
