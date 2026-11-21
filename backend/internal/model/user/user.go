package user

import (
	"smart-space-management/backend/internal/model/base"
)

// UserRole 用户角色
type UserRole string

const (
	RoleUser       UserRole = "user"
	RoleMerchant   UserRole = "merchant"
	RoleAdmin      UserRole = "admin"
	RoleSuperAdmin UserRole = "super_admin"
)

// UserStatus 用户状态
type UserStatus int

const (
	StatusNormal  UserStatus = 1
	StatusDisable UserStatus = 2
)

// User 用户模型
type User struct {
	base.Model
	OpenID       string     `gorm:"column:open_id;type:varchar(128);uniqueIndex;not null" json:"open_id"`
	UnionID      string     `gorm:"column:union_id;type:varchar(64);index" json:"union_id"`
	Nickname     string     `gorm:"column:nickname;type:varchar(64)" json:"nickname"`
	Avatar       string     `gorm:"column:avatar;type:varchar(512)" json:"avatar"`
	Phone        string     `gorm:"column:phone;type:varchar(20);index" json:"phone"`
	PasswordHash string     `gorm:"column:password_hash;type:varchar(256)" json:"-"` // bcrypt 哈希，仅 admin/merchant 使用
	Role         UserRole   `gorm:"column:role;type:varchar(20);not null;default:'user'" json:"role"`
	Status       UserStatus `gorm:"column:status;type:tinyint;not null;default:1" json:"status"`
	StoreID      uint64     `gorm:"column:store_id;index" json:"store_id"` // 商家绑定的门店
	Balance      int64      `gorm:"column:balance;not null;default:0" json:"balance"` // 账户余额（分）
	LastLogin    int64      `gorm:"column:last_login;autoUpdateTime:milli" json:"last_login"`
}

func (User) TableName() string { return "users" }

// UserAddress 用户常用地址 (预留)
type UserAddress struct {
	base.Model
	UserID   uint64 `gorm:"column:user_id;index;not null" json:"user_id"`
	Province string `gorm:"column:province;type:varchar(32)" json:"province"`
	City     string `gorm:"column:city;type:varchar(32)" json:"city"`
	District string `gorm:"column:district;type:varchar(32)" json:"district"`
	Detail   string `gorm:"column:detail;type:varchar(255)" json:"detail"`
}

func (UserAddress) TableName() string { return "user_addresses" }
