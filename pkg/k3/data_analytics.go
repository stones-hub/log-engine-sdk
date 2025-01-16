package k3

import (
	"errors"
	"log-engine-sdk/pkg/k3/protocol"
	"sync"
	"time"
)

type DataAnalytics struct {
	consumer        protocol.K3Consumer
	superProperties map[string]interface{}
	mutex           *sync.RWMutex
}

func NewDataAnalytics(consumer protocol.K3Consumer) DataAnalytics {
	K3LogDebug("New Data Analytics")

	return DataAnalytics{
		consumer:        consumer,
		superProperties: make(map[string]interface{}),
		mutex:           new(sync.RWMutex),
	}
}

func (i *DataAnalytics) GetSuperProperties() map[string]interface{} {
	res := make(map[string]interface{})
	i.mutex.Lock()
	defer i.mutex.Unlock()
	MergeProperties(res, i.superProperties)
	return res
}

func (i *DataAnalytics) SetSuperProperties(superProperties map[string]interface{}) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	MergeProperties(i.superProperties, superProperties)
}

func (i *DataAnalytics) CleanSuperProperties() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.superProperties = make(map[string]interface{})
}

func (i *DataAnalytics) Track(accountId, appId, ip, indexName string, properties map[string]interface{}) error {
	return i.track(accountId, appId, indexName, ip, properties)
}

func (i *DataAnalytics) track(accountId, appId, indexName, ip string, properties map[string]interface{}) error {
	var (
		msg string
		p   map[string]interface{}
	)

	defer func() {
		// 防止异常导致整个程序结束, 并不是协程的问题
		if r := recover(); r != nil {
			K3LogError("Recovered in track: %v", r)
		}
	}()

	if len(accountId) == 0 {
		msg = "invalid parameters: account_id cannot be empty "
		K3LogError(msg)
		return errors.New(msg)
	}

	if len(appId) == 0 {
		msg = "the app id must be provided "
		K3LogError(msg)
		return errors.New(msg)
	}

	if len(indexName) == 0 {
		msg = "the event name must be provided "
		K3LogError(msg)
		return errors.New(msg)
	}

	p = i.GetSuperProperties()
	MergeProperties(p, properties)
	return i.add(accountId, appId, indexName, ip, p)
}

func (i *DataAnalytics) add(accountId, appId, indexName, ip string, properties map[string]interface{}) error {
	var (
		uuid string
		data protocol.Data
	)

	uuid = GenerateUUID()
	data = protocol.Data{
		AccountId:  accountId,
		AppId:      appId,
		IndexName:  indexName,
		Ip:         ip,
		Timestamp:  time.Now(),
		UUID:       uuid,
		Properties: properties,
	}
	return i.consumer.Add(data)
}

func (i *DataAnalytics) Close() {
	i.consumer.Close()
}
