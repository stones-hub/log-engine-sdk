package watch

import (
	"context"
	"fmt"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/config"
	"log-engine-sdk/pkg/k3/protocol"
	"log-engine-sdk/pkg/k3/sender"
	"os"
	"path/filepath"
	"sync"
)

type FileState struct {
	Path          string
	Offset        int64
	StartReadTime int64
	LastReadTime  int64
	IndexName     string
}

func (f *FileState) String() string {
	return fmt.Sprintf("Path: %s, Offset: %d, StartReadTime: %d, LastReadTime: %d, IndexName: %s", f.Path, f.Offset, f.StartReadTime, f.LastReadTime, f.IndexName)
}

var (
	FileStateLock       sync.Mutex                    // 控制GlobalFileStates的锁
	GlobalFileStates    = make(map[string]*FileState) // 对应监控的所有文件的状态，映射 core.json文件
	DefaultMaxReadCount = 200                         // 每次读取日志文件的最大次数

	// DefaultSyncInterval 单位秒, 默认为60s
	// 将硬盘上最新的文件列表同步到GlobalFileStates，并将GlobalFileStates数据同步到Disk硬盘存储
	DefaultSyncInterval = 60

	// DefaultObsoleteInterval  单位小时，默认1.
	//  会员卡每小时检查GlobalFileStates中所有文件，如果超过DefaultObsoleteDate天没有读写，就检查文件是否已经读取完，如果没有读取完就读取一次文件，一次最多读取DefaultObsoleteMaxReadCount次
	DefaultObsoleteInterval     = 1
	DefaultObsoleteDate         = 1    // 单位天， 默认1， 表示文件如果1天没有写入, 就查看下是不是读取完了，没读完就读完整个文件.
	DefaultObsoleteMaxReadCount = 5000 // 对于长时间没有读写的文件， 一次最大读取次数

	GlobalWatchContextCancel context.CancelFunc // 每个indexName都对应一批目录，被一个单独的watch监控。用于取消watch的协程
	GlobalWatchContext       context.Context    // 控制watch协程主动退出
	GlobalWatchWG            *sync.WaitGroup    // 控制watch级别协程的等待退出
	GlobalWatchMutex         sync.Mutex         // 控制watch级别的并发操作的锁

	GlobalDataAnalytics k3.DataAnalytics // 日志接收器
)

func InitConsumerBatchLog() error {
	var (
		elk      *sender.ElasticSearchClient
		err      error
		consumer protocol.K3Consumer
	)
	if elk, err = sender.NewElasticsearch(config.GlobalConfig.ELK.Address,
		config.GlobalConfig.ELK.Username,
		config.GlobalConfig.ELK.Password); err != nil {
		return err
	}

	if consumer, err = k3.NewBatchConsumerWithConfig(k3.K3BatchConsumerConfig{
		Sender:        elk,
		BatchSize:     config.GlobalConfig.Consumer.ConsumerBatchSize,
		AutoFlush:     config.GlobalConfig.Consumer.ConsumerBatchAutoFlush,
		Interval:      config.GlobalConfig.Consumer.ConsumerBatchInterval,
		CacheCapacity: config.GlobalConfig.Consumer.ConsumerBatchCapacity,
	}); err != nil {
		return err
	}
	GlobalDataAnalytics = k3.NewDataAnalytics(consumer)

	return nil
}

// InitFileState 初始化FileState
func InitFileState() {
	// 解决硬盘state/core.json文件问题

	// 判断state/core.json文件是否存在, 如果存在就不管，如果不存在就创建

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

// ForceSyncFileStateToDisk 强制遍历硬盘所有文件，同步到FileState 中并生成硬盘文件
func ForceSyncFileStateToDisk() error {

	return nil
}

// InitWatcher 初始化Watcher监听
func InitWatcher(indexName string) {

}

// Run 启动监听
func Run() {
	// 初始化批量日志写入
	if err := InitConsumerBatchLog(); err != nil {
		k3.K3LogError("[Run] InitConsumerBatchLog failed: ", err.Error())
		return
	}

}

// ClockSyncFileState2Disk 定时将GlobalFileStates数据同步到Disk硬盘存储
func ClockSyncFileState2Disk() {

}

/*

1. 初始化日志批量写入程序, (已引入elk)
2. 初始化FileState文件 state/core.json
	1)、如果core.json存在，就遍历当前硬盘所有文件，比对core.json的数据，如果文件名存在就跳过，如果不存在就新增并初始化
	2)、如果core.json不存在，就遍历当前硬盘所有文件，全部写入到core.json
	3)、剔除core.json存在，但硬盘中不存在的文件
3. 初始化watcher，每个index_name 创建一个协程来监听, 如果有协程创建不成功，或者意外退出，则程序终止
*/
