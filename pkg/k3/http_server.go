package k3

import (
	"context"
	"fmt"
	"log-engine-sdk/pkg/k3/config"
	"net/http"
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

// FindStatusRouter
// TODO  获取Reqeust URI 的信息, 来决定业务逻辑
func FindStatusRouter(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello world"))
}
