package watch

import (
	"context"
	"fmt"
	"log-engine-sdk/pkg/k3"
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
	// 每小时检查GlobalFileStates中所有文件，如果超过DefaultObsoleteDate天没有读写，就检查文件是否已经读取完，如果没有读取完就读取一次文件，一次最多读取DefaultObsoleteMaxReadCount次
	DefaultObsoleteInterval     = 1
	DefaultObsoleteDate         = 1    // 单位天， 默认1， 表示文件如果1天没有写入, 就查看下是不是读取完了，没读完就读完整个文件.
	DefaultObsoleteMaxReadCount = 5000 // 对于长时间没有读写的文件， 一次最大读取次数

	GlobalWatchContextCancel context.CancelFunc // 每个indexName都对应一批目录，被一个单独的watch监控。用于取消watch的协程
	GlobalWatchContext       context.Context    // 控制watch协程主动退出
	GlobalWatchWG            *sync.WaitGroup    // 控制watch级别协程的等待退出
	GlobalWatchMutex         sync.Mutex         // 控制watch级别的并发操作的锁

	GlobalDataAnalytics k3.DataAnalytics // 日志接收器
)
