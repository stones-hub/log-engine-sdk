package log

import (
	"encoding/json"
	"os"
	"sync"
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
