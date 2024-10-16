package watch

import (
	"log-engine-sdk/pkg/k3"
	"os"
	"path/filepath"
	"time"
)

type SateFile struct {
	OnLine   map[string]FileSate `json:"on_line"`
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

func WatchRun() {

}

// ParserConfig 解析配置文件, 生成SateFile
func ParserConfig() {

}

// FetchWatchPath 获取需要监控的目录中的所有子目录
func FetchWatchPath(watchPath string) ([]string, error) {

	var (
		paths []string
		err   error
	)

	if err = filepath.WalkDir(watchPath, func(currentPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			paths = append(paths, currentPath)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return paths, err
}

// FetchWatchPathFile 获取监控目录中的所有文件
func FetchWatchPathFile(subPath string) ([]string, error) {
	return k3.FetchDirectory(subPath, -1)
}
