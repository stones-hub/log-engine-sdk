package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/config"
	"log-engine-sdk/pkg/k3/watch"
	"os"
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
	)

	for _, readDir := range config.GlobalConfig.System.ReadPath {
		ReadDirectory = append(ReadDirectory, readDir)
	}

	err = watch.Run(ReadDirectory, dir+config.GlobalConfig.System.StateFilePath)

	if err != nil {
		k3.K3LogError("watch error: %s", err)
		return
	}

	// 启动http服务器
	httpClean, _ := k3.HttpServer(context.Background())
	k3.GraceExit(dir+config.GlobalConfig.System.StateFilePath, httpClean)
}
