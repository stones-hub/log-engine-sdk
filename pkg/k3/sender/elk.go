package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"log"
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
	MaxBulkSize           = 10    // 提交给ElasticSearch的批量大小
	BulkData              []*Bulk
)

type Bulk struct {
	Index      string
	DocumentId string
	body       string
}

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
	defer client.sg.Done()

	BulkData = make([]*Bulk, 0)

	for {
		var (
			requestBody string
			index       string
		)

		select {
		// 获取consumer 提交过来的日志
		case data, ok := <-client.dataChan:
			if !ok {
				k3.K3LogError("WriteDataToElasticSearch Data channel closed !")
				return
			}

			if requestBody = consumerDataToElkData(data); len(requestBody) == 0 {
				continue
			}

			k3.K3LogDebug("WriteDataToElasticSearch: %s", requestBody)

			if len(data.IndexName) == 0 {
				index = config.GlobalConfig.ELK.DefaultIndexName
			} else {
				index = data.IndexName
			}

			if config.GlobalConfig.ELK.IsUseSuffixDate {
				index = index + "_" + time.Now().Format("20060102")
			}

			// 将数据写入BulkData
			BulkData = append(BulkData, &Bulk{
				Index:      index,
				DocumentId: fmt.Sprintf("%s", data.UUID),
				body:       requestBody,
			})

			sendBulkElasticSearch(client.client, false)
		}
	}
}

func (e *ElasticSearchClient) Close() error {
	close(e.dataChan)
	e.sg.Wait()
	sendBulkElasticSearch(e.client, true)
	return nil
}

func sendBulkElasticSearch(client *elasticsearch.Client, force bool) {

	var buffer strings.Builder
	currentBulkSize := len(BulkData)
	if currentBulkSize == 0 {
		return
	}

	// 检查是否满足批量提交的条件
	if currentBulkSize >= MaxBulkSize || force == true {

		for _, item := range BulkData {
			action := map[string]interface{}{
				"index": map[string]interface{}{
					"_index": item.Index,
					"_id":    item.DocumentId,
				},
			}
			buffer.WriteString(mustMarshal(action))
			buffer.WriteString("\n")
			buffer.WriteString(item.body)
			buffer.WriteString("\n")
		}

		k3.K3LogDebug("bulk_data:%s\n", buffer.String())

		BulkData = make([]*Bulk, 0)

		// 创建批量请求
		req := esapi.BulkRequest{
			Body: strings.NewReader(buffer.String()),
		}

		// 批量提交
		res, err := req.Do(context.Background(), client)

		if err != nil {
			k3.K3LogError("Failed to send data to Elasticsearch: %v", err)
			return
		}

		if res.IsError() {
			k3.K3LogError("Unexpected error in Elasticsearch response: %s", res.String())
			res.Body.Close()
			return
		}

		res.Body.Close()
		k3.K3LogDebug("Send data(line:%v) to Elasticsearch successfully.", currentBulkSize)
	} else {
		k3.K3LogDebug("Bulk size(%v) is less than MaxBulkSize(%v)", currentBulkSize, MaxBulkSize)
	}
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

// consumerDataToElkData 将consumer的数据转换为elk的数据
func consumerDataToElkData(data *protocol.Data) string {

	var (
		ok       bool
		_data    interface{}
		err      error
		b        []byte
		elkData  protocol.ElasticSearchData
		hostName string
	)

	// consumer的数据没有_data, 证明无需处理当前日志
	if _data, ok = data.Properties["_data"]; !ok {
		k3.K3LogError("No _data field in data: %v", data)
		return ""
	}

	// host_ip 和 host_name 、uuid 需要生成，SubmitLog 中并没有这些数据
	if hostName, err = os.Hostname(); err != nil {
		k3.K3LogError("Failed to get hostname: %v", err)
		hostName = "unknown"
	}

	// 将日志解析成elkData ， 解析失败，就将原来的数据封装到elkData下的text字段内发送
	if err = json.Unmarshal([]byte(_data.(string)), &elkData); err != nil {
		elkData.HostIp = data.Ip
		elkData.HostName = hostName
		elkData.UUID = data.UUID
		elkData.AccountId = data.AccountId
		elkData.AppId = data.AppId
		elkData.Timestamp = data.Timestamp
		elkData.ExtendData = protocol.ExtendData{
			Content: map[string]interface{}{
				"text": _data.(string),
			},
		}
		if b, err = json.Marshal(&elkData); err != nil {
			fmt.Println("1======>", _data.(string))
			return _data.(string)
		} else {
			return string(b)
		}
	} else {
		if elkData.EventName == "" { // 旧日志，但是是一个json文件
			fmt.Println("2======>", _data.(string))
			return _data.(string)
		} else {
			// 新日志
			elkData.HostIp = data.Ip
			elkData.HostName = hostName
			elkData.UUID = data.UUID
			elkData.AccountId = data.AccountId
			elkData.AppId = data.AppId
			elkData.Timestamp = data.Timestamp
			if b, err = json.Marshal(elkData); err != nil {
				k3.K3LogError("Failed to marshal elkData: %v", err)
				return _data.(string)
			} else {
				fmt.Println("3========>", string(b))
				return string(b)
			}
		}
	}
}

// mustMarshal 将结构体或映射转换为JSON字符串
func mustMarshal(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("Error marshaling JSON: %s", err)
	}
	return string(b)
}
