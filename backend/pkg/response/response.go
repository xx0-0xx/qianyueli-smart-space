// Package response 统一 HTTP 响应格式
package response

import (
	"net/http"

	"smart-space-management/backend/pkg/errcode"

	"github.com/gin-gonic/gin"
)

// Body 标准响应体
type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageData 分页数据
type PageData struct {
	List       interface{} `json:"list"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// OK 成功响应 (带 data)
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:    int(errcode.OK),
		Message: "ok",
		Data:    data,
	})
}

// OKMsg 成功响应 (自定义消息)
func OKMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Body{
		Code:    int(errcode.OK),
		Message: msg,
	})
}

// Created 201 创建成功
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Body{
		Code:    int(errcode.OK),
		Message: "created",
		Data:    data,
	})
}

// Fail 业务失败 (4xx/5xx)
func Fail(c *gin.Context, e *errcode.Error) {
	status := codeToHTTP(e.Code)
	c.AbortWithStatusJSON(status, Body{
		Code:    int(e.Code),
		Message: e.Message,
	})
}

// FailCode 用错误码 + 默认消息响应
func FailCode(c *gin.Context, code errcode.Code) {
	Fail(c, &errcode.Error{Code: code, Message: code.Message()})
}

// FailMsg 用错误码 + 自定义消息响应
func FailMsg(c *gin.Context, code errcode.Code, msg string) {
	Fail(c, &errcode.Error{Code: code, Message: msg})
}

// FailData 业务失败并携带附加数据（如验证码挑战）
func FailData(c *gin.Context, code errcode.Code, msg string, data interface{}) {
	status := codeToHTTP(code)
	c.AbortWithStatusJSON(status, Body{
		Code:    int(code),
		Message: msg,
		Data:    data,
	})
}

// PageOK 分页成功
func PageOK(c *gin.Context, data PageData) {
	c.JSON(http.StatusOK, Body{
		Code:    int(errcode.OK),
		Message: "ok",
		Data:    data,
	})
}

// codeToHTTP 错误码 → HTTP 状态码
func codeToHTTP(code errcode.Code) int {
	switch code {
	case errcode.OK:
		return http.StatusOK
	case errcode.ErrBadRequest, errcode.ErrPhoneRequired:
		return http.StatusBadRequest
	case errcode.ErrUnauthorized, errcode.ErrTokenExpired, errcode.ErrTokenInvalid:
		return http.StatusUnauthorized
	case errcode.ErrForbidden, errcode.ErrPermissionDenied, errcode.ErrNoControlPerm:
		return http.StatusForbidden
	case errcode.ErrNotFound, errcode.ErrUserNotFound, errcode.ErrStoreNotFound,
		errcode.ErrResourceNotFound, errcode.ErrOrderNotFound, errcode.ErrPaymentNotFound,
		errcode.ErrDeviceNotFound, errcode.ErrConfigNotFound, errcode.ErrRoleNotFound:
		return http.StatusNotFound
	case errcode.ErrConflict, errcode.ErrResourceNotFree, errcode.ErrResourceLockFail,
		errcode.ErrOrderTimeConflict, errcode.ErrPaymentDuplicate, errcode.ErrConfigDuplicate,
		errcode.ErrResourceInMaint, errcode.ErrOrderStatusInvalid:
		return http.StatusConflict
	case errcode.ErrRateLimited, errcode.ErrLoginLocked, errcode.ErrCaptchaRequired:
		return http.StatusTooManyRequests
	case errcode.ErrElysDisabled, errcode.ErrElysStoreNotBound,
		errcode.ErrElysPlatformUnauthorized, errcode.ErrElysPrepareFailed,
		errcode.ErrElysConsumeFailed, errcode.ErrElysQuotaExhausted:
		return http.StatusBadRequest
	case errcode.ErrElysAlreadyConsumed:
		return http.StatusConflict
	case errcode.ErrElysResultUnknown, errcode.ErrElysPendingCredit:
		return http.StatusAccepted
	default:
		return http.StatusInternalServerError
	}
}
