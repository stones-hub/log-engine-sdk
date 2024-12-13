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
	LogPath    string // Makefile 中设置，解决log path不在项目内的问题
	Tag        string
	Version    string
	BuildTime  string
)

func main() {
	var (
		dir       string
		err       error
		configs   []string
		configDir string
		logDir    string
	)

	k3.K3LogInfo("Start with arguments Version: %s, BuildTime: %s, Tag: %s, ConfigPath: %s\n", Version, BuildTime, Tag, ConfigPath)

	// 1. 如果ConfigPath没有设置，则使用当前目录作为配置文件目录
	if len(ConfigPath) != 0 {
		configDir = ConfigPath
	} else {

		if dir, err = os.Getwd(); err != nil {
			k3.K3LogError("[main] get current work dir error: %s", err)
			return
		}
		configDir = dir + "/configs"
	}

	// 2. 如果LogPath没有设置，则使用当前目录作为日志文件目录

	// 2. 初始化配置文件
	if configs, err = k3.FetchDirectory(configDir, -1); err != nil {
		k3.K3LogError("fetch directory error: %s", err)
	}
	config.MustLoad(configs...)

	// 3. 初始化日志记录器, 用于记录日志同K3_log一样，只是单纯的用于记录日志而已
	config.GlobalConsumer, _ = k3.NewLogConsumerWithConfig(k3.K3LogConsumerConfig{
		Directory:      config.GlobalConfig.System.RootPath + "/log",
		RoteMode:       k3.ROTATE_DAILY,
		FileSize:       1024,
		FileNamePrefix: "disk",
		ChannelSize:    1024,
	})
	defer config.GlobalConsumer.Close()

	/*
		fmt.Println("----------------------------------")
		fmt.Printf("configDir : %s\n", configDir)
		fmt.Println("----------------------------------")

	*/

	if config.GlobalConfig.System.LogLevel > 0 {
		k3.CurrentLogLevel = k3.K3LogLevel(config.GlobalConfig.System.LogLevel)
	}

	if config.GlobalConfig.System.PrintEnabled == true {
		if configJson, err := json.Marshal(config.GlobalConfig); err != nil {
			k3.K3LogError("json marshal error: %s", err)
			return
		} else {
			fmt.Println(string(configJson))
		}
	}

	var (
		ReadDirectory []string
		httpClean     func()
	)

	// 遍历需要监控的目录
	for _, readDirs := range config.GlobalConfig.Watch.ReadPath {
		for _, readDir := range readDirs {
			if !strings.HasSuffix(readDir, "/") {
				readDir = readDir + "/"
			}
			ReadDirectory = append(ReadDirectory, readDir)
		}
	}

	// 剔除ReadDirectory中的重复目录
	ReadDirectory = k3.RemoveDuplicateElement(ReadDirectory)

	// err = watch.Run(ReadDirectory, dir+config.GlobalConfig.Watch.StateFilePath)
	err = watch.Run()

	if err != nil {
		k3.K3LogError("watch error: %s", err)
		return
	}

	if config.GlobalConfig.Http.Enable == true {
		// 启动http服务器
		httpClean, _ = k3.HttpServer(context.Background())
	}

	pprof()
	graceExit(dir+config.GlobalConfig.Watch.StateFilePath, httpClean)
}

// GraceExit 保持进程常驻， 等待信号在退出
func graceExit(stateFile string, cleanFuncs ...func()) {
	var (
		state      = -1
		signalChan = make(chan os.Signal, 1)
		err        error
	)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	{
	EXIT:
		select {
		case sig, ok := <-signalChan:
			if !ok {
				// 直接退出，关闭
				break EXIT
			}
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				state = 0
				break EXIT
			case syscall.SIGHUP:
			default:
				state = -1
				break EXIT
			}
		}
	}

	// 关闭资源退出, 清理 watch 和 batch 日志提交资源
	watch.Close()

	// 退出前全量更新一次state file文件内容
	if err = watch.ForceSyncFileState(); err != nil {
		k3.K3LogError("Closed watcher run save stateFile error: %s", err)
	}

	// 清理各种资源
	for _, cleanFunc := range cleanFuncs {
		if cleanFunc != nil {
			cleanFunc()
		}
	}

	time.Sleep(1 * time.Second)
	os.Exit(state)
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
