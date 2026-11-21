package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"smart-space-management/backend/internal/model/member"
	"smart-space-management/backend/internal/model/user"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MemberHandler struct {
	ctx *Context
}

func NewMemberHandler(ctx *Context) *MemberHandler {
	return &MemberHandler{ctx: ctx}
}

func validateMemberLevel(name string, discountBP int) error {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 32 {
		return fmt.Errorf("等级名称不能为空且最多32个字")
	}
	if discountBP < 1 || discountBP > 10000 {
		return fmt.Errorf("折扣必须在 0.01 折到 10 折之间")
	}
	return nil
}

func customerCanBeMember(phone string) bool {
	return strings.TrimSpace(phone) != ""
}

func netSpent(total, refund int64) int64 {
	return total - refund
}

type levelDTO struct {
	ID          uint64 `json:"id"`
	StoreID     uint64 `json:"store_id"`
	Name        string `json:"name"`
	DiscountBP  int    `json:"discount_bp"`
	Sort        int    `json:"sort"`
	Status      int    `json:"status"`
	MemberCount int64  `json:"member_count"`
}

func newLevelDTO(level member.Level, memberCount int64) levelDTO {
	return levelDTO{
		ID:          level.ID,
		StoreID:     level.StoreID,
		Name:        level.Name,
		DiscountBP:  level.DiscountBP,
		Sort:        level.Sort,
		Status:      level.Status,
		MemberCount: memberCount,
	}
}

type storeMemberRow struct {
	UserID      uint64
	LevelID     uint64
	LevelName   string
	LevelStatus int
	ExpiresAt   *int64
	Status      int
	Remark      string
}

type customerOrderDTO struct {
	OrderNo      string    `json:"order_no"`
	ResourceName string    `json:"resource_name"`
	Status       int       `json:"status"`
	TotalAmount  int       `json:"total_amount"`
	StartTime    int64     `json:"start_time"`
	EndTime      int64     `json:"end_time"`
	CreatedAt    time.Time `json:"created_at"`
}

