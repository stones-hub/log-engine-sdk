package k3

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	DefaultTimeout       = 30000 // 默认超时时间
	DefaultBatchSize     = 100   // 默认批量提交大小
	MaxBatchSize         = 1000  // 批量提交最大值
	DefaultInterval      = 5     // 默认定时检查缓存时间间隔
	DefaultCacheCapacity = 100   // 默认缓存容量
)

type K3BatchConsumer struct {
	serverURL   string        // 数据发送地址
	appId       string        // 应用ID
	timeout     time.Duration // 超时时间
	compress    bool          // 是否压缩
	bufferMutex *sync.RWMutex // buffer锁，用于Data数据在缓存中读取是否安全
	cacheMutex  *sync.RWMutex // cache锁

	buffer        []Data   // 缓存数据slice
	batchSize     int      // 批量提交大小
	cacheBuffer   [][]Data // buffer的数据会先写入 cacheBuffer中
	cacheCapacity int      // cacheBuffer的最大容量
}

// fetchBufferLength returns the length of buffer
func (k *K3BatchConsumer) fetchBufferLength() int {
	k.bufferMutex.RLocker()
	defer k.bufferMutex.RUnlock()
	return len(k.buffer)
}

// fetchCacheLength returns the length of cacheBuffer
func (k *K3BatchConsumer) fetchCacheLength() int {
	k.cacheMutex.RLocker()
	defer k.cacheMutex.RUnlock()
	return len(k.cacheBuffer)
}

// Add adds data to buffer
func (k *K3BatchConsumer) Add(data Data) error {
	k.bufferMutex.Lock()

	k.buffer = append(k.buffer, data)

	k.bufferMutex.Unlock()

	K3LogInfo("Add data to buffer, current buffer length: %d", k.fetchBufferLength())

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
		k.buffer = make([]Data, 0, k.batchSize)
	}

	// 当cacheBuffer长度大于等于 cacheCapacity，则将cacheBuffer中的数据写入server，并清空cacheBuffer
	if len(k.cacheBuffer) > k.cacheCapacity {
		// 减少一个cache buffer , 并上传
		err = k.upload()
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
		if err = k.Flush(); err != nil {
			return err
		}
	}
	return err
}

// upload uploads the first cache buffer data to the server
func (k *K3BatchConsumer) upload() error {
	var (
		buffer    []Data
		err       error
		jsonBytes []byte
		params    string
		status    int
		code      int
	)

	buffer = k.cacheBuffer[0]
	if jsonBytes, err = json.Marshal(buffer); err != nil {
		return err
	}

	params = parseTime(jsonBytes)
	if status, code, err = k.send(params, len(buffer)); err != nil {
		return err
	}
	fmt.Println(status, code)
	return err
}

// send sends the data to the server, params 需要提交的数据, bsize 数据在slice阶段的长度
func (k *K3BatchConsumer) send(params string, bsize int) (int, int, error) {
	var (
		data         string
		err          error
		compressType = "gzip"
		resp         *http.Response
		req          *http.Request
		c            *http.Client
		res          []byte
	)

	// TODO 提交给elk或类似elk的远端上传接口, 需要跟elk类似的日志存储系统对接

	if k.compress {
		data, err = CompressGzip(params)
	} else {
		data = params
		compressType = "none"
	}

	if err != nil {
		return 0, 0, err
	}

	if req, err = http.NewRequest("POST", k.serverURL, bytes.NewBufferString(data)); err != nil {
		return 0, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("compress", compressType)
	req.Header.Set("app_id", k.appId)
	req.Header.Set("size", strconv.Itoa(bsize))

	c = &http.Client{Timeout: k.timeout}
	if resp, err = c.Do(req); err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		if res, err = io.ReadAll(resp.Body); err != nil {
			return resp.StatusCode, 0, err
		}

		// 正常获取数据

		fmt.Println(res)
		return resp.StatusCode, 0, nil

	} else {
		return resp.StatusCode, -1, nil
	}
}

// Close closes the consumer
func (k *K3BatchConsumer) Close() error {
	K3LogInfo("Close K3BatchConsumer")
	return k.FlushAll()
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

// NewBatchConsumer creates a new K3BatchConsumer with default batch size.
func NewBatchConsumer(serverURL string, appId string) (K3Consumer, error) {
	return initBatchConsumer(K3BatchConsumerConfig{
		ServerURL: serverURL,
		AppId:     appId,
		Compress:  true,
	})
}

// NewBatchConsumerWithBatchSize creates a new K3BatchConsumer with batch size.
// batchSize should be between 1 and 1000.
func NewBatchConsumerWithBatchSize(serverURL string, appId string, batchSize int) (K3Consumer, error) {
	return initBatchConsumer(K3BatchConsumerConfig{
		ServerURL: serverURL,
		AppId:     appId,
		BatchSize: batchSize,
		Compress:  true,
	})
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
