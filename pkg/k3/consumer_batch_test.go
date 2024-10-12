package k3

import (
	"fmt"
	"log-engine-sdk/pkg/k3/protocol"
	"testing"
	"time"
)

func TestBatchConsumer(t *testing.T) {
	var (
		ips      []string
		err      error
		consumer protocol.K3Consumer
	)

	consumer, err = NewBatchConsumerWithConfig(K3BatchConsumerConfig{
		Sender:    new(Default),
		AutoFlush: true,
	})

	if ips, err = GetLocalIPs(); err != nil {
		fmt.Println(err)
		return
	}

	consumer.Add(protocol.Data{
		UUID:      GenerateUUID(),
		AccountId: "1001",
		AppId:     "appid-1001",
		Ip:        ips[0],
		Timestamp: time.Now(),
		EventName: "1001",
		Properties: map[string]interface{}{
			"user_name": "stones",
			"age":       18,
		},
	})

	consumer.Add(protocol.Data{
		UUID:      GenerateUUID(),
		AccountId: "1002",
		AppId:     "appid-1001",
		Ip:        ips[0],
		Timestamp: time.Now(),
		EventName: "1001",
		Properties: map[string]interface{}{
			"user_name": "stones",
			"age":       18,
		},
	})

	consumer.Add(protocol.Data{
		UUID:      GenerateUUID(),
		AccountId: "1003",
		AppId:     "appid-1001",
		Ip:        ips[0],
		Timestamp: time.Now(),
		EventName: "1001",
		Properties: map[string]interface{}{
			"user_name": "stones",
			"age":       18,
		},
	})
	consumer.Close()

}
