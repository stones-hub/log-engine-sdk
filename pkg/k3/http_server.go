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
	status.Alloc = memStats.Alloc / 1024
	status.TotalAlloc = memStats.TotalAlloc / 1024
	status.Sys = memStats.Sys / 1024
	status.NumGC = memStats.NumGC

	if b, err = json.Marshal(status); err != nil {
		w.Write([]byte(err.Error()))
	} else {
		w.Write(b)
	}

}

// TODO
// 当前程序所占内存指标
// 当前管道队列长度
// 写入ELK累计失败数
// 写入ELK累计写入数

var (
	GlobalWriteFailedCount  int
	GlobalWriteSuccessCount int
)

type Status struct {
	Alloc      uint64 // 当前已分配的内存 KB
	TotalAlloc uint64 // 程序运行以来总共分配的内存字节数 KB。
	Sys        uint64 // 操作系统获取的总内存量 KB
	NumGC      uint32 // 表示垃圾回收的次
}
