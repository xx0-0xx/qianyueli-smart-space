// Package imageutil 图片压缩工具（上传时自动压缩，减小体积）
package imageutil

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	// 注册解码器
	_ "image/gif"

	"golang.org/x/image/draw"
)

// CompressImage 压缩图片：宽度超 maxW 则等比缩小；PNG 无透明自动转 JPEG
// 仅当体积更小才落盘。返回最终文件名（可能扩展名从 .png 变 .jpg）
func CompressImage(path string, maxW, quality int) (string, error) {
	origInfo, err := os.Stat(path)
	if err != nil {
		return filepath.Base(path), err
	}
	origSize := origInfo.Size()

	f, err := os.Open(path)
	if err != nil {
		return filepath.Base(path), err
	}
	img, format, err := image.Decode(f)
	f.Close()
	if err != nil {
		return filepath.Base(path), err
	}

	// 缩放（仅当超过最大宽度）
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if maxW > 0 && w > maxW {
		newH := h * maxW / w
		if newH < 1 {
			newH = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, maxW, newH))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
		img = dst
	}

	// 决定编码格式与目标路径
	var buf bytes.Buffer
	newPath := path
	if format == "png" {
		if hasAlpha(img) {
			enc := png.Encoder{CompressionLevel: png.BestCompression}
			if err := enc.Encode(&buf, img); err != nil {
				return filepath.Base(path), err
			}
		} else {
			// 无透明 → 转 JPEG（大幅减小）
			if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
				return filepath.Base(path), err
			}
			newPath = strings.TrimSuffix(path, filepath.Ext(path)) + ".jpg"
		}
	} else {
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return filepath.Base(path), err
		}
	}

	// 只有更小才落盘（防止膨胀）
	if int64(buf.Len()) >= origSize {
		return filepath.Base(path), nil
	}
	if err := os.WriteFile(newPath, buf.Bytes(), 0644); err != nil {
		return filepath.Base(path), err
	}
	// 转换了扩展名 → 删原文件
	if newPath != path {
		os.Remove(path)
	}
	return filepath.Base(newPath), nil
}

// hasAlpha 检测图片是否含透明像素
func hasAlpha(img image.Image) bool {
	b := img.Bounds()
	// 采样检测（步长 2，足够判断）
	for y := b.Min.Y; y < b.Max.Y; y += 2 {
		for x := b.Min.X; x < b.Max.X; x += 2 {
			_, _, _, a := img.At(x, y).RGBA()
			if a < 0xffff {
				return true
			}
		}
	}
	return false
}
