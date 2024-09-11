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

func New(consumer protocol.K3Consumer) DataAnalytics {
	K3LogInfo("New Data Analytics")

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

func (i *DataAnalytics) Track(accountId, appId, eventName, eventId, ip string, properties map[string]interface{}) error {
	return i.track(accountId, appId, eventName, eventId, ip, properties)
}

func (i *DataAnalytics) track(accountId, appId, eventName, eventId, ip string, properties map[string]interface{}) error {
	var (
		msg string
		p   map[string]interface{}
	)

	defer func() {
		if r := recover(); r != nil {
			K3LogError("Recovered in track: %v", r)
		}
	}()

	if len(accountId) == 0 {
		msg = "invalid parameters: account_id cannot be empty "
		K3LogError(msg)
		return errors.New(msg)
	}

	if len(eventId) == 0 {
		msg = "the event id must be provided"
		K3LogError(msg)
		return errors.New(msg)
	}

	p = i.GetSuperProperties()
	MergeProperties(p, properties)
	return i.add(accountId, appId, eventName, eventId, ip, p)
}

func (i *DataAnalytics) add(accountId, appId, eventName, eventId, ip string, properties map[string]interface{}) error {
	var (
		now  string
		uuid string
		data protocol.Data
	)

	now = time.Now().Format("2006-01-02 15:04:05")
	uuid = GenerateUUID()
	data = protocol.Data{
		AccountId:  accountId,
		AppId:      appId,
		EventId:    eventId,
		EventName:  eventName,
		Ip:         ip,
		Properties: properties,
		Time:       now,
		UUID:       uuid,
	}

	return i.consumer.Add(data)
}
