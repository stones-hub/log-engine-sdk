package k3

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

type RotateMode int

const (
	DefaultChannelSize            = 1000
	ROTATE_DAILY       RotateMode = 0
	ROTATE_HOURLY      RotateMode = 1
)

type K3Consumer interface {
	Add(data Data) error
	Flush() error
	Close() error
}

type K3LogConsumer struct {
	directory      string
	dateFormat     string
	fileSize       int64
	fileNamePrefix string
	currentFile    *os.File
	wg             sync.WaitGroup
	ch             chan []byte
	mutex          *sync.RWMutex
	sdkClose       bool
}

func (k *K3LogConsumer) Add(data Data) error {
	//TODO implement me
	panic("implement me")
}

func (k *K3LogConsumer) Flush() error {
	//TODO implement me
	panic("implement me")
}

func (k *K3LogConsumer) Close() error {
	//TODO implement me
	panic("implement me")
}

func (k *K3LogConsumer) init() error {
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
	return os.OpenFile(k.generateFileName(time.Now().Format(k.dateFormat), 0), os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
}

func (k *K3LogConsumer) generateFileName(t string, i int) string {
	var (
		prefix string
	)

	if len(k.fileNamePrefix) != 0 {
		prefix = k.fileNamePrefix + "."
	}

	if k.fileSize > 0 {
		return fmt.Sprintf("%s/%slog.%s_%d", k.directory, prefix, t, i)
	} else {
		return fmt.Sprintf("%s/%slog.%s", k.directory, prefix, t)
	}
}

func (k *K3LogConsumer) rsyncFile() {

}

type K3LogConsumerConfig struct {
	Directory      string
	RoteMode       RotateMode
	FileSize       int
	FileNamePrefix string
	ChannelSize    int
}

func NewLogConsumer(directory string, r RotateMode) (K3Consumer, error) {
	return NewLogConsumerWithFileSize(directory, r, 0)
}

func NewLogConsumerWithFileSize(directory string, r RotateMode, size int) (K3Consumer, error) {

	return NewLogConsumerWithConfig(K3LogConsumerConfig{
		Directory: directory,
		RoteMode:  r,
		FileSize:  size,
	})
}

func NewLogConsumerWithConfig(config K3LogConsumerConfig) (K3Consumer, error) {
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
