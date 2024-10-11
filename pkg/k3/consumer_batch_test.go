package k3

import (
	"fmt"
	"log-engine-sdk/pkg/k3/protocol"
	"log-engine-sdk/pkg/k3/sender"
	"testing"
	"time"
)

func TestBatchConsumer(t *testing.T) {
	var (
		ips      []string
		elk      *sender.ElasticSearchClient
		err      error
		consumer protocol.K3Consumer
	)

	if elk, err = sender.NewElasticsearch([]string{"http://127.0.0.1:8080"}, "", ""); err != nil {
		fmt.Println(err)
		return
	}

	consumer, err = NewBatchConsumerWithConfig(K3BatchConsumerConfig{
		Sender:    elk,
		AutoFlush: true,
	})

	if ips, err = GetLocalIPs(); err != nil {
		fmt.Println(err)
		return
	}

	consumer.Add(protocol.Data{
		AccountId: "1001",
		AppId:     "appid-1001",
		Time:      time.Now().Format("2006-01-02 15:04:05"),
		Ip:        ips[0],
		UUID:      GenerateUUID(),
		Properties: map[string]interface{}{
			"user_name": "stones",
			"age":       18,
		},
	})

	consumer.Add(protocol.Data{
		AccountId: "1002",
		AppId:     "appid-1001",
		Time:      time.Now().Format("2006-01-02 15:04:05"),
		Ip:        ips[0],
		UUID:      GenerateUUID(),
		Properties: map[string]interface{}{
			"user_name": "stones",
			"age":       18,
		},
	})

	consumer.Add(protocol.Data{
		AccountId: "1003",
		AppId:     "appid-1001",
		Time:      time.Now().Format("2006-01-02 15:04:05"),
		Ip:        ips[0],
		UUID:      GenerateUUID(),
		Properties: map[string]interface{}{
			"user_name": "stones",
			"age":       18,
		},
	})

	consumer.Close()
}
