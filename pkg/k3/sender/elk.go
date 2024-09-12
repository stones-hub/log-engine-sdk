package sender

import (
	"encoding/json"
	"fmt"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"log-engine-sdk/pkg/k3/protocol"
)

// TODO 目前只做一个伪装实现，后续补充，方便测试

type ELK struct {
	config elasticsearch.Config
	client elastictransport.Client
}

func NewELK(address []string) (*ELK, error) {
	return NewELKWithConfig(protocol.ELKConfig{
		Address: address,
	})
}

// todo 后续根据情况调整
func NewELKWithConfig(elkConfig protocol.ELKConfig) (*ELK, error) {
	return nil, nil
}

func (e *ELK) Send(data []protocol.Data) error {
	var (
		jsonData []byte
		err      error
	)
	if jsonData, err = json.Marshal(data); err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

/*
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type Data struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Timestamp string    `json:"timestamp"`
	Properties map[string]interface{} `json:"properties"`
}

func main() {
	// 配置 Elasticsearch 客户端
	esConfig := elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
		// 可以添加更多配置项，如用户名、密码等
		// Username: "username",
		// Password: "password",
	}

	// 创建 Elasticsearch 客户端
	esClient, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	// 定义要插入的数据
	data := []Data{
		{ID: 1, Name: "Alice", Timestamp: "2023-09-01T12:00:00Z", Properties: map[string]interface{}{"age": 25}},
		{ID: 2, Name: "Bob", Timestamp: "2023-09-01T13:00:00Z", Properties: map[string]interface{}{"age": 30}},
		{ID: 3, Name: "Charlie", Timestamp: "2023-09-01T14:00:00Z", Properties: map[string]interface{}{"age": 35}},
		{ID: 4, Name: "David", Timestamp: "2023-09-01T15:00:00Z", Properties: map[string]interface{}{"age": 40}},
		{ID: 5, Name: "Eve", Timestamp: "2023-09-01T16:00:00Z", Properties: map[string]interface{}{"age": 45}},
	}

	// 批量插入数据
	bulkData, err := prepareBulkData(data)
	if err != nil {
		log.Fatalf("Error preparing bulk data: %s", err)
	}

	// 构建批量请求
	bulkReq := esapi.BulkRequest{
		Body:          ioutil.NopCloser(bytes.NewReader(bulkData)),
		Index:         "example_index",
		Refresh:       "true",
		ContentType:   "application/x-ndjson",
		MaxRetries:    3,
		RetryOnStatus: []int{502, 503, 504},
	}

	// 发送批量请求
	res, err := bulkReq.Do(context.Background(), esClient)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := ioutil.ReadAll(res.Body)
		log.Fatalf("Error indexing documents: %s", body)
	} else {
		fmt.Println("Documents indexed successfully with status code", res.StatusCode)
	}
}

func prepareBulkData(data []Data) ([]byte, error) {
	var bulkData bytes.Buffer

	for _, d := range data {
		jsonData, err := json.Marshal(d)
		if err != nil {
			return nil, err
		}

		bulkData.WriteString(fmt.Sprintf(`{"index":{"_id":"%d"}}\n`, d.ID))
		bulkData.Write(jsonData)
		bulkData.WriteString("\n")
	}

	return bulkData.Bytes(), nil
}

*/
