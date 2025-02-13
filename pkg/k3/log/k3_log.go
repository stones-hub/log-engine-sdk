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

type LogInterface interface {
	Add(data interface{}) error
	Flush() error
	Close() error
}

type Log struct {
	fileIndex      int            // 文件索引
	directory      string         // 日志存储地址
	dateFormat     string         // 时间格式
	fileSize       int64          // 单个文件大小
	fileNamePrefix string         // 文件前缀
	currentFile    *os.File       // 当前记录日志的文件fd
	wg             sync.WaitGroup // 协程退出等待
	ch             chan []byte    // 队列
	mutex          *sync.RWMutex  // 读写锁
}

// NewLogger 初始化日志实例
// directory 日志存储地址
// fileNamePrefix 文件前缀
// dateFormat 时间格式
// fileSize 单个文件大小
// channelSize 队列大小
func NewLogger(directory string, fileNamePrefix string, dateFormat string, fileSize int64, channelSize int) (*Log, error) {
	var (
		log *Log
		err error
		fd  *os.File
	)

	// 1. 初始化日志目录
	if _, err = os.Stat(directory); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(directory, os.ModePerm); err != nil {
			return nil, err
		}
	}

	// 2. 初始化日志文件
	fd, err = os.OpenFile(directory+"/"+fileNamePrefix+"_"+time.Now().Format(dateFormat)+"_0"+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	// 3. 生成日志实例
	log = &Log{
		directory:      directory,
		dateFormat:     dateFormat, // "2006-01-02",
		fileSize:       fileSize,
		fileNamePrefix: fileNamePrefix,
		currentFile:    fd,
		wg:             sync.WaitGroup{},
		ch:             make(chan []byte, channelSize),
		mutex:          &sync.RWMutex{},
	}

	// 4.初始化协程,从管道中拿数据
	log.wg.Add(1)

	go func() {
		defer func() {
			if e := recover(); e != nil {
				fmt.Println("log panic:", e)
			}
			log.wg.Done()
		}()

		for {
			select {
			case res, ok := <-log.ch:
				if !ok {
					return
				}
				// 将管道数据写入硬盘
				write(string(res))
			}
		}
	}()

	return log, nil
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
