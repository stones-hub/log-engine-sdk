package k3

import (
	"log-engine-sdk/pkg/k3/protocol"
	"sync"
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

func (i *DataAnalytics) track() {

	defer func() {
		if r := recover(); r != nil {
			K3LogError("Recovered in track: %v", r)
		}
	}()

}

func (i *DataAnalytics) add() {

}
