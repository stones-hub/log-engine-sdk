package watch

import (
	"encoding/json"
	"io"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/config"
	"os"
	"path/filepath"
	"time"
)

type SateFile struct {
	OnLine   map[string]FileSate `json:"on_line"`  // key : 一批文件的索引名称, value : 文件信息
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
	var (
		watchConfig = config.GlobalConfig.Watch
		stateFile   *SateFile
		watchPaths  = make(map[string][]string)
		err         error
	)

	// 加载state文件到内存
	if stateFile, err = CreateAndLoadFileState(watchConfig.StateFilePath); err != nil {
		k3.K3LogError("WatchRun CreateAndLoadFileState error: %s", err.Error())
		return
	}

	// 遍历配置文件所有目录, 求并集
	/*
	  read_path : # read_path每个Key的目录不可以重复，且value不可以包含相同的子集
	    index_nginx: ["/Users/yelei/data/code/go-projects/logs/nginx"] # 必须是目录
	    index_admin : [ "/Users/yelei/data/code/go-projects/logs/admin"]
	    index_api : [ "/Users/yelei/data/code/go-projects/logs/api"]
	  max_read_count : 100 # 监控到文件变化时，一次读取文件最大次数
	  start_date : "2020-01-01 00:00:00" # 监控什么时间起创建的文件
	  obsolete_date_interval : 1 # 单位小时hour, 默认1小时, 超过多少时间文件未变化, 认为文件应该删除
	  state_file_path : "/state/core
	*/

	// 遍历所有的目录,找到所有需要监控的目录(包含子目录)
	for indexName, paths := range watchConfig.ReadPath {
		for _, path := range paths {
			subPaths, err := FetchWatchPath(path)
			if err != nil {
				k3.K3LogError("FetchWatchPath error: %s", err.Error())
				return
			}
			watchPaths[indexName] = subPaths
		}
	}

	// 完善StateFile 中的文件信息

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
