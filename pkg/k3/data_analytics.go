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
