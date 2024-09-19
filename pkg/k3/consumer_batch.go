package k3

import (
	"log-engine-sdk/pkg/k3/protocol"
	"sync"
	"time"
)

const (
	DefaultInterval      = 5   // 默认定时检查缓存时间间隔
	DefaultBatchSize     = 100 // 默认批量提交大小
	MaxBatchSize         = 200 // 批量提交最大值, 需要控制一下
	DefaultCacheCapacity = 100 // 默认缓存容量
)

type K3BatchConsumer struct {
	bufferMutex *sync.RWMutex // buffer锁，用于Data数据在缓存中读取是否安全
	cacheMutex  *sync.RWMutex // cache锁

	buffer        []protocol.Data   // 缓存数据slice
	batchSize     int               // 批量提交大小
	cacheBuffer   [][]protocol.Data // buffer的数据会先写入 cacheBuffer中
	cacheCapacity int               // cacheBuffer的最大容量

	wg        *sync.WaitGroup // 用于管控自动刷新协程
	closed    chan struct{}
	autoFlush bool            // 是否自动上报
	sender    protocol.Sender // 不同的日志存储类型，用不同的实现即可
}

// fetchBufferLength returns the length of buffer
func (k *K3BatchConsumer) fetchBufferLength() int {
	k.bufferMutex.RLock()
	defer k.bufferMutex.RUnlock()
	return len(k.buffer)
}

// fetchCacheLength returns the length of cacheBuffer
func (k *K3BatchConsumer) fetchCacheLength() int {
	k.cacheMutex.RLock()
	defer k.cacheMutex.RUnlock()
	return len(k.cacheBuffer)
}

// Add adds data to buffer
func (k *K3BatchConsumer) Add(data protocol.Data) error {
	k.bufferMutex.Lock()
	k.buffer = append(k.buffer, data)
	k.bufferMutex.Unlock()
	// K3LogInfo("Add data to buffer, current buffer length: %d\n", k.fetchBufferLength())

	// 当buffer长度大于等于 batchSize 或者 cacheBuffer的长度大于0，则立即flush, 要么buffer满了，要么cacheBuffer有数据都可以刷新发送
	if k.fetchBufferLength() >= k.batchSize || k.fetchCacheLength() > 0 {
		return k.Flush()
	}

	return nil
}

// Flush flushes the buffer to the server
func (k *K3BatchConsumer) Flush() error {
	var (
		err error
	)

	k.cacheMutex.Lock()
	defer k.cacheMutex.Unlock()

	k.bufferMutex.Lock()
	defer k.bufferMutex.Unlock()

	// 当buffer为空，cacheBuffer为空，则直接返回, 没有数据不用提交数据
	if len(k.buffer) == 0 && len(k.cacheBuffer) == 0 {
		return nil
	}

	// 当buffer长度大于等于 batchSize，则将buffer中的数据写入cacheBuffer中，并清空buffer
	if len(k.buffer) >= k.batchSize {
		k.cacheBuffer = append(k.cacheBuffer, k.buffer)
		k.buffer = make([]protocol.Data, 0, k.batchSize)
	}

	// 当cacheBuffer长度大于等于 cacheCapacity，则将cacheBuffer中的数据写入server，并清空cacheBuffer
	if len(k.cacheBuffer) >= k.cacheCapacity || len(k.cacheBuffer) > 0 {
		// 减少一个cache buffer , 并上传
		err = k.sender.Send(k.cacheBuffer[0])
		k.cacheBuffer = k.cacheBuffer[1:]
	}

	return err
}

func (k *K3BatchConsumer) FlushAll() error {
	var (
		err error
	)
	// 缓存中一直有数据，就需要不断的send， 直到结束
	for k.fetchCacheLength() > 0 || k.fetchBufferLength() > 0 {
		k.cacheBuffer = append(k.cacheBuffer, k.buffer)
		k.buffer = make([]protocol.Data, 0, k.batchSize)
		if err = k.sender.Send(k.cacheBuffer[0]); err != nil {
			return err
		}
		k.cacheBuffer = k.cacheBuffer[1:]
	}

	return err
}

// Close closes the consumer
func (k *K3BatchConsumer) Close() error {
	K3LogInfo("Close K3BatchConsumer")
	if k.autoFlush {
		close(k.closed)
		k.wg.Wait()
	}
	return k.FlushAll()
}

type K3BatchConsumerConfig struct {
	Sender        protocol.Sender
	BatchSize     int
	AutoFlush     bool
	Interval      int
	CacheCapacity int
}

// NewBatchConsumer creates a new K3BatchConsumer with default batch size.
func NewBatchConsumer(sender protocol.Sender) (protocol.K3Consumer, error) {
	return initBatchConsumer(K3BatchConsumerConfig{
		Sender: sender,
	})
}

// NewBatchConsumerWithBatchSize creates a new K3BatchConsumer with batch size.
// batchSize should be between 1 and 1000.
func NewBatchConsumerWithBatchSize(sender protocol.Sender, batchSize int) (protocol.K3Consumer, error) {
	return initBatchConsumer(K3BatchConsumerConfig{
		Sender:    sender,
		BatchSize: batchSize,
	})
}

func NewBatchConsumerWithConfig(config K3BatchConsumerConfig) (protocol.K3Consumer, error) {
	return initBatchConsumer(config)
}

func initBatchConsumer(config K3BatchConsumerConfig) (protocol.K3Consumer, error) {
	var (
		batchSize       int
		cacheCapacity   int
		k3BatchConsumer *K3BatchConsumer
		interval        int
	)

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

	k3BatchConsumer = &K3BatchConsumer{
		bufferMutex:   &sync.RWMutex{},
		cacheMutex:    &sync.RWMutex{},
		buffer:        make([]protocol.Data, 0, batchSize),
		cacheBuffer:   make([][]protocol.Data, 0, cacheCapacity),
		batchSize:     batchSize,
		cacheCapacity: cacheCapacity,
		wg:            &sync.WaitGroup{},
		closed:        make(chan struct{}),
		autoFlush:     config.AutoFlush,
		sender:        config.Sender,
	}

	if config.Interval == 0 {
		interval = DefaultInterval
	} else {
		interval = config.Interval
	}

	if k3BatchConsumer.autoFlush {
		k3BatchConsumer.wg.Add(1)

		go func() {
			t := time.NewTicker(time.Duration(interval) * time.Second)
			defer func() {
				if r := recover(); r != nil {
					K3LogError("Auto flush goroutine panic: %v\n", r)
				}

				t.Stop()
				k3BatchConsumer.wg.Done()
			}()

			for {
				select {
				case <-t.C:
					_ = k3BatchConsumer.Flush()
				case _, ok := <-k3BatchConsumer.closed: // 处理协程退出
					if !ok {
						return
					}
				}
			}
		}()
	}

	return k3BatchConsumer, nil
}
