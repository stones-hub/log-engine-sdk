package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type Data struct {
	ID   int
	Data string
}

type ElasticSearchClient struct {
	dataChan chan *Data
	mu       sync.Mutex
}

func NewElasticSearchClient(capacity int) *ElasticSearchClient {
	return &ElasticSearchClient{
		dataChan: make(chan *Data, capacity),
	}
}

func (e *ElasticSearchClient) Send(data []Data) error {
	// 循环发送数据
	for _, d := range data {
		if err := e.sendWithRetry(&d); err != nil {
			k3LogInfo(fmt.Sprintf("Failed to send data (ID: %d): %v", d.ID, err))
		}
	}
	return nil
}

func (e *ElasticSearchClient) sendWithRetry(data *Data) error {
	maxRetries := 6
	retryDelay := 1 * time.Second
	timeout := 5 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for i := 0; i <= maxRetries; i++ {
		select {
		case <-ctx.Done():
			fmt.Println("Timeout exceeded")
			return ctx.Err()
		case e.dataChan <- data:
			// 数据成功发送
			fmt.Println("Data sent successfully", data.ID)
			return nil
		default:
			// 数据通道已满，记录日志并重试
			k3LogInfo(fmt.Sprintf("Data channel is full, retrying... (ID: %d)", data.ID))
			time.Sleep(retryDelay)
		}
	}

	// 最终重试仍然失败，记录日志并放弃
	k3LogInfo(fmt.Sprintf("Data channel is still full, data will be discarded (ID: %d)", data.ID))
	return fmt.Errorf("data channel is full after retries")
}

func k3LogInfo(msg string) {
	log.Println(msg)
}

func main() {
	// 初始化 Elasticsearch 客户端
	client := NewElasticSearchClient(10)

	// 测试数据
	testData := []Data{
		{ID: 1, Data: "Data 1"},
		{ID: 2, Data: "Data 2"},
		{ID: 3, Data: "Data 3"},
		{ID: 4, Data: "Data 4"},
		{ID: 5, Data: "Data 5"},
		{ID: 6, Data: "Data 6"},
		{ID: 7, Data: "Data 7"},
		{ID: 8, Data: "Data 8"},
		{ID: 9, Data: "Data 9"},
		{ID: 10, Data: "Data 10"},
		{ID: 11, Data: "Data 11"},
		{ID: 12, Data: "Data 12"},
	}

	// 发送数据
	err := client.Send(testData)
	if err != nil {
		log.Fatalf("Error sending data: %v", err)
	}

	fmt.Println("All data sent successfully")
}
