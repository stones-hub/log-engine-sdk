package main

import (
	"bufio"
	"fmt"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/protocol"
	"log-engine-sdk/pkg/k3/sender"
	_ "net/http/pprof"
	"os"
	"time"
)

var jString = `{"time_local":"2024-10-12 16:43:47","trace_id":"ecab3e28ac94df49caf298d2204a242f","hostname":"ali-gnfx-admin-center-02","channel":"trace","log_level":"INFO","context":"{\"源数据\":\"app_id=2fa4cd360802b096&appid=mp_api&ip=14.18.194.140&passport_token=ee94e0af5993e9dbcfaf0b09a0a94702&timestamp=1728722627&secret=0b746e4165cba2e8758d2594cf91f006\",\"签名\":\"7B153732E4C5D8F8865901DD75100778\"}","content":"签名信息"}`

func TestK3Log() {
	k3.K3LogError("test: %s", "err")
	k3.K3LogInfo("test: %s", "info")
	k3.K3LogWarn("test: %s", "warn")
	k3.K3LogDebug("test: %s", "debug")
}

func TestConsumerLog() {
	consumerLog, err := k3.NewLogConsumerWithConfig(k3.K3LogConsumerConfig{
		Directory:      "log/",
		RoteMode:       k3.ROTATE_DAILY,
		FileSize:       1024,
		FileNamePrefix: "test",
		ChannelSize:    1024,
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	consumerLog.Add(protocol.Data{
		UUID:      k3.GenerateUUID(),
		AccountId: "1001",
		AppId:     "1001-01",
		Ip:        "127.0.0.1",
		Timestamp: time.Now(),
		IndexName: "index_name",
		Properties: map[string]interface{}{
			"property": "test",
		},
	})

	consumerLog.Close()
}

func TestConsumerBatchLog() {

	batchConsumer, err := k3.NewBatchConsumerWithConfig(k3.K3BatchConsumerConfig{
		Sender:        new(sender.Default),
		BatchSize:     10,    // 批量日志单次批量提交最大值
		AutoFlush:     false, // 是否自动刷新
		Interval:      5,     // 批量日志检查缓存列表时间间隔
		CacheCapacity: 100,   // 批量日志缓存容量
	})

	if err != nil {
		return
	}

	batchConsumer.Add(protocol.Data{
		UUID:      k3.GenerateUUID(),
		AccountId: "1001",
		AppId:     "1001-01",
		Ip:        "127.0.0.1",
		Timestamp: time.Now(),
		IndexName: "index_name",
		Properties: map[string]interface{}{
			"property": "batch test",
		},
	})
	batchConsumer.Close()
}

func TestDataAnalytics() {

	var (
		dataAnalytics k3.DataAnalytics
		err           error
		consumer      protocol.K3Consumer
	)

	if consumer, err = k3.NewBatchConsumerWithConfig(k3.K3BatchConsumerConfig{
		Sender:    new(sender.Default),
		AutoFlush: true,
	}); err != nil {
		fmt.Println(err.Error())
		return
	}

	dataAnalytics = k3.NewDataAnalytics(consumer)
	dataAnalytics.SetSuperProperties(map[string]interface{}{"user": "yelei", "age": 12})
	dataAnalytics.Track("account_id", "app_id", "ip", "1001", map[string]interface{}{"name": "stones", "age": 111})
	dataAnalytics.Track("account_id", "app_id", "ip", "1002", map[string]interface{}{"name": "stones", "age": 112})
	dataAnalytics.Track("account_id", "app_id", "ip", "1003", map[string]interface{}{"name": "stones", "age": 113})
	dataAnalytics.Track("account_id", "app_id", "ip", "1004", map[string]interface{}{"name": "stones", "age": 114})
	dataAnalytics.Close()
}

// TotalTestLog 测试日志
func TotalTestLog() {
	TestK3Log()
	TestConsumerLog()
	TestConsumerBatchLog()
	TestDataAnalytics()
}

type TestData struct {
	Flag bool
}

func main() {
	logger, err := k3.NewLogger("/Users/yelei/data/code/go-projects/log-engine-sdk/logs", 0, "test", 1024*1024*1024, 10, 0)
	fmt.Println(err)
	for i := 0; i < 20; i++ {
		logger.Add("123131321231231")
	}

	logger2, err2 := k3.NewLogger("/Users/yelei/data/code/go-projects/log-engine-sdk/logs", 0, "test", 1024*1024*1024, 10, 0)
	fmt.Println(err2)
	for i := 0; i < 30; i++ {
		logger2.Add("111111111111")
	}

	logger.Close()
	logger2.Close()
}

func TestAddData() {

	var (
		count         = 10
		nginxFile     = "/Users/yelei/data/code/go-projects/logs/nginx/nginx.log"
		adminFile     = "/Users/yelei/data/code/go-projects/logs/admin/admin.log"
		apiFile       = "/Users/yelei/data/code/go-projects/logs/api/api.log"
		fd1, fd2, fd3 *os.File
		err           error
	)

	for count > 0 {
		if fd1, err = os.OpenFile(nginxFile, os.O_RDWR|os.O_APPEND, 0666); err != nil {
			fmt.Println(err)
			return
		} else {
			writer1 := bufio.NewWriter(fd1)
			writer1.WriteString(jString + "\n")
			writer1.Flush()
		}

		fd1.Close()

		if fd2, err = os.OpenFile(adminFile, os.O_RDWR|os.O_APPEND, 0666); err != nil {
			fmt.Println(err)
			return
		} else {
			writer2 := bufio.NewWriter(fd2)
			writer2.WriteString(jString + "\n")
			writer2.Flush()
		}

		fd2.Close()

		if fd3, err = os.OpenFile(apiFile, os.O_RDWR|os.O_APPEND, 0666); err != nil {
			fmt.Println(err)
			return
		} else {
			writer3 := bufio.NewWriter(fd3)
			writer3.WriteString(jString + "\n")
			writer3.Flush()
		}

		fd3.Close()

		count--
		time.Sleep(1 * time.Second)
	}
}