type customerRechargeDTO struct {
	ID        uint64    `json:"id"`
	Amount    int64     `json:"amount"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type customerDetailDTO struct {
	ID           uint64                `json:"id"`
	Nickname     string                `json:"nickname"`
	Avatar       string                `json:"avatar"`
	Phone        string                `json:"phone"`
	LevelID      uint64                `json:"level_id"`
	LevelName    string                `json:"level_name"`
	LevelStatus  int                   `json:"level_status"`
	ExpiresAt    *int64                `json:"expires_at"`
	MemberStatus int                   `json:"member_status"`
	Remark       string                `json:"remark"`
	Orders       []customerOrderDTO    `json:"orders"`
	Recharges    []customerRechargeDTO `json:"recharges"`
}

func (h *MemberHandler) ListMemberLevels(c *gin.Context) {
	storeID := GetStoreID(c)
	var levels []member.Level
	if err := h.ctx.DB.Where("store_id = ?", storeID).Order("sort ASC, id ASC").Find(&levels).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	type countRow struct {
		LevelID     uint64
		MemberCount int64
	}
	var counts []countRow
	if err := h.ctx.DB.Model(&member.StoreMember{}).
		Select("level_id, COUNT(*) AS member_count").
		Where("store_id = ? AND status = 1", storeID).
		Group("level_id").
		Scan(&counts).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	countMap := make(map[uint64]int64, len(counts))
	for _, row := range counts {
		countMap[row.LevelID] = row.MemberCount
	}
	rows := make([]levelDTO, 0, len(levels))
	for _, level := range levels {
		rows = append(rows, newLevelDTO(level, countMap[level.ID]))
	}
	response.OK(c, rows)
}

func (h *MemberHandler) CreateMemberLevel(c *gin.Context) {
	storeID := GetStoreID(c)
	var req struct {
		Name       string `json:"name" binding:"required"`
		DiscountBP int    `json:"discount_bp" binding:"required"`
		Sort       int    `json:"sort"`
		Status     *int   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	if err := validateMemberLevel(req.Name, req.DiscountBP); err != nil {
		response.FailMsg(c, errcode.ErrBadRequest, err.Error())
		return
	}
	status := 1
	if req.Status != nil {
		status = *req.Status
	}
	level := member.Level{
		StoreID:    storeID,
		Name:       strings.TrimSpace(req.Name),
		DiscountBP: req.DiscountBP,
		Sort:       req.Sort,
		Status:     status,
	}
	level.ID = utils.ShortID()
	if err := h.ctx.DB.Create(&level).Error; err != nil {
		response.FailMsg(c, errcode.ErrConflict, "等级名称已存在")
		return
	}
	response.Created(c, newLevelDTO(level, 0))
}

func (h *MemberHandler) UpdateMemberLevel(c *gin.Context) {
	storeID := GetStoreID(c)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var level member.Level
	if err := h.ctx.DB.Where("id = ? AND store_id = ?", id, storeID).First(&level).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailCode(c, errcode.ErrNotFound)
			return
		}
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	var req struct {
		Name       string `json:"name"`
		DiscountBP int    `json:"discount_bp"`
		Sort       *int   `json:"sort"`
		Status     *int   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = level.Name
	}
	if req.DiscountBP == 0 {
		req.DiscountBP = level.DiscountBP
	}
	if err := validateMemberLevel(req.Name, req.DiscountBP); err != nil {
		response.FailMsg(c, errcode.ErrBadRequest, err.Error())
		return
	}
	updates := map[string]interface{}{"name": strings.TrimSpace(req.Name), "discount_bp": req.DiscountBP}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if err := h.ctx.DB.Model(&level).Updates(updates).Error; err != nil {
		response.FailMsg(c, errcode.ErrConflict, "更新失败，等级名称可能重复")
		return
	}
	level.Name = strings.TrimSpace(req.Name)
	level.DiscountBP = req.DiscountBP
	if req.Sort != nil {
		level.Sort = *req.Sort
	}
	if req.Status != nil {
		level.Status = *req.Status
	}
	var count int64
	if err := h.ctx.DB.Model(&member.StoreMember{}).
		Where("store_id = ? AND level_id = ? AND status = 1", storeID, level.ID).
		Count(&count).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	response.OK(c, newLevelDTO(level, count))
}

func (h *MemberHandler) DeleteMemberLevel(c *gin.Context) {
	storeID := GetStoreID(c)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var level member.Level
	if err := h.ctx.DB.Where("id = ? AND store_id = ?", id, storeID).First(&level).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailCode(c, errcode.ErrNotFound)
			return
		}
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	var count int64
	if err := h.ctx.DB.Model(&member.StoreMember{}).
		Where("store_id = ? AND level_id = ?", storeID, id).
		Count(&count).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if count > 0 {
		response.FailMsg(c, errcode.ErrBadRequest, "该等级已有会员使用，请先停用或调整会员等级")
		return
	}
	if err := h.ctx.DB.Unscoped().Where("id = ? AND store_id = ?", id, storeID).Delete(&member.Level{}).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	response.OKMsg(c, "已删除")
}

type customerAgg struct {
	UserID      uint64
	OrderCount  int64
	TotalSpent  int64
	LastOrderAt int64
}

func (h *MemberHandler) ListCustomers(c *gin.Context) {
	storeID := GetStoreID(c)
	keyword := strings.TrimSpace(c.Query("keyword"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if size < 1 || size > 100 {
		size = 20
	}

	var ids []uint64
	if err := h.ctx.DB.Raw(`
		SELECT DISTINCT user_id FROM orders
		WHERE store_id = ? AND status IN (1,2,3,5,6) AND deleted_at IS NULL
		UNION
		SELECT DISTINCT user_id FROM recharges WHERE store_id = ? AND status = 1 AND deleted_at IS NULL
		UNION
		SELECT DISTINCT user_id FROM balance_logs WHERE store_id = ? AND deleted_at IS NULL
	`, storeID, storeID, storeID).Scan(&ids).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if len(ids) == 0 {
		response.OK(c, gin.H{"list": []any{}, "total": 0, "page": page, "page_size": size, "total_pages": 0})
		return
	}

	q := h.ctx.DB.Model(&user.User{}).Where("id IN ?", ids)
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("(nickname LIKE ? OR phone LIKE ?)", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}

	type userRow struct {
		ID       uint64
		Nickname string
		Avatar   string
		Phone    string
	}
	var users []userRow
	if err := q.Select("id, nickname, avatar, phone").
		Order("id DESC").Offset((page - 1) * size).Limit(size).Scan(&users).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	userIDs := make([]uint64, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
	}

	aggs := make([]customerAgg, 0)
	// MySQL returns SUM/MAX expressions as DECIMAL, which cannot scan into int64; cast to SIGNED.
	if err := h.ctx.DB.Raw(`
		SELECT user_id,
		       COUNT(*) AS order_count,
		       CAST(COALESCE(SUM(total_amount - COALESCE(refund_amount, 0)), 0) AS SIGNED) AS total_spent,
		       CAST(COALESCE(MAX(UNIX_TIMESTAMP(created_at) * 1000), 0) AS SIGNED) AS last_order_at
		FROM orders
		WHERE store_id = ? AND user_id IN ? AND status IN (1,2,3,5,6) AND deleted_at IS NULL
		GROUP BY user_id
	`, storeID, userIDs).Scan(&aggs).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	aggMap := make(map[uint64]customerAgg, len(aggs))
	for _, a := range aggs {
		aggMap[a.UserID] = a
	}

	var members []storeMemberRow
	if err := h.ctx.DB.Table("store_members sm").
		Select("sm.user_id, sm.level_id, ml.name AS level_name, ml.status AS level_status, sm.expires_at, sm.status, sm.remark").
		Joins("LEFT JOIN member_levels ml ON ml.id = sm.level_id").
		Where("sm.store_id = ? AND sm.user_id IN ? AND sm.deleted_at IS NULL", storeID, userIDs).
		Scan(&members).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	memberMap := make(map[uint64]storeMemberRow, len(members))
	for _, m := range members {
		memberMap[m.UserID] = m
	}

	list := make([]gin.H, 0, len(users))
	for _, u := range users {
		agg := aggMap[u.ID]
		m := memberMap[u.ID]
		list = append(list, gin.H{
			"id": u.ID, "nickname": u.Nickname, "avatar": u.Avatar, "phone": u.Phone,
			"order_count": agg.OrderCount, "total_spent": agg.TotalSpent,
			"last_order_at": agg.LastOrderAt, "level_id": m.LevelID,
			"level_name": m.LevelName, "level_status": m.LevelStatus,
			"expires_at": m.ExpiresAt, "member_status": m.Status,
			"remark": m.Remark,
		})
	}
	response.OK(c, gin.H{
		"list": list, "total": total, "page": page, "page_size": size,
		"total_pages": (total + int64(size) - 1) / int64(size),
	})
}

func (h *MemberHandler) isStoreCustomer(storeID, userID uint64) (bool, error) {
	var count int64
	err := h.ctx.DB.Raw(`
		SELECT COUNT(*) FROM (
			SELECT user_id FROM orders
			WHERE store_id = ? AND user_id = ? AND status IN (1,2,3,5,6) AND deleted_at IS NULL
			UNION
			SELECT user_id FROM recharges
			WHERE store_id = ? AND user_id = ? AND status = 1 AND deleted_at IS NULL
			UNION
			SELECT user_id FROM balance_logs
			WHERE store_id = ? AND user_id = ? AND deleted_at IS NULL
		) t
	`, storeID, userID, storeID, userID, storeID, userID).Scan(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (h *MemberHandler) GetCustomer(c *gin.Context) {
	storeID := GetStoreID(c)
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 64)
	ok, err := h.isStoreCustomer(storeID, userID)
	if err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if !ok {
		response.FailCode(c, errcode.ErrNotFound)
		return
	}
	var u user.User
	if err := h.ctx.DB.First(&u, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailCode(c, errcode.ErrNotFound)
			return
		}
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	var memberRows []storeMemberRow
	if err := h.ctx.DB.Table("store_members sm").
		Select("sm.user_id, sm.level_id, ml.name AS level_name, ml.status AS level_status, sm.expires_at, sm.status, sm.remark").
		Joins("LEFT JOIN member_levels ml ON ml.id = sm.level_id").
		Where("sm.store_id = ? AND sm.user_id = ? AND sm.deleted_at IS NULL", storeID, userID).
		Scan(&memberRows).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	var mr storeMemberRow
	if len(memberRows) > 0 {
		mr = memberRows[0]
	}

	orders := make([]customerOrderDTO, 0)
	if err := h.ctx.DB.Table("orders").
		Select("order_no, resource_name, status, total_amount, start_time, end_time, created_at").
		Where("store_id = ? AND user_id = ? AND deleted_at IS NULL", storeID, userID).
		Order("created_at DESC").Limit(50).Scan(&orders).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	recharges := make([]customerRechargeDTO, 0)
	if err := h.ctx.DB.Table("recharges").
		Select("id, amount, status, created_at").
		Where("store_id = ? AND user_id = ? AND deleted_at IS NULL", storeID, userID).
		Order("created_at DESC").Limit(50).Scan(&recharges).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}

	response.OK(c, customerDetailDTO{
		ID: u.ID, Nickname: u.Nickname, Avatar: u.Avatar, Phone: u.Phone,
		LevelID: mr.LevelID, LevelName: mr.LevelName, LevelStatus: mr.LevelStatus,
		ExpiresAt: mr.ExpiresAt, MemberStatus: mr.Status, Remark: mr.Remark,
		Orders: orders, Recharges: recharges,
	})
}

func (h *MemberHandler) SetCustomerMembership(c *gin.Context) {
	storeID := GetStoreID(c)
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 64)
	var req struct {
		LevelID   uint64 `json:"level_id" binding:"required"`
		ExpiresAt *int64 `json:"expires_at"`
		Remark    string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	if req.ExpiresAt != nil && *req.ExpiresAt <= 0 {
		response.FailMsg(c, errcode.ErrBadRequest, "到期时间必须大于0")
		return
	}
	ok, err := h.isStoreCustomer(storeID, userID)
	if err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if !ok {
		response.FailCode(c, errcode.ErrNotFound)
		return
	}
	var u user.User
	if err := h.ctx.DB.First(&u, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailCode(c, errcode.ErrNotFound)
			return
		}
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if !customerCanBeMember(u.Phone) {
		response.FailMsg(c, errcode.ErrBadRequest, "请先让用户授权绑定手机号")
		return
	}
	var level member.Level
	if err := h.ctx.DB.Where("id = ? AND store_id = ? AND status = 1", req.LevelID, storeID).First(&level).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailMsg(c, errcode.ErrBadRequest, "会员等级不存在或已停用")
			return
		}
		response.FailCode(c, errcode.ErrInternal)
		return
	}

	m := member.StoreMember{
		StoreID: storeID, UserID: userID, LevelID: req.LevelID,
		ExpiresAt: req.ExpiresAt, Status: 1, Remark: req.Remark, CreatedBy: GetUserID(c),
	}
	m.ID = utils.ShortID()
	err = h.ctx.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "store_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"level_id":   req.LevelID,
			"expires_at": req.ExpiresAt,
			"status":     1,
			"remark":     req.Remark,
			"updated_at": time.Now(),
			"deleted_at": nil,
		}),
	}).Create(&m).Error
	if err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	response.OKMsg(c, "会员设置成功")
}

func (h *MemberHandler) RemoveCustomerMembership(c *gin.Context) {
	storeID := GetStoreID(c)
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 64)
	ok, err := h.isStoreCustomer(storeID, userID)
	if err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if !ok {
		response.FailCode(c, errcode.ErrNotFound)
		return
	}
	var m member.StoreMember
	if err := h.ctx.DB.Where("store_id = ? AND user_id = ?", storeID, userID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailCode(c, errcode.ErrNotFound)
			return
		}
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if err := h.ctx.DB.Model(&m).Update("status", 0).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	response.OKMsg(c, "已取消会员")
}
