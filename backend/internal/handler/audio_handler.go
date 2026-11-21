package handler

import (
	"os"
	"path/filepath"

	"smart-space-management/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type AudioHandler struct {
	ctx *Context
}

func NewAudioHandler(ctx *Context) *AudioHandler {
	return &AudioHandler{ctx: ctx}
}

// ListAudioFiles 返回 audio 目录下的所有 MP3 文件名
// GET /api/v1/audio/list
func (h *AudioHandler) ListAudioFiles(c *gin.Context) {
	dir := "./audio"
	entries, err := os.ReadDir(dir)
	if err != nil {
		response.OK(c, []string{})
		return
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".mp3" {
			files = append(files, e.Name())
		}
	}
	response.OK(c, files)
}
