package k3

import (
	"encoding/json"
	"fmt"
	"log-engine-sdk/pkg/k3/protocol"
	"testing"
)

type Default struct {
}

func (d *Default) Send(data []protocol.Data) error {
	var (
		b   []byte
		err error
	)
	if b, err = json.Marshal(data); err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func (d *Default) Close() error {
	fmt.Println("close default sender")
	return nil
}

func TestDataAnalytics(t *testing.T) {

	var (
		dataAnalytics DataAnalytics
		err           error
		consumer      protocol.K3Consumer
	)

	if consumer, err = NewBatchConsumerWithConfig(K3BatchConsumerConfig{
		Sender:    new(Default),
		AutoFlush: true,
	}); err != nil {
		t.Error(err)
		return
	}

	// TODO 2544-05-10 08:41:20, 将截止时间改为可配置, 尽量配置大一点

	dataAnalytics = NewDataAnalytics(consumer)
	dataAnalytics.SetSuperProperties(map[string]interface{}{"user": "yelei", "age": 12})
	dataAnalytics.Track("account_id", "app_id", "ip", "1001", map[string]interface{}{"name": "stones", "age": 111})
	dataAnalytics.Track("account_id", "app_id", "ip", "1001", map[string]interface{}{"name": "stones", "age": 112})
	dataAnalytics.Track("account_id", "app_id", "ip", "1001", map[string]interface{}{"name": "stones", "age": 113})
	dataAnalytics.Track("account_id", "app_id", "ip", "1001", map[string]interface{}{"name": "stones", "age": 114})
	dataAnalytics.Close()

}
