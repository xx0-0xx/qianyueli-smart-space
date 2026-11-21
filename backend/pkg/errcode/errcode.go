// Package errcode 统一错误码体系。
// 错误码格式: 10xxx 通用 / 20xxx 用户 / 30xxx 门店 / 40xxx 资源 / 50xxx 订单 / 60xxx 支付 / 70xxx 设备 / 80xxx 配置 / 90xxx 权限
package errcode

import "fmt"

// Code 错误码
type Code int

// 通用 10xxx
const (
	OK                 Code = 0
	ErrInternal        Code = 10001
	ErrBadRequest      Code = 10002
	ErrNotFound        Code = 10003
	ErrConflict        Code = 10004
	ErrForbidden       Code = 10005
	ErrUnauthorized    Code = 10006
	ErrRateLimited     Code = 10007
	ErrLoginLocked     Code = 10008
	ErrCaptchaRequired Code = 10009
)

// 用户 20xxx
const (
	ErrUserNotFound    Code = 20001
	ErrWechatLoginFail Code = 20002
	ErrTokenExpired    Code = 20003
	ErrTokenInvalid    Code = 20004
	ErrPhoneRequired   Code = 20005
)

// 门店 30xxx
const (
	ErrStoreNotFound Code = 30001
)

// 资源 40xxx
const (
	ErrResourceNotFound Code = 40001
	ErrResourceNotFree  Code = 40002
	ErrResourceLockFail Code = 40003
	ErrResourceInMaint  Code = 40004
)

// 订单 50xxx
const (
	ErrOrderNotFound      Code = 50001
	ErrOrderStatusInvalid Code = 50002
	ErrOrderTimeConflict  Code = 50003
)

// 支付 60xxx
const (
	ErrPaymentNotFound  Code = 60001
	ErrPaymentDuplicate Code = 60002
	ErrWechatPayFail    Code = 60003
	ErrRefundFail       Code = 60004
)

// 团购核销 62xxx
const (
	ErrElysDisabled             Code = 62001
	ErrElysStoreNotBound        Code = 62002
	ErrElysPlatformUnauthorized Code = 62003
	ErrElysPrepareFailed        Code = 62004
	ErrElysConsumeFailed        Code = 62005
	ErrElysAlreadyConsumed      Code = 62006
	ErrElysResultUnknown        Code = 62007
	ErrElysQuotaExhausted       Code = 62008
	ErrElysPendingCredit        Code = 62009
)

// 设备 70xxx
const (
	ErrDeviceNotFound Code = 70001
	ErrDeviceOffline  Code = 70002
	ErrDeviceCmdFail  Code = 70003
	ErrNoControlPerm  Code = 70004
)

// 配置 80xxx
const (
	ErrConfigNotFound  Code = 80001
	ErrConfigInvalid   Code = 80002
	ErrConfigDuplicate Code = 80003
)

// 权限 90xxx
const (
	ErrPermissionDenied Code = 90001
	ErrRoleNotFound     Code = 90002
)

// 错误码 → 默认消息
var messages = map[Code]string{
	OK:                 "ok",
	ErrInternal:        "内部服务器错误",
	ErrBadRequest:      "请求参数错误",
	ErrNotFound:        "资源不存在",
	ErrConflict:        "资源冲突",
	ErrForbidden:       "权限不足",
	ErrUnauthorized:    "未登录或登录已过期",
	ErrRateLimited:     "请求过于频繁，请稍后再试",
	ErrLoginLocked:     "登录尝试过于频繁，请稍后再试",
	ErrCaptchaRequired: "请完成验证码后重试",

	ErrUserNotFound:    "用户不存在",
	ErrWechatLoginFail: "微信登录失败",
	ErrTokenExpired:    "令牌已过期",
	ErrTokenInvalid:    "令牌无效",
	ErrPhoneRequired:   "请先授权绑定手机号",

	ErrStoreNotFound: "门店不存在",

	ErrResourceNotFound: "资源不存在",
	ErrResourceNotFree:  "资源已被占用",
	ErrResourceLockFail: "资源锁定失败，请重试",
	ErrResourceInMaint:  "资源维护中，暂不可用",

	ErrOrderNotFound:      "订单不存在",
	ErrOrderStatusInvalid: "订单状态不可操作",
	ErrOrderTimeConflict:  "预约时段冲突",

	ErrPaymentNotFound:  "支付记录不存在",
	ErrPaymentDuplicate: "支付已处理，请勿重复操作",
	ErrWechatPayFail:    "微信支付失败",
	ErrRefundFail:       "退款失败",

	ErrElysDisabled:             "团购核销未启用",
	ErrElysStoreNotBound:        "该门店暂未接入团购核销",
	ErrElysPlatformUnauthorized: "该门店尚未完成平台授权",
	ErrElysPrepareFailed:        "验券失败，请核对券码",
	ErrElysConsumeFailed:        "核销失败，请稍后重试",
	ErrElysAlreadyConsumed:      "该券已核销，请勿重复充值",
	ErrElysResultUnknown:        "核销结果待核实，请稍后查看",
	ErrElysQuotaExhausted:       "核销额度不足，请联系管理员",
	ErrElysPendingCredit:        "核销成功，余额处理中",

	ErrDeviceNotFound: "设备不存在",
	ErrDeviceOffline:  "设备离线",
	ErrDeviceCmdFail:  "设备指令执行失败",
	ErrNoControlPerm:  "无设备控制权限",

	ErrConfigNotFound:  "配置不存在",
	ErrConfigInvalid:   "配置不合法",
	ErrConfigDuplicate: "配置已存在",

	ErrPermissionDenied: "权限不足",
	ErrRoleNotFound:     "角色不存在",
}

// Message 返回错误码对应的默认中文消息
func (c Code) Message() string {
	if msg, ok := messages[c]; ok {
		return msg
	}
	return fmt.Sprintf("未知错误 (code=%d)", c)
}

// New 创建一个带自定义消息的 Error
func (c Code) New(msg string) *Error {
	return &Error{Code: c, Message: msg}
}

// Errorf 创建一个带格式化消息的 Error
func (c Code) Errorf(format string, args ...interface{}) *Error {
	return &Error{Code: c, Message: fmt.Sprintf(format, args...)}
}

// Error 统一错误类型
type Error struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}
