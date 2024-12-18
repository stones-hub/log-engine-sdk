package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/config"
	"log-engine-sdk/pkg/k3/watch"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	ConfigPath string // Makefile 中设置，解决config path不在项目内的问题
	Tag        string
	Version    string
	BuildTime  string
)

func main() {
	var (
		err       error
		configs   []string
		configDir string // 配置文件目录
		ctx       context.Context
	)

	k3.K3LogInfo("Start with arguments Version: %s, BuildTime: %s, Tag: %s, ConfigPath: %s\n", Version, BuildTime, Tag, ConfigPath)

	// 1. 如果ConfigPath没有设置，则使用当前目录作为配置文件目录
	if len(ConfigPath) != 0 {
		configDir = ConfigPath
	} else {

		if currentDir, err := os.Getwd(); err != nil {
			k3.K3LogError("[main] get current work dir error: %s", err)
			return
		} else {
			configDir = currentDir + "/configs"
		}
	}

	// 2. 初始化配置文件, 将配置文件的内容全部写入全局变量GlobalConfig
	if configs, err = k3.FetchDirectory(configDir, -1); err != nil {
		k3.K3LogError("fetch directory error: %s", err)
	}
	config.MustLoad(configs...)

	// 3. 初始化日志文件目录, 应用的日志目录是以工作根目录为基准的相对目录
	if len(strings.ReplaceAll(config.GlobalConfig.System.LogPath, " ", "")) == 0 {
		if currentDir, err := os.Getwd(); err != nil {
			k3.K3LogError("[main] get current work dir error: %s", err)
			return
		} else {
			config.GlobalConfig.System.LogPath = currentDir + "/log"
		}
	}

	// 4. 初始化日志记录器, 用于记录日志同K3_log一样，只是单纯的用于记录日志而已
	config.GlobalConsumer, _ = k3.NewLogConsumerWithConfig(k3.K3LogConsumerConfig{
		Directory:      config.GlobalConfig.System.LogPath,
		RoteMode:       k3.ROTATE_DAILY,
		FileSize:       1024,
		FileNamePrefix: "disk",
		ChannelSize:    1024,
	})
	defer config.GlobalConsumer.Close()

	/*
		fmt.Println("----------------------------------")
		fmt.Printf("configDir : %s\n", configDir)
		fmt.Printf("logDir : %s\n", config.GlobalConfig.System.LogPath)
		fmt.Println("----------------------------------")
	*/

	// 5. 根据配置文件设置日志等级和配置文件打印到控制台权限
	if config.GlobalConfig.System.LogLevel > 0 {
		k3.CurrentLogLevel = k3.K3LogLevel(config.GlobalConfig.System.LogLevel)
	}

	if config.GlobalConfig.System.PrintEnabled == true {
		if configJson, err := json.Marshal(config.GlobalConfig); err != nil {
			k3.K3LogError("[main] json marshal error: %s", err)
			return
		} else {
			fmt.Println(string(configJson))
		}
	}

	// 6. 遍历配置文件的监控目录，由于watch碰到子目录是不会主动监控的，所以需要子目录递归添加
	watchDirectory := make(map[string][]string)

	for indexName, dirs := range config.GlobalConfig.Watch.ReadPath {
		// 递归目录
		for _, dir := range dirs {
			if paths, err := k3.FetchDirectoryPath(dir, -1); err != nil {
				k3.K3LogError("[main] fetch directory path error: %s", err)
				continue
			} else {
				watchDirectory[indexName] = append(watchDirectory[indexName], paths...)
			}
		}
	}

	// 7. 清理可能重复的目录
	for indexName, dirs := range watchDirectory {
		watchDirectory[indexName] = k3.RemoveDuplicateElement(dirs)
	}

	k3.K3LogDebug("需要监控的目录列表: %v", watchDirectory)

	// 8. 将需要监控的目录，放入监控器中，跑起来
	if ctx, err = watch.Run(watchDirectory); err != nil {
		k3.K3LogError("[main] watch error: %s", err)
		return
	}

	var (
		httpClean func()
	)

	if config.GlobalConfig.Http.Enable == true {
		// 启动http服务器
		httpClean, _ = k3.HttpServer(context.Background())
	}

	pprof()
	graceExit(ctx, httpClean)

}

func pprof() {

	go func() {

		listener, err := net.Listen("tcp", ":6060")
		if err != nil {
			log.Fatalf("Failed to start pprof server: %v", err)
		}
		log.Println(http.Serve(listener, nil))

	}()
}

// GraceExit 保持进程常驻， 一是收到退出信号要退出， 二是协程异常退出时要退出
func graceExit(ctx context.Context, cleans ...func()) {
	var (
		state      = -1
		signalChan = make(chan os.Signal, 1)
	)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

EXIT:
	select {
	case sig, ok := <-signalChan:
		if !ok {
			k3.K3LogError("[graceExit] signal chan closed")
			break EXIT
		}
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			state = 0
			break EXIT
		case syscall.SIGHUP:
		default:
			state = 1
			break EXIT
		}
	case <-ctx.Done():
		k3.K3LogError("[graceExit] context done")
	}

	// TODO 注意回收资源
	watch.Closed()

	// TODO 退出前全量更新一次state file文件内容

	// 清理各种资源
	for _, cleanFunc := range cleans {
		if cleanFunc != nil {
			cleanFunc()
		}
	}
	time.Sleep(1 * time.Second)
	os.Exit(state)
}
