package k3

import (
	"fmt"
	"log-engine-sdk/pkg/k3/protocol"
	"sync"
	"testing"
	"time"
)

func TestConsumerLog(t *testing.T) {
	var wg sync.WaitGroup

	consumerLog, err := NewLogConsumerWithFileSize("logs", ROTATE_DAILY, 1024)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer consumerLog.Close()

	ips, err := GetLocalIPs()
	if err != nil {
		fmt.Println(err)
		return
	}

	wg.Add(1)

	go func() {
		total := 10
		t := time.NewTicker(time.Second * 3)
		defer func() {
			t.Stop()
			wg.Done()
		}()

		for total > 0 {
			select {
			case <-t.C:
				err = consumerLog.Add(protocol.Data{
					UUID:      GenerateUUID(),
					AccountId: "1001",
					AppId:     "app_id_1001",
					Ip:        ips[0],
					Timestamp: time.Now(),
					EventName: "",
					Properties: map[string]interface{}{
						"user_name": "stones",
						"age":       18,
					},
				})
				if err != nil {
					fmt.Println(err)

				}
				total--
			}
		}
	}()

	wg.Wait()
	consumerLog.Flush()
}
