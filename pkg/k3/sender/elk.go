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
	"os"
	"strings"
	"sync"
	"time"
)

var (
	DefaultMaxChannelSize = 10000 // 队列管道的最大长度
	DefaultMaxRetry       = 10    // 重试次数
	DefaultTimeout        = 30    // 秒, 数据发送的超时时间
	DefaultRetryInterval  = 3     // 秒， 默认队列满等待时间间隔
)

type ElasticSearchClient struct {
	config        elasticsearch.Config
	client        *elasticsearch.Client
	dataChan      chan *protocol.Data // 最大队列
	maxRetries    int                 // 最大重试次数
	retryInterval int                 // 每次重试时间间隔
	timeout       int                 // 最后超时时间
	sg            *sync.WaitGroup
}

func NewElasticsearch(address []string, username, password string) (*ElasticSearchClient, error) {

	if config.GlobalConfig.ELK.MaxChannelSize == 0 || config.GlobalConfig.ELK.MaxChannelSize >= DefaultMaxChannelSize {
		config.GlobalConfig.ELK.MaxChannelSize = DefaultMaxChannelSize
	}

	if config.GlobalConfig.ELK.MaxRetry == 0 || config.GlobalConfig.ELK.MaxRetry >= DefaultMaxRetry {
		config.GlobalConfig.ELK.MaxRetry = DefaultMaxRetry
	}

	if config.GlobalConfig.ELK.RetryInterval == 0 || config.GlobalConfig.ELK.RetryInterval >= DefaultRetryInterval {
		config.GlobalConfig.ELK.RetryInterval = DefaultRetryInterval
	}

	if config.GlobalConfig.ELK.Timeout == 0 || config.GlobalConfig.ELK.Timeout >= DefaultTimeout {
		config.GlobalConfig.ELK.Timeout = DefaultTimeout
	}

	return NewElasticsearchWithConfig(config.ELK{
		Address:        address,
		Username:       username,
		Password:       password,
		MaxChannelSize: config.GlobalConfig.ELK.MaxChannelSize,
		MaxRetry:       config.GlobalConfig.ELK.MaxRetry,
		RetryInterval:  config.GlobalConfig.ELK.RetryInterval,
		Timeout:        config.GlobalConfig.ELK.Timeout,
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
		config:        cfg,
		client:        client,
		dataChan:      make(chan *protocol.Data, elasticsearchConfig.MaxChannelSize),
		maxRetries:    elasticsearchConfig.MaxRetry,
		retryInterval: elasticsearchConfig.RetryInterval,
		timeout:       elasticsearchConfig.Timeout,
		sg:            &sync.WaitGroup{},
	}

	c.sg.Add(1)
	go WriteDataToElasticSearch(c)

	return c, nil
}

// WriteDataToElasticSearch 从管道读取数据，写入elk
func WriteDataToElasticSearch(client *ElasticSearchClient) {

	defer func() {
		if r := recover(); r != nil {
			k3.K3LogError("Recovered WriteDataToElasticSearch from panic: %v", r)
		}
		client.sg.Done()
	}()

	for {
		var (
			b                 []byte
			err               error
			req               esapi.IndexRequest
			res               *esapi.Response
			e                 map[string]interface{}
			elasticSearchData protocol.ElasticSearchData
			_index            = config.GlobalConfig.ELK.DefaultIndexName
		)

		select {
		case data, ok := <-client.dataChan:
			if !ok {
				k3.K3LogError("WriteDataToElasticSearch Data channel closed !")
				return
			}

		outerLoop:
			// 获取索引名称
			for index, filepaths := range config.GlobalConfig.Watch.ReadPath {
				for _, filepath := range filepaths {
					if strings.HasPrefix(data.EventName, filepath) {
						_index = index
						break outerLoop
					}
				}
			}

			if config.GlobalConfig.ELK.IsUseSuffixDate {
				_index = _index + "_" + data.Timestamp.Format("20060102")
			}

			// 如果解析失败，则直接赋值给text字段
			if err = json.Unmarshal([]byte(data.Properties["_data"].(string)), &elasticSearchData); err != nil || elasticSearchData.Flag == false {
				elasticSearchData.ExtendData.Content = make(map[string]interface{})
				elasticSearchData.ExtendData.Content["text"] = data.Properties["_data"]
			}

			if elasticSearchData.UUID == "" {
				elasticSearchData.UUID = data.UUID
			}
			if elasticSearchData.HostName == "" {
				elasticSearchData.HostName, _ = os.Hostname()
			}
			if elasticSearchData.HostIp == "" {
				elasticSearchData.HostIp = data.Ip
			}
			elasticSearchData.Timestamp = data.Timestamp

			if b, err = json.Marshal(elasticSearchData); err != nil {
				k3.K3LogError("WriteDataToElasticSearch Failed to marshal data: %v", err)
				continue
			}

			// fmt.Printf("document_id(%s), _index(%s), body(%s)\n", elasticSearchData.UUID, _index, string(b))

			req = esapi.IndexRequest{
				Index:      _index,
				DocumentID: fmt.Sprintf("%s", elasticSearchData.UUID),
				Body:       strings.NewReader(string(b)),
				Pretty:     true,
			}

			if res, err = req.Do(context.Background(), client.client); err != nil {
				k3.K3LogError("Failed to send data to Elasticsearch: %v", err)
				continue
			}

			if res.IsError() {
				if err = json.NewDecoder(res.Body).Decode(&e); err != nil {
					k3.K3LogError("Error parsing the response body: %s", err)
					continue
				} else {
					k3.K3LogError("Error response: %s: %s\n", res.Status(), e["error"])
					continue
				}
			}
			res.Body.Close()
			k3.K3LogDebug("Send data (document_id: %v) to Elasticsearch successfully.", data.UUID)
		}
	}
}

func (e *ElasticSearchClient) Close() error {
	close(e.dataChan)
	e.sg.Wait()
	return nil
}

func (e *ElasticSearchClient) Send(data []protocol.Data) error {
	// 循环发送数据
	for _, d := range data {
		if err := e.sendWithRetries(&d); err != nil {
			k3.K3LogError("Failed to send data(UUID: %s) to Elasticsearch: %v", d.UUID, err)
		}
	}
	return nil
}

func (e *ElasticSearchClient) sendWithRetries(d *protocol.Data) error {
	timeout, cancel := context.WithTimeout(context.Background(), time.Duration(e.timeout)*time.Second)
	defer cancel()

	for i := 0; i < e.maxRetries; i++ {
		select {
		case <-timeout.Done():
			k3.K3LogError("Timeout exceeded while sending data to Elasticsearch")
			return timeout.Err()
		case e.dataChan <- d:
			return nil
		default:
			time.Sleep(time.Duration(e.retryInterval) * time.Second)
			k3.K3LogWarn("%d attempt, the data channel is full, data number [%s], retry ......", i, d.UUID)
		}
	}

	k3.K3LogError("Data channel is still full, data will be discarded, data (UUID: %d): %v", d.UUID, d)
	return fmt.Errorf("data channel is full after retries")
}

// 将多条数据封装成1条数据, 暂时先不测试
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
