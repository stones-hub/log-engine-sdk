package main

import (
	"bufio"
	"fmt"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/protocol"
	"log-engine-sdk/pkg/k3/sender"
	"os"
	"time"
)

var jString = `{"timestamp":"2024-10-09T17:41:30.703011223+08:00","log_level":"INFO","domain":"gnuser.3k.com","protocol":"HTTP/1.1","http_code":200,"log_src":"default","client_ip":"34.12.10.1","trace_id":"f0713b0d0ea4b3b5a068a808e2f38aa","org":"gnfx","project":"ywzx","code_name":"api_sdk4","event_id":1001,"event_name":"user_login","extend_data":{"uid":"2029753648","game_name":"坦克前线","amount":30,"currency":"RMB","language":"Android","version":"10.7.2","code":1001,"content":{"time_local":"2024-10-09 15:40:03","channel":"trace","content":"[trace.server]接收请求POST http://gnuser.3k.com/v5/user/login","context":{"log_type":"trace.server","url":"http://gnuser.3k.com/v5/user/login","method":"POST","input":"p=nh2%252FcdpyD","output":{"msg":"Success"},"cost_ms":49,"start_time":"2024-10-09 15:39:23.644","end_time":"2024-10-09 15:39:23.693"}}}}`

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
		EventName: "event_name",
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
		EventName: "event_name",
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

func main() {
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
