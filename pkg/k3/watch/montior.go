package watch

import (
	"log-engine-sdk/pkg/k3/config"
	"os"
	"time"
)

type SateFile struct {
	Watch    map[string]FileSate `json:"watch"`
	Obsolete []string            `json:"obsolete"` // 被删除的文件
}

type FileSate struct {
	Path          string    `json:"path"`            // 文件地址
	Offset        int64     `json:"offset"`          // 当前文件读取的偏移量
	StartReadTime time.Time `json:"start_read_time"` // 开始读取时间
	LastReadTime  time.Time `json:"last_read_time"`  // 最后一次读取文件的时间
	Fd            *os.File  `json:"fd"`              // 文件描述符
	IndexName     string    `json:"index_name"`
}

// ParserConfig 解析配置文件, 生成SateFile
func ParserConfig() {

}

// FetchWatchPath 获取需要监控的目录
func FetchWatchPath(watch config.Watch) (path []string, err error) {

}
