package utils

import (
	"regexp"
	"unicode/utf8"
)

var (
	phoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)
	emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// IsPhone 校验中国大陆手机号
func IsPhone(s string) bool {
	return phoneRe.MatchString(s)
}

// IsEmail 校验邮箱地址
func IsEmail(s string) bool {
	return emailRe.MatchString(s)
}

// StrLen 返回字符串的字符数 (非字节数)
func StrLen(s string) int {
	return utf8.RuneCountInString(s)
}

// InSlice 判断值是否在切片中
func InSlice[T comparable](val T, slice []T) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
