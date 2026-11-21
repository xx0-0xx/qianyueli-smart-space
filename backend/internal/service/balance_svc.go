package service

import (
	"errors"
	"fmt"

	"smart-space-management/backend/internal/model/balance"
	"smart-space-management/backend/pkg/utils"

	"gorm.io/gorm"
)

// ErrInsufficientBalance 余额不足
var ErrInsufficientBalance = errors.New("insufficient balance")

// GroupChangeAction 门店改组的动作
type GroupChangeAction int

const (
	GroupChangeNone               GroupChangeAction = iota // 目标与当前相同
	GroupChangeRejectSplitBalance                          // 拆分且源组有余额 → 拒绝
	GroupChangeMoveAccounts                                // 源组仅此门店 → 迁移(求和)账户
	GroupChangeLeaveEmpty                                  // 源组有其他门店且无余额 → 仅改归属
)

// DecideGroupChange 纯决策函数（可单测）
func DecideGroupChange(sameGroup, srcHasOtherStores, srcGroupBalancePositive bool) (GroupChangeAction, error) {
	if sameGroup {
		return GroupChangeNone, nil
	}
	if srcHasOtherStores {
		if srcGroupBalancePositive {
			return GroupChangeRejectSplitBalance, fmt.Errorf("该门店所在共享组仍有用户余额，请先处理余额再拆组")
		}
		return GroupChangeLeaveEmpty, nil
	}
	return GroupChangeMoveAccounts, nil
}

// BalanceService 余额账户解析与增减
type BalanceService struct{ DB *gorm.DB }

func NewBalanceService(db *gorm.DB) *BalanceService { return &BalanceService{DB: db} }

// ResolveGroupID 门店 → 余额组
func ResolveGroupID(db *gorm.DB, storeID uint64) (uint64, error) {
	if storeID == 0 {
		return 0, fmt.Errorf("store id 为空")
	}
	var row struct{ BalanceGroupID uint64 }
	if err := db.Table("stores").Select("balance_group_id").Where("id = ?", storeID).Scan(&row).Error; err != nil {
		return 0, err
	}
	if row.BalanceGroupID == 0 {
		return 0, fmt.Errorf("门店 %d 未配置余额组", storeID)
	}
	return row.BalanceGroupID, nil
}

func getOrCreateAccountTx(tx *gorm.DB, userID, groupID uint64) (*balance.Account, error) {
	var acc balance.Account
	err := tx.Where("user_id = ? AND group_id = ?", userID, groupID).First(&acc).Error
	if err == nil {
		return &acc, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	acc = balance.Account{UserID: userID, GroupID: groupID, Balance: 0}
	acc.ID = utils.ShortID()
	if err := tx.Create(&acc).Error; err != nil {
		// 并发首次创建可能撞唯一键：回查
		var again balance.Account
		if e2 := tx.Where("user_id = ? AND group_id = ?", userID, groupID).First(&again).Error; e2 == nil {
			return &again, nil
		}
		return nil, err
	}
	return &acc, nil
}

// Credit 加款，返回变动后余额与所属组
func (s *BalanceService) Credit(tx *gorm.DB, userID, storeID uint64, amount int64) (int64, uint64, error) {
	gid, err := ResolveGroupID(tx, storeID)
	if err != nil {
		return 0, 0, err
	}
	acc, err := getOrCreateAccountTx(tx, userID, gid)
	if err != nil {
		return 0, 0, err
	}
	if err := tx.Model(&balance.Account{}).Where("id = ?", acc.ID).
		Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
		return 0, 0, err
	}
	var after balance.Account
	if err := tx.First(&after, acc.ID).Error; err != nil {
		return 0, 0, err
	}
	return after.Balance, gid, nil
}

// Debit 扣款，余额不足返回 ErrInsufficientBalance
func (s *BalanceService) Debit(tx *gorm.DB, userID, storeID uint64, amount int64) (int64, uint64, error) {
	gid, err := ResolveGroupID(tx, storeID)
	if err != nil {
		return 0, 0, err
	}
	res := tx.Model(&balance.Account{}).
		Where("user_id = ? AND group_id = ? AND balance >= ?", userID, gid, amount).
		Update("balance", gorm.Expr("balance - ?", amount))
	if res.Error != nil {
		return 0, 0, res.Error
	}
	if res.RowsAffected != 1 {
		return 0, 0, ErrInsufficientBalance
	}
	var after balance.Account
	if err := tx.Where("user_id = ? AND group_id = ?", userID, gid).First(&after).Error; err != nil {
		return 0, 0, err
	}
	return after.Balance, gid, nil
}

// BalanceOf 查询余额（不创建账户）
func (s *BalanceService) BalanceOf(userID, storeID uint64) (int64, uint64, error) {
	gid, err := ResolveGroupID(s.DB, storeID)
	if err != nil {
		return 0, 0, err
	}
	var acc balance.Account
	err = s.DB.Where("user_id = ? AND group_id = ?", userID, gid).First(&acc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, gid, nil
	}
	if err != nil {
		return 0, 0, err
	}
	return acc.Balance, gid, nil
}
