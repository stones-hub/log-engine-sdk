package k3

import (
	"errors"
	"net/url"
	"sync"
	"time"
)

const (
	DefaultTimeout       = 30000
	DefaultBatchSize     = 100
	MaxBatchSize         = 1000
	DefaultInterval      = 5
	DefaultCacheCapacity = 100
)

type K3BatchConsumer struct {
	serverURL   string
	appId       string
	timeout     time.Duration
	compress    bool
	bufferMutex *sync.RWMutex
	cacheMutex  *sync.RWMutex

	buffer        []Data
	batchSize     int
	cacheBuffer   [][]Data
	cacheCapacity int
}

func (k K3BatchConsumer) Add(data Data) error {
	//TODO implement me
	panic("implement me")
}

func (k K3BatchConsumer) Flush() error {
	//TODO implement me
	panic("implement me")
}

func (k K3BatchConsumer) Close() error {
	//TODO implement me
	panic("implement me")
}

type K3BatchConsumerConfig struct {
	ServerURL     string
	AppId         string
	BatchSize     int
	Timeout       time.Duration
	Compress      bool
	AutoFlush     bool
	Interval      int
	CacheCapacity int
}

// NewBatchConsumerWithBatchSize creates a new K3BatchConsumer with batch size.
// batchSize should be between 1 and 1000.
func NewBatchConsumerWithBatchSize(serverURL string, appId string, batchSize int) (K3Consumer, error) {

}

func initBatchConsumer(config K3BatchConsumerConfig) (K3Consumer, error) {
	var (
		err             error
		u               *url.URL
		batchSize       int
		cacheCapacity   int
		timeout         time.Duration
		k3BatchConsumer *K3BatchConsumer
		interval        int
	)

	if len(config.ServerURL) == 0 {
		K3LogInfo("serverURL is empty, use default serverURL")
		return nil, errors.New("serverURL is empty")
	}

	// 解析日志提交端的地址
	if u, err = url.Parse(config.ServerURL); err != nil {
		return nil, err
	}

	// 批量大小
	if config.BatchSize > MaxBatchSize {
		batchSize = MaxBatchSize
	} else if config.BatchSize <= 0 {
		batchSize = DefaultBatchSize
	} else {
		batchSize = config.BatchSize
	}

	// 缓存大小
	if config.CacheCapacity <= 0 {
		cacheCapacity = DefaultCacheCapacity
	} else {
		cacheCapacity = config.CacheCapacity
	}

	// 超时时间
	if config.Timeout <= 0 {
		timeout = DefaultTimeout
	} else {
		timeout = config.Timeout
	}

	k3BatchConsumer = &K3BatchConsumer{
		serverURL:     u.String(),
		appId:         config.AppId,
		timeout:       time.Duration(timeout) * time.Millisecond,
		compress:      config.Compress,
		bufferMutex:   &sync.RWMutex{},
		cacheMutex:    &sync.RWMutex{},
		buffer:        make([]Data, 0, batchSize),
		cacheBuffer:   make([][]Data, cacheCapacity),
		batchSize:     batchSize,
		cacheCapacity: cacheCapacity,
	}

	if config.Interval == 0 {
		interval = DefaultInterval
	} else {
		interval = config.Interval
	}

	if config.AutoFlush {
		go func() {
			t := time.NewTicker(time.Duration(interval) * time.Second)

			defer t.Stop()

			for {
				<-t.C
				// TODO flush数据
			}
		}()
	}

	return k3BatchConsumer, nil
}
