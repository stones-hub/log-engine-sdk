package k3

import (
	"log-engine-sdk/pkg/k3/watch"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ForceExit 强制退出当前程序
func ForceExit() {
	_ = syscall.Kill(os.Getpid(), syscall.SIGHUP)
}

// GraceExit 保持进程常驻， 等待信号在退出
func GraceExit(stateFile string, cleanFuncs ...func()) {
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

	// 关闭资源退出
	watch.Clean()

	// 退出前全量更新一次state file文件内容
	if err = watch.SyncToSateFile(stateFile); err != nil {
		K3LogError("Closed watcher run save stateFile error: %s", err)
	}

	// 清理各种资源
	for _, cleanFunc := range cleanFuncs {
		cleanFunc()
	}

	time.Sleep(1 * time.Second)
	os.Exit(state)
}
