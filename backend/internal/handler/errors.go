package handler

import (
	"errors"

	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func failFromErr(c *gin.Context, err error) {
	var e *errcode.Error
	if errors.As(err, &e) {
		response.Fail(c, e)
		return
	}
	response.FailCode(c, errcode.ErrInternal)
}
