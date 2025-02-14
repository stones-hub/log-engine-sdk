package k3

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

/*
type RotateMode int
const (
	ROTATE_DAILY  RotateMode = 0
	ROTATE_HOURLY RotateMode = 1
)
*/

// 当前日志组件，支撑日志轮转，轮转方式采用文件大小和时间格式的方式，时间格式支持小时和天为单位

type IBaseLog interface {
	Add(data interface{}) error
	Flush()
	Close() error
}

type Logger struct {
	index     int             // 文件索引
	directory string          // 日志存储地址
	format    string          // 时间格式
	prefix    string          // 文件前缀
	size      int64           // 日志文件最大大小
	fd        *os.File        // 当前记录日志的文件fd
	wg        *sync.WaitGroup // 协程退出等待
	ch        chan []byte     // 队列
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

// 获取当前应该写入的文件路径和文件名
func getLogFileName(directory string, format string, prefix string, index int) string {
	return fmt.Sprintf("%s/%s_%s_%d.log", directory, prefix, time.Now().Format(format), index)
}

// NewLogger 多个场景都会生成一个Logger对象来记录适合自己模块的日志文件
func NewLogger(directory string, rotate RotateMode, prefix string, size int64, channelSize int, index int) (*Logger, error) {
	var (
		logger      *Logger
		err         error
		fd          *os.File
		logFileName string
	)

	logger = &Logger{
		index:     index,
		directory: directory,
		format:    getLogFormatter(rotate),
		prefix:    prefix,
		size:      size,
		fd:        nil,
		wg:        &sync.WaitGroup{},
		ch:        make(chan []byte, channelSize),
	}

	// 初始化日志目录
	if _, err = os.Stat(logger.directory); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(logger.directory, os.ModePerm); err != nil {
			return nil, err
		}
	}

	// 初始化日志文件
	logFileName = getLogFileName(logger.directory, logger.format, logger.prefix, logger.index)

	if fd, err = os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666); err != nil {
		return nil, err
	}
	logger.fd = fd

	logger.wg.Add(1)
	go logger.Flush()
	return logger, nil
}

// Add  将数据加入到管道, 协程管道的写或者读操作都是原子的，在没有多协程同时对一个管道执行读/写操作时，无需加锁
func (l *Logger) Add(data interface{}) error {
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

	if l.ch != nil {
		l.ch <- []byte(s)
		return nil
	} else {
		return errors.New("add event failed, Logger has been closed ")
	}
}

func (l *Logger) Flush() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Logger Flush goroutine panic: ", r)
		}
		l.wg.Done()
	}()

	for {
		select {
		case res, ok := <-l.ch:
			if !ok {
				log.Println("flush log data, channel is closed.")
				return
			}
			l.write(string(res))
		}
	}
}

func (l *Logger) write(data string) {
	var (
		logFileName string
		err         error
		fileInfo    os.FileInfo
	)

	// 获取当前应该写入的文件
	logFileName = getLogFileName(l.directory, l.format, l.prefix, l.index)

	if l.fd != nil && l.fd.Name() != logFileName {
		l.fd.Sync()
		l.fd.Close()
		if l.fd, err = os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644); err != nil {
			log.Fatalf("open log file failed: %s", err.Error())
			return
		}
	} else if l.fd == nil {
		if l.fd, err = os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644); err != nil {
			log.Fatalf("open log file failed: %s", err.Error())
			return
		}
	}

	// 解决日志超过最大轮转问题
	if l.size > 0 {
		if fileInfo, err = l.fd.Stat(); err != nil {
			log.Fatalf("get file stat error: %s", err.Error())
			return
		}

		if fileInfo.Size() >= l.size {
			l.fd.Sync()
			l.fd.Close()
			l.index++
			logFileName = getLogFileName(l.directory, l.format, l.prefix, l.index)
			if l.fd, err = os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644); err != nil {
				log.Fatalf("open log file failed: %s", err.Error())
				return
			}
		}
	}

	fmt.Fprintf(l.fd, "%s\n", data)
}

// Close 回收资源
func (l *Logger) Close() error {
	close(l.ch)
	l.wg.Wait()
	if l.fd != nil {
		_ = l.fd.Sync()
		_ = l.fd.Close()
		l.fd = nil
	}
	l.ch = nil
	log.Println("close logger success.")
	return nil
}
