package watch

import (
	"encoding/json"
	"fmt"
	"io"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/config"
	"os"
	"path/filepath"
	"time"
)

type FileState struct {
	Path          string
	Offset        int64
	StartReadTime time.Time
	LastReadTime  time.Time
	IndexName     string
}

var (
	GlobalFileFds    = make(map[string]*os.File)
	GlobalFileStates = make(map[string]*FileState)
)

func Run() {

	var (
		watchConfig    = config.GlobalConfig.Watch
		watchPaths     = make(map[string][]string)
		watchFilePaths = make(map[string][]string)
		err            error
	)

	// 用于测试用
	watchConfig = config.Watch{
		ReadPath: map[string][]string{
			"index_nginx": []string{
				"/Users/yelei/data/code/go-projects/logs/nginx",
			},
			"index_admin": []string{
				"/Users/yelei/data/code/go-projects/logs/admin",
			},
			"index_api": []string{
				"/Users/yelei/data/code/go-projects/logs/api",
			},
		},
		StateFilePath:        "state/core.json",
		MaxReadCount:         1000,
		StartDate:            time.Now(),
		ObsoleteDateInterval: 1,
	}

	// 如果state file文件没有就创建，如果有就load文件内容到stateFile
	if GlobalFileStates, err = CreateORLoadFileState(watchConfig.StateFilePath); err != nil {
		k3.K3LogError("WatchRun CreateAndLoadFileState error: %s", err.Error())
		return
	}

	// 遍历所有的目录,找到所有需要监控的目录(包含子目录) 和 所有文件
	for indexName, paths := range watchConfig.ReadPath {
		for _, path := range paths {
			subPaths, err := FetchWatchPath(path)
			if err != nil {
				k3.K3LogError("FetchWatchPath error: %s", err.Error())
				return
			}
			watchPaths[indexName] = subPaths

			filePaths, err := FetchWatchPathFile(path)
			if err != nil {
				k3.K3LogError("FetchWatchPathFile error: %s", err.Error())
				return
			}
			watchFilePaths[indexName] = filePaths
		}
	}

	fmt.Println(watchPaths, watchFilePaths, GlobalFileStates)

	/*
		watch.yaml 配置文件信息
		read_path : # read_path每个Key的目录不可以重复，且value不可以包含相同的子集
		  index_nginx: ["/Users/yelei/data/code/go-projects/logs/nginx"] # 必须是目录
		  index_admin : [ "/Users/yelei/data/code/go-projects/logs/admin"]
		  index_api : [ "/Users/yelei/data/code/go-projects/logs/api"]
		max_read_count : 100 # 监控到文件变化时，一次读取文件最大次数
		start_date : "2020-01-01 00:00:00" # 监控什么时间起创建的文件
		obsolete_date_interval : 1 # 单位小时hour, 默认1小时, 超过多少时间文件未变化, 认为文件应该删除
		state_file_path : "/state/core

		watchPaths : map[
		index_admin:[/Users/yelei/data/code/go-projects/logs/admin /Users/yelei/data/code/go-projects/logs/admin/err]
		index_api:[/Users/yelei/data/code/go-projects/logs/api /Users/yelei/data/code/go-projects/logs/api/err]
		index_nginx:[/Users/yelei/data/code/go-projects/logs/nginx /Users/yelei/data/code/go-projects/logs/nginx/err]]

		watchFilePaths : map[
		index_admin:[/Users/yelei/data/code/go-projects/logs/admin/admin.log /Users/yelei/data/code/go-projects/logs/admin/err/err.log]
		index_api:[/Users/yelei/data/code/go-projects/logs/api/api.log /Users/yelei/data/code/go-projects/logs/api/err/err.log]
		index_nginx:[/Users/yelei/data/code/go-projects/logs/nginx/err/err.log /Users/yelei/data/code/go-projects/logs/nginx/nginx.log]]
	*/
}

// SyncWatchFiles2FileStates
// 初始化时
// 遍历硬盘上被监控目录的所有文件, 判断文件是否在FileState中，如果不在，证明是新增的文件, 则添加到FileState中
func SyncWatchFiles2FileStates(watchFiles map[string][]string) {

}

// CheckDiskFileIsExistInFileStates 判断文件是否在FileState中
func CheckDiskFileIsExistInFileStates(diskFilePath string) bool {
	for filePath := range GlobalFileStates {
		if filePath == diskFilePath {
			return true
		}
	}
	return false
}

// CheckFileStateIsExistInDiskFiles 判断FileState是否在硬盘中
func CheckFileStateIsExistInDiskFiles(fileState *FileState) bool {
	return false
}

// SyncFileStates2WatchFiles 初始化时
// 遍历FileState中记录的所有文件，如果文件不存在于本地硬盘中，证明已经被删除了，对应在FileState中删除 func SyncFileStates2WatchFiles() {

// 启动后，定时检查FileState中的记录文件，如果一段时间都没有变化，证明文件不会再写入了， 就检查是否已经读完, 没读完就一次性读完它

// 启动后，定时检查FileState中的记录文件，是否还存在在硬盘中，如果不存在就更新FileState

// CreateORLoadFileState 创建并加载状态文件
func CreateORLoadFileState(fileSatePath string) (map[string]*FileState, error) {
	var (
		fd         *os.File
		err        error
		decoder    *json.Decoder
		fileStates = make(map[string]*FileState)
	)
	// 判断文件是否存在, 不存在就创建, 存在就将文本内容加载出来,映射到SateFile中
	if fd, err = os.OpenFile(fileSatePath, os.O_CREATE|os.O_RDWR, 0666); err != nil {
		return nil, err
	}
	defer fd.Close()

	decoder = json.NewDecoder(fd)

	if err = decoder.Decode(&fileStates); err != nil && err != io.EOF {
		return nil, err
	}

	return fileStates, nil
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
func FetchWatchPathFile(watchPath string) ([]string, error) {
	return k3.FetchDirectory(watchPath, -1)
}
