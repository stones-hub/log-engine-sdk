package main

import (
	"context"
	"fmt"
	"time"
)

// 模拟一个长时间运行的任务
func longRunningTask(ctx context.Context) error {
	fmt.Println("开始执行长时间运行的任务...")

	// 模拟一个长时间运行的任务
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case <-ctx.Done():
			fmt.Println("任务被取消")
			return ctx.Err()
		default:
			fmt.Println("仍在执行任务...")
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Println("任务完成")
	return nil
}

func main() {
	// 创建一个基础的上下文
	ctx := context.Background()

	// 使用 WithTimeout 创建一个带有超时时间的上下文
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	<-ctxWithTimeout.Done()

	fmt.Println("aaaaa")

	// 调用 longRunningTask 函数
	// err := longRunningTask(ctxWithTimeout)

}
