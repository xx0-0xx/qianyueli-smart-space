package handler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"smart-space-management/backend/internal/model/lead"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// LeadHandler 官网留资线索
type LeadHandler struct{ ctx *Context }

func NewLeadHandler(ctx *Context) *LeadHandler { return &LeadHandler{ctx: ctx} }

// 中国大陆手机号
var phoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

// CreateDemoLead 官网「预约演示」留资（公开，免登录）
// POST /api/v1/demo-lead
func (h *LeadHandler) CreateDemoLead(c *gin.Context) {
	var req struct {
		Name      string `json:"name"`
		Phone     string `json:"phone"`
		Company   string `json:"company"`
		City      string `json:"city"`
		StoreType string `json:"store_type"`
		Remark    string `json:"remark"`
		Website   string `json:"website"` // 蜜罐字段：正常用户不可见，填了视为机器人
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	// 蜜罐命中：机器人 → 直接返回成功但不落库
	if strings.TrimSpace(req.Website) != "" {
		response.OK(c, gin.H{"id": "0"})
		return
	}

	// 归一化
	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Company = strings.TrimSpace(req.Company)
	req.City = strings.TrimSpace(req.City)
	req.StoreType = strings.TrimSpace(req.StoreType)
	req.Remark = strings.TrimSpace(req.Remark)

	// 必填 + 格式校验
	if req.Name == "" || req.Phone == "" {
		response.FailMsg(c, errcode.ErrBadRequest, "姓名和手机号必填")
		return
	}
	if len([]rune(req.Name)) > 30 {
		response.FailMsg(c, errcode.ErrBadRequest, "姓名最多30个字")
		return
	}
	if !phoneRe.MatchString(req.Phone) {
		response.FailMsg(c, errcode.ErrBadRequest, "手机号格式不正确")
		return
	}
	if len([]rune(req.Company)) > 60 {
		response.FailMsg(c, errcode.ErrBadRequest, "公司名最多60个字")
		return
	}
	if len([]rune(req.City)) > 30 {
		response.FailMsg(c, errcode.ErrBadRequest, "城市最多30个字")
		return
	}
	if len([]rune(req.StoreType)) > 30 {
		response.FailMsg(c, errcode.ErrBadRequest, "门店类型最多30个字")
		return
	}
	if len([]rune(req.Remark)) > 200 {
		response.FailMsg(c, errcode.ErrBadRequest, "备注最多200个字")
		return
	}

	ctx := c.Request.Context()
	ip := c.ClientIP()

	// 限流：同 IP 每分钟 ≤3 次
	ipKey := "ssm:lead:ip:" + ip
	n, err := h.ctx.Redis.Incr(ctx, ipKey).Result()
	if err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if n == 1 {
		h.ctx.Redis.Expire(ctx, ipKey, time.Minute)
	}
	if n > 3 {
		response.FailCode(c, errcode.ErrRateLimited)
		return
	}

	// 去重：同手机号 10 分钟内只能提交一次
	phoneKey := "ssm:lead:phone:" + req.Phone
	ok, err := h.ctx.Redis.SetNX(ctx, phoneKey, "1", 10*time.Minute).Result()
	if err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if !ok {
		response.FailMsg(c, errcode.ErrBadRequest, "该手机号已提交过，请勿重复提交")
		return
	}

	// 落库
	ua := c.Request.UserAgent()
	if len([]rune(ua)) > 255 {
		ua = string([]rune(ua)[:255])
	}
	l := lead.Lead{
		Name:      req.Name,
		Phone:     req.Phone,
		Company:   req.Company,
		City:      req.City,
		StoreType: req.StoreType,
		Source:    "website",
		Remark:    req.Remark,
		IP:        ip,
		UserAgent: ua,
		Status:    lead.StatusNew,
	}
	l.ID = utils.ShortID()
	if err := h.ctx.DB.Create(&l).Error; err != nil {
		// 落库失败，释放去重标记，允许用户重试
		h.ctx.Redis.Del(ctx, phoneKey)
		response.FailCode(c, errcode.ErrInternal)
		return
	}

	response.OK(c, gin.H{"id": fmt.Sprintf("%d", l.ID)})
}

// ListDemoLeads 管理后台查看留资线索（需管理员）
// GET /api/v1/admin/leads?status=&keyword=&page=&page_size=
func (h *LeadHandler) ListDemoLeads(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if size < 1 || size > 100 {
		size = 20
	}

	q := h.ctx.DB.Model(&lead.Lead{})
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	if kw := strings.TrimSpace(c.Query("keyword")); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR phone LIKE ? OR company LIKE ?", like, like, like)
	}

	var total int64
	q.Count(&total)
	var list []lead.Lead
	q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&list)

	response.PageOK(c, response.PageData{
		List:       list,
		Total:      total,
		Page:       page,
		PageSize:   size,
		TotalPages: int((total + int64(size) - 1) / int64(size)),
	})
}

// UpdateDemoLead 管理后台更新线索状态/备注（需管理员）
// PUT /api/v1/admin/leads/:id
func (h *LeadHandler) UpdateDemoLead(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	var req struct {
		Status int    `json:"status"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	if req.Status < 0 || req.Status > 3 {
		response.FailMsg(c, errcode.ErrBadRequest, "状态值不合法")
		return
	}
	updates := map[string]interface{}{"status": req.Status}
	if strings.TrimSpace(req.Remark) != "" {
		updates["remark"] = strings.TrimSpace(req.Remark)
	}
	res := h.ctx.DB.Model(&lead.Lead{}).Where("id = ?", id).Updates(updates)
	if res.RowsAffected == 0 {
		response.FailCode(c, errcode.ErrNotFound)
		return
	}
	response.OKMsg(c, "已更新")
}
