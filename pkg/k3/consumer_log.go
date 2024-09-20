package k3

import (
	"encoding/json"
	"errors"
	"fmt"
	"log-engine-sdk/pkg/k3/protocol"
	"os"
	"sync"
	"time"
)

var (
	// 日志文件索引， 用于创建有文件最大容量边界的文件名编号， 默认为0
	LogFileIndex = 0
)

type RotateMode int

const (
	DefaultChannelSize            = 1000 // 默认队列大小
	ROTATE_DAILY       RotateMode = 0    //  时间格式
	ROTATE_HOURLY      RotateMode = 1    // 时间格式
)

type K3LogConsumer struct {
	directory      string         // 日志存储地址
	dateFormat     string         // 时间格式
	fileSize       int64          // 单个文件大小
	fileNamePrefix string         // 文件前缀
	currentFile    *os.File       // 当前记录日志的文件fd
	wg             sync.WaitGroup // 协程退出等待
	ch             chan []byte    // 队列
	mutex          *sync.RWMutex  // 读写锁
	sdkClose       bool           // sdk关闭
}

// Add 写入日志, 将日志写入到 chan
func (k *K3LogConsumer) Add(data protocol.Data) error {
	var (
		err error
		b   []byte
	)

	k.mutex.Lock()
	defer k.mutex.Unlock()

	if k.sdkClose {
		err = errors.New("add event failed, SDK has been closed ")
		K3LogError(err.Error())
	} else {
		if b, err = json.Marshal(data); err != nil {
			return err
		} else {
			k.ch <- b
		}
	}
	return err
}

func (k *K3LogConsumer) Flush() error {
	var (
		err error
	)

	K3LogInfo("flush log data")
	k.mutex.Lock()
	defer func() { k.mutex.Unlock() }()

	if k.currentFile != nil {
		err = k.currentFile.Sync()
	}
	return err
}

func (k *K3LogConsumer) Close() error {
	var (
		err error
	)

	K3LogInfo("close log consumer")
	k.mutex.Lock()
	defer k.mutex.Unlock()

	if k.sdkClose {
		err = errors.New("sdk has been closed")
	} else {
		close(k.ch) // 关闭channel，初始化管道数据无需再写入数据了
		k.wg.Wait() // 等待协程退出

		// 清理fd
		if k.currentFile != nil {
			_ = k.currentFile.Sync()
			err = k.currentFile.Close()
			k.currentFile = nil
		}
	}

	k.sdkClose = true
	return err
}

func (k *K3LogConsumer) init() error {
	var (
		fd  *os.File
		err error
	)
	if fd, err = k.initLogFile(); err != nil {
		K3LogError("init log file error: %s", err.Error())
		return err
	} else {
		k.currentFile = fd
	}

	k.wg.Add(1)

	// 开始用协程来处理数据写入日志文件
	go func() {
		defer func() {
			if e := recover(); e != nil {
				K3LogError("consumer log panic: %s", e)
			}

			k.wg.Done()
		}()

		for {
			select {
			case res, ok := <-k.ch:
				if !ok {
					return
				}

				jsonStr := parseTime(res)
				// K3LogInfo("write event data :%s", jsonStr)
				// 将管道的数据写入到 log 文件
				k.rsyncFile(jsonStr)
			}
		}
	}()

	K3LogInfo("log consumer init success, log path :", k.directory)
	return nil
}

func (k *K3LogConsumer) initLogFile() (*os.File, error) {
	var (
		err error
	)
	if _, err = os.Stat(k.directory); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(k.directory, os.ModePerm); err != nil {
			return nil, err
		}
	}
	// 创建文件名

	logFileName := k.generateFileName(time.Now().Format(k.dateFormat), 0)
	K3LogInfo("log file name:%s", logFileName)

	return os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
}

// generateFileName 生成文件名
// t 当前时间
// i 文件索引
func (k *K3LogConsumer) generateFileName(t string, i int) string {
	var (
		prefix string
	)

	if len(k.fileNamePrefix) != 0 {
		prefix = k.fileNamePrefix + "."
	}

	if k.fileSize > 0 {
		// logs/prefix.log.2023-01-01_1
		return fmt.Sprintf("%s/%slog.%s_%d", k.directory, prefix, t, i)
	} else {
		// logs/log.2023-01-01
		return fmt.Sprintf("%s/%slog.%s", k.directory, prefix, t)
	}
}

func (k *K3LogConsumer) rsyncFile(jsonStr string) {
	var (
		fName string
		err   error
		stat  os.FileInfo
	)

	// 取当前应该写入的文件
	fName = k.generateFileName(time.Now().Format(k.dateFormat), LogFileIndex)

	if k.currentFile != nil && k.currentFile.Name() != fName {
		_ = k.currentFile.Close()
		if k.currentFile, err = os.OpenFile(fName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm); err != nil {
			K3LogError("open file error: %s", err.Error())
			return
		}
	} else if k.currentFile == nil {
		if k.currentFile, err = os.OpenFile(fName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm); err != nil {
			K3LogError("open file error: %s", err.Error())
			return
		}
	}

	if k.fileSize > 0 {
		if stat, err = k.currentFile.Stat(); err != nil {
			K3LogError("get file stat error: %s", err.Error())
			return
		}

		if stat.Size() >= k.fileSize {
			_ = k.currentFile.Close()
			LogFileIndex++
			fName = k.generateFileName(time.Now().Format(k.dateFormat), LogFileIndex)
			if k.currentFile, err = os.OpenFile(fName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm); err != nil {
				K3LogError("open file error: %s", err.Error())
				return
			}
		}
	}

	// 数据写入文件
	if _, err = fmt.Fprint(k.currentFile, jsonStr+"\n"); err != nil {
		K3LogError("write file error: %s", err.Error())
		return
	}
}

type K3LogConsumerConfig struct {
	Directory      string
	RoteMode       RotateMode
	FileSize       int
	FileNamePrefix string
	ChannelSize    int
}

func NewLogConsumer(directory string, r RotateMode) (protocol.K3Consumer, error) {
	return NewLogConsumerWithFileSize(directory, r, 0)
}

func NewLogConsumerWithFileSize(directory string, r RotateMode, size int) (protocol.K3Consumer, error) {

	return NewLogConsumerWithConfig(K3LogConsumerConfig{
		Directory: directory,
		RoteMode:  r,
		FileSize:  size,
	})
}

func NewLogConsumerWithConfig(config K3LogConsumerConfig) (protocol.K3Consumer, error) {
	var (
		dateFormat string
		chSize     int
	)

	switch config.RoteMode {
	case ROTATE_DAILY:
		dateFormat = "2006-01-02"
	case ROTATE_HOURLY:
		dateFormat = "2006-01-02-15"
	default:
		K3LogError("unknown rotate mode")
		return nil, errors.New("rotate mode")
	}

	if config.ChannelSize > 0 {
		chSize = config.ChannelSize
	} else {
		chSize = DefaultChannelSize
	}

	logConsumer := &K3LogConsumer{
		directory:      config.Directory,
		dateFormat:     dateFormat,
		fileSize:       int64(config.FileSize * 1024 * 1024),
		fileNamePrefix: config.FileNamePrefix,
		currentFile:    nil,
		wg:             sync.WaitGroup{},
		ch:             make(chan []byte, chSize),
		mutex:          new(sync.RWMutex),
		sdkClose:       false,
	}
	return logConsumer, logConsumer.init()
}
