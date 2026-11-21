package utils

import (
	"sync/atomic"
	"time"

	"github.com/bwmarrin/snowflake"
)

var node *snowflake.Node
var shortCounter uint64 = 0

// InitSnowflake 初始化雪花 ID 生成器 (nodeID 1-1023)
func InitSnowflake(nodeID int64) error {
	n, err := snowflake.NewNode(nodeID)
	if err != nil {
		return err
	}
	node = n
	return nil
}

// GenID 生成雪花 ID（仅用于不需要前端传递的场景）
func GenID() uint64 {
	if node == nil {
		_ = InitSnowflake(1)
	}
	return uint64(node.Generate().Int64())
}

// ShortID 生成短 ID（6-9位，用于前端传递的场景）
// 格式：时间戳后5位 + 递增序号2位
func ShortID() uint64 {
	n := atomic.AddUint64(&shortCounter, 1)
	ts := uint64(time.Now().UnixMilli()) % 100000
	return ts*100 + (n % 100)
}
