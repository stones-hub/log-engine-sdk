package k3

import "sync"

type DataAnalytics struct {
	consumer               K3Consumer
	superProperties        map[string]interface{}
	mutex                  *sync.RWMutex
	dynamicSuperProperties func() map[string]interface{}
}
