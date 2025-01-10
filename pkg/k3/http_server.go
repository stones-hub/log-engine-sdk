package k3

import (
	"context"
	"encoding/json"
	"fmt"
	"log-engine-sdk/pkg/k3/config"
	"net/http"
	"runtime"
	"time"
)

// HttpServer http服务器， 用户对进程状态的查询, 这个系统并不是专业web服务，只需考虑简单的系统内部信息的查询
func HttpServer(ctx context.Context) (func(), error) {
	var (
		addr string
		mux  *http.ServeMux
	)
	addr = fmt.Sprintf("%s:%d", config.GlobalConfig.Http.Host, config.GlobalConfig.Http.Port)

	mux = http.NewServeMux()
	mux.HandleFunc("/status", FindStatusRouter)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Duration(config.GlobalConfig.Http.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.GlobalConfig.Http.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(config.GlobalConfig.Http.IdleTimeout) * time.Second,
	}

	go func() {

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			K3LogError("http server error: %s", err.Error)
			panic(err)
		}
	}()

	return func() {
		K3LogInfo("http server will been shutdown . ")
		timeoutCTX, cancel := context.WithTimeout(ctx, time.Duration(config.GlobalConfig.Http.ShutdownTimeout)*time.Second)
		defer cancel()
		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(timeoutCTX); err != nil {
			K3LogError("http server shutdown error: %s", err.Error)
		}
	}, nil
}

// FindStatusRouter 查询当前进程状态
func FindStatusRouter(w http.ResponseWriter, r *http.Request) {

	var (
		memStats runtime.MemStats
		status   Status
		b        []byte
		err      error
	)

	runtime.ReadMemStats(&memStats)
	status.Alloc = memStats.Alloc / 1024 / 1024
	status.TotalAlloc = memStats.TotalAlloc / 1024 / 1024
	status.Sys = memStats.Sys / 1024 / 1024
	status.NumGC = memStats.NumGC
	status.HeapAlloc = memStats.HeapAlloc / 1024 / 1024
	status.WriteToChannelFailedCount = GlobalWriteToChannelFailedCount
	status.WriteSuccessCount = GlobalWriteSuccessCount
	status.WriteFailedCount = GlobalWriteFailedCount

	if b, err = json.Marshal(status); err != nil {
		_, _ = w.Write([]byte(err.Error()))
	} else {
		_, _ = w.Write(b)
	}
}

var (
	GlobalWriteFailedCount          int
	GlobalWriteSuccessCount         int
	GlobalWriteToChannelFailedCount int
)

type Status struct {
	Alloc                     uint64 `json:"alloc"`                         // 当前已分配的内存 KB
	TotalAlloc                uint64 `json:"total_alloc"`                   // 程序运行以来总共分配的内存字节数 KB。
	HeapAlloc                 uint64 `json:"heap_alloc"`                    // 当前实际分配的堆内存大小，如果这个值持续增长，则可能存在内存泄漏。
	Sys                       uint64 `json:"sys"`                           // 操作系统获取的总内存量 KB
	NumGC                     uint32 `json:"num_gc"`                        // 表示垃圾回收的次
	WriteFailedCount          int    `json:"write_failed_count"`            // 写入ELK失败条数
	WriteSuccessCount         int    `json:"write_success_count"`           // 写入ELK成功条数
	WriteToChannelFailedCount int    `json:"write_to_channel_failed_count"` // 写入缓存失败条数
}
