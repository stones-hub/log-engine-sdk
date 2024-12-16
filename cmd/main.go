package main

import (
	"encoding/json"
	"fmt"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/config"
	"log-engine-sdk/pkg/k3/watch"
	_ "net/http/pprof"
	"os"
	"strings"
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
	if err = watch.Run(watchDirectory); err != nil {
		k3.K3LogError("[main] watch error: %s", err)
		return
	}

	os.Exit(0)

	/*
		if config.GlobalConfig.Http.Enable == true {
			// 启动http服务器
			httpClean, _ = k3.HttpServer(context.Background())
		}

		pprof()

		var (
			httpClean func()
		)
		graceExit(dir+config.GlobalConfig.Watch.StateFilePath, httpClean)

	*/
}

/*
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


*/
