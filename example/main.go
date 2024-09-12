package main

import (
	"fmt"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/protocol"
	"log-engine-sdk/pkg/k3/sender"
	"time"
)

func main() {
	var (
		ips      []string
		elk      *sender.ELK
		err      error
		consumer protocol.K3Consumer
	)

	if elk, err = sender.NewELK([]string{"http://127.0.0.1:9200"}); err != nil {
		fmt.Println(err)
		return
	}

	consumer, err = k3.NewBatchConsumerWithConfig(k3.K3BatchConsumerConfig{
		Sender:    elk,
		AutoFlush: true,
	})

	if ips, err = k3.GetLocalIPs(); err != nil {
		fmt.Println(err)
		return
	}

	consumer.Add(protocol.Data{
		AccountId: "1001",
		AppId:     "appid-1001",
		Time:      time.Now().Format("2006-01-02 15:04:05"),
		EventName: "batch_test",
		EventId:   "batch_test_id",
		Ip:        ips[0],
		UUID:      k3.GenerateUUID(),
		Properties: map[string]interface{}{
			"user_name": "stones",
			"age":       18,
		},
	})

	consumer.Close()
}
