package watch

import (
	"encoding/json"
	"io"
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

// CreateAndLoadFileState 创建并加载状态文件
func CreateAndLoadFileState(fileSatePath string) (*SateFile, error) {
	var (
		fd        *os.File
		err       error
		decoder   *json.Decoder
		stateFile SateFile
	)
	// 判断文件是否存在, 不存在就创建, 存在就将文本内容加载出来,映射到SateFile中
	if fd, err = os.OpenFile(fileSatePath, os.O_CREATE|os.O_RDWR, 0666); err != nil {
		return nil, err
	}
	defer fd.Close()

	decoder = json.NewDecoder(fd)

	if err = decoder.Decode(&stateFile); err != nil && err != io.EOF {
		return nil, err
	}

	return &stateFile, nil
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
