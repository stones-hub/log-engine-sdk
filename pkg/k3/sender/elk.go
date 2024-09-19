package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/config"
	"log-engine-sdk/pkg/k3/protocol"
	"strings"
	"time"
)

var (
	DefaultMaxChannelSize = 1000
)

type ElasticSearchClient struct {
	config   elasticsearch.Config
	client   *elasticsearch.Client
	dataChan chan *protocol.Data // 最大队列
}

func NewElasticsearch(address []string, username, password string) (*ElasticSearchClient, error) {

	if config.GlobalConfig.ELK.MaxChannelSize >= DefaultMaxChannelSize {
		config.GlobalConfig.ELK.MaxChannelSize = DefaultMaxChannelSize
	}

	return NewElasticsearchWithConfig(config.ELK{
		Address:        address,
		Username:       username,
		Password:       password,
		MaxChannelSize: config.GlobalConfig.ELK.MaxChannelSize,
	})
}

func NewElasticsearchWithConfig(elasticsearchConfig config.ELK) (*ElasticSearchClient, error) {
	var (
		cfg    elasticsearch.Config
		client *elasticsearch.Client
		err    error
	)

	cfg = elasticsearch.Config{
		Addresses: elasticsearchConfig.Address,
		Username:  elasticsearchConfig.Username,
		Password:  elasticsearchConfig.Password,
	}

	if client, err = elasticsearch.NewClient(cfg); err != nil {
		k3.K3LogError("Failed to create Elasticsearch client: %v", err)
		return nil, err
	}

	// 开启协程，从管道中读取数据， 写入集群

	c := &ElasticSearchClient{
		config:   cfg,
		client:   client,
		dataChan: make(chan *protocol.Data, elasticsearchConfig.MaxChannelSize),
	}

	go WriteDataToElasticSearch(c)

	return c, nil
}

// WriteDataToElasticSearch 从管道读取数据，写入elk
func WriteDataToElasticSearch(client *ElasticSearchClient) {

	for {

		var (
			b   []byte
			err error
			req esapi.IndexRequest
			res *esapi.Response
			e   map[string]interface{}
		)

		select {
		case data := <-client.dataChan:
			// 解析数据
			if b, err = json.Marshal(data); err != nil {
				k3.K3LogError("Failed to marshal data: %v", err)
				continue
			}

			req = esapi.IndexRequest{
				Index:      data.EventName,
				DocumentID: fmt.Sprintf("%s", data.UUID),
				Body:       strings.NewReader(string(b)),
				Pretty:     true,
			}

			if res, err = req.Do(context.Background(), client.client); err != nil {
				k3.K3LogError("Failed to send data to Elasticsearch: %v", err)
				continue
			}

			if res.IsError() {
				if err = json.NewDecoder(res.Body).Decode(&e); err != nil {
					fmt.Printf("Error parsing the response body: %s", err)
					continue
				} else {
					fmt.Printf("Error response: %s: %s\n", res.Status(), e["error"])
					continue
				}
			}
			// TODO 考虑循环结束一次退出，会不会有问题，需要测试
			res.Body.Close()
		}
	}

}

func (e *ElasticSearchClient) Close() {

	close(e.dataChan)
	// TODO 关闭客户端

}

func (e *ElasticSearchClient) Send(data []protocol.Data) error {

	// 循环发送数据
	for _, d := range data {
		select {
		// TODO 测试一下当管道满了， default语句continue的时候，会不会丢弃当前循环的d数据
		case e.dataChan <- &d:
		default:
			k3.K3LogInfo("Data channel is full, data will be discarded.")
			time.Sleep(1 * time.Second)
			continue
		}
	}
	return nil
}

func (e *ElasticSearchClient) prepareBulkData(data []protocol.Data) ([]byte, error) {
	var bulkData bytes.Buffer

	for _, d := range data {
		jsonData, err := json.Marshal(d)
		if err != nil {
			return nil, err
		}

		bulkData.WriteString(fmt.Sprintf(`{"index":{"_id":"%s"}}\n`, d.UUID))
		bulkData.Write(jsonData)
		bulkData.WriteString("\n")
	}
	return bulkData.Bytes(), nil
}
