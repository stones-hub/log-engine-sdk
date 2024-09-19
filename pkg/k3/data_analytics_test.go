package k3

import (
	"fmt"
	"log-engine-sdk/pkg/k3/protocol"
	"log-engine-sdk/pkg/k3/sender"
	"testing"
)

func TestDataAnalytics(t *testing.T) {

	var (
		dataAnalytics DataAnalytics
		elk           *sender.ElasticSearchClient
		err           error
		consumer      protocol.K3Consumer
	)

	if elk, err = sender.NewElasticsearch([]string{"http://127.0.0.1:9200"}, "admin", "admin"); err != nil {
		fmt.Println(err)
		return
	}

	consumer, err = NewBatchConsumerWithConfig(K3BatchConsumerConfig{
		Sender:    elk,
		AutoFlush: true,
	})

	dataAnalytics = NewDataAnalytics(consumer)

	dataAnalytics.SetSuperProperties(map[string]interface{}{"user": "yelei", "age": 12})

	dataAnalytics.Track("account_id", "app_id", "event_name", "event_id", "ip", map[string]interface{}{"name": "stones", "age": 111})
	dataAnalytics.Track("account_id", "app_id", "event_name", "event_id", "ip", map[string]interface{}{"name": "stones", "age": 112})
	dataAnalytics.Track("account_id", "app_id", "event_name", "event_id", "ip", map[string]interface{}{"name": "stones", "age": 113})
	dataAnalytics.Track("account_id", "app_id", "event_name", "event_id", "ip", map[string]interface{}{"name": "stones", "age": 114})

	dataAnalytics.Close()
}
