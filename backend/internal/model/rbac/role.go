package rbac

import (
	"smart-space-management/backend/internal/model/base"
)

// Role 角色定义
type Role struct {
	base.Model
	Name        string `gorm:"column:name;type:varchar(32);not null" json:"name"`
	Code        string `gorm:"column:code;type:varchar(32);uniqueIndex;not null" json:"code"` // super_admin / admin / merchant / user
	Description string `gorm:"column:description;type:varchar(255)" json:"description"`
	StoreID     uint64 `gorm:"column:store_id;index;default:0" json:"store_id"` // 0 = 全局角色
	IsSystem    bool   `gorm:"column:is_system;default:false" json:"is_system"` // 系统内置角色不可删除
}

func (Role) TableName() string { return "roles" }

// Permission 权限定义
type Permission struct {
	base.Model
	Name        string `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Code        string `gorm:"column:code;type:varchar(64);uniqueIndex;not null" json:"code"` // store:read / order:write / device:control
	Resource    string `gorm:"column:resource;type:varchar(128)" json:"resource"`
	Action      string `gorm:"column:action;type:varchar(32)" json:"action"` // read / write / delete / control
	Description string `gorm:"column:description;type:varchar(255)" json:"description"`
}

func (Permission) TableName() string { return "permissions" }

// RolePermission 角色-权限关联
type RolePermission struct {
	base.Model
	RoleID       uint64 `gorm:"column:role_id;index;not null" json:"role_id"`
	PermissionID uint64 `gorm:"column:permission_id;index;not null" json:"permission_id"`
}

func (RolePermission) TableName() string { return "role_permissions" }

// UserRole 用户-角色关联 (一个用户可有多个角色)
type UserRole struct {
	base.Model
	UserID uint64 `gorm:"column:user_id;index;not null" json:"user_id"`
	RoleID uint64 `gorm:"column:role_id;index;not null" json:"role_id"`
}

func (UserRole) TableName() string { return "user_roles" }
