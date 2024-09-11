package k3

import (
	"log-engine-sdk/pkg/k3/protocol"
	"sync"
)

type DataAnalytics struct {
	consumer               protocol.K3Consumer
	superProperties        map[string]interface{}
	mutex                  *sync.RWMutex
	dynamicSuperProperties func() map[string]interface{}
}

func New(consumer protocol.K3Consumer) DataAnalytics {
	K3LogInfo("New Data Analytics")

	return DataAnalytics{
		consumer:        consumer,
		superProperties: make(map[string]interface{}),
		mutex:           new(sync.RWMutex),
	}
}
