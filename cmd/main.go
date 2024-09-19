package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/config"
	"log-engine-sdk/pkg/k3/watch"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var (
		dir     string
		err     error
		configs []string
	)
	// 初始化配置文件, 必须通过make运行
	if dir, err = os.Getwd(); err != nil {
		k3.K3LogError("get current dir error: %s", err)
		return
	}

	// 初始化日志记录器, 用于记录日志同K3_log一样，只是单纯的用于记录日志而已
	config.GlobalConsumer, _ = k3.NewLogConsumerWithConfig(k3.K3LogConsumerConfig{
		Directory:      dir + "/log",
		RoteMode:       k3.ROTATE_DAILY,
		FileSize:       1024,
		FileNamePrefix: "disk",
		ChannelSize:    1024,
	})
	defer config.GlobalConsumer.Close()

	// 获取configs文件目录所有文件
	if configs, err = k3.FetchDirectory(dir+"/configs", -1); err != nil {
		k3.K3LogError("fetch directory error: %s", err)
	}
	config.MustLoad(configs...)

	// 注入RootPath
	if len(config.GlobalConfig.System.RootPath) == 0 {
		config.GlobalConfig.System.RootPath = dir
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

	for _, readDir := range config.GlobalConfig.System.ReadPath {
		ReadDirectory = append(ReadDirectory, readDir)
	}

	err = watch.Run(ReadDirectory, dir+config.GlobalConfig.System.StateFilePath)

	if err != nil {
		k3.K3LogError("watch error: %s", err)
		return
	}

	if config.GlobalConfig.Http.Enable == true {
		// 启动http服务器
		httpClean, _ = k3.HttpServer(context.Background())
	}
	graceExit(dir+config.GlobalConfig.System.StateFilePath, httpClean)
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
	watch.Clean()

	// 退出前全量更新一次state file文件内容
	if err = watch.SyncToSateFile(stateFile); err != nil {
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
