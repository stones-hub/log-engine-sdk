package watch

import (
	"context"
	"errors"
	"fmt"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/config"
	"log-engine-sdk/pkg/k3/protocol"
	"log-engine-sdk/pkg/k3/sender"
	"os"
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

func LoadFileState(filePath string) error {
	var (
		fd  *os.File
		err error
	)

	if fd, err = os.OpenFile(stateFilePath, os.O_RDWR, os.ModePerm); err != nil {
		return errors.New("[Run] open state file failed: " + err.Error())
	}
	defer fd.Close()

	return err
}

// Run 启动监听
func Run(directory map[string][]string) error {
	var (
		err           error
		stateFilePath string
	)

	// 1. 初始化批量日志写入, 引入elk
	if err = InitConsumerBatchLog(); err != nil {
		return errors.New("[Run] InitConsumerBatchLog failed: " + err.Error())
	}

	// 2. 初始化FileState 文件
	stateFilePath = k3.GetRootPath() + config.GlobalConfig.Watch.StateFilePath
	// 2.1. 检查core.json是否存在，不存在就创建，并且load到FileState变量中
	if !k3.FileExists(stateFilePath) {
		// 创建文件
		if _, err = os.OpenFile(stateFilePath, os.O_CREATE, os.ModePerm); err != nil {
			return errors.New("[Run] create state file failed: " + err.Error())
		}
	}

	// 打开文件state，load到FileState中
	if err = LoadFileState(stateFilePath); err != nil {
		return errors.New("[Run] load file state failed : " + err.Error())
	}

	// 2.2. 遍历硬盘上的所有文件，如果FileState中没有，就add
	// 2.3. 检查FileState中的文件是否存在，不存在就delete掉
	// 2.4. 将FileState数据写入硬盘

	// 3. 初始化watcher，每个index_name 创建一个协程来监听, 如果有协程创建不成功，或者意外退出，则程序终止

	// 4. 定时更新 FileState 数据到硬盘
	return nil
}
