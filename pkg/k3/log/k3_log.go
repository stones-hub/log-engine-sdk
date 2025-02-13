package log

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

/**
当前日志组件，支撑日志轮转，轮转方式采用文件大小和时间格式的方式，时间格式支持小时和天为单位
*/

type RotateMode int

type IBaseLog interface {
	Add(data interface{}) error
	Flush() error
	Close() error
}

const (
	ROTATE_DAILY  RotateMode = 0
	ROTATE_HOURLY RotateMode = 1
)

type Log struct {
	index     int             // 文件索引
	directory string          // 日志存储地址
	format    string          // 时间格式
	prefix    string          // 文件前缀
	size      int64           // 单个文件大小
	fd        *os.File        // 当前记录日志的文件fd
	wg        *sync.WaitGroup // 协程退出等待
	ch        chan []byte     // 队列
	mutex     *sync.RWMutex   // 读写锁
}

// 获取日志轮转类型
func getLogFormatter(t RotateMode) string {
	switch t {
	case ROTATE_DAILY:
		return "2006-01-02"
	case ROTATE_HOURLY:
		return "2006-01-02-15"
	default:
		return "2006-01-02"
	}
}

func NewLogger(directory string, rotate RotateMode, prefix string, size int64, channelSize int, index int) (*Log, error) {
	var (
		log *Log
		err error
		fd  *os.File
	)

	log = &Log{
		index:     index,
		directory: directory,
		format:    getLogFormatter(rotate),
		prefix:    prefix,
		size:      size,
		fd:        nil,
		wg:        &sync.WaitGroup{},
		ch:        make(chan []byte, channelSize),
		mutex:     &sync.RWMutex{},
	}

	// 初始化日志目录
	if _, err = os.Stat(log.directory); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(log.directory, os.ModePerm); err != nil {
			return nil, err
		}
	}

	// 初始化日志文件
	if fd, err = initLogFile(log.directory, log.format, log.prefix, log.index); err != nil {
		return nil, err
	}
	log.fd = fd

	run(log)

	return log, nil
}

// 初始化日志文件
func initLogFile(directory string, format string, prefix string, index int) (*os.File, error) {
	var (
		fileName string
	)
	fileName = fmt.Sprintf("%s/%s_%s_%d.log", directory, prefix, time.Now().Format(format), index)
	return os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
}

// 协程获取管道数据，写入硬盘文件
func run(log *Log) {

}

func write(data string) {

}

// Add 将数据加入到管道
func (l *Log) Add(data interface{}) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	var s string

	switch v := data.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		s = string(jsonData)
	}

	l.ch <- []byte(s)
	return nil
}

func (l *Log) Flush() error {
	//TODO implement me
	panic("implement me")
}

func (l *Log) Close() error {
	l.wg.Wait()
	return nil
}
