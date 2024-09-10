package k3

import "sync"

type Data struct {
	AccountId  string                 `json:"account_id,omitempty"` // 账号ID
	Type       string                 `json:"type,omitempty"`       // 事件类型
	EventName  string                 `json:"event_name,omitempty"` // 事件名称
	EventId    string                 `json:"event_id,omitempty"`   // 事件ID
	IP         string                 `json:"ip,omitempty"`         // IP
	UUID       string                 `json:"uuid,omitempty"`       // UUID
	AppId      string                 `json:"app_id,omitempty"`     // 应用ID
	Properties map[string]interface{} `json:"properties,omitempty"` // 事件属性
}

type K3Analytics struct {
	consumer   K3Consumer
	properties map[string]interface{}
	mutex      *sync.RWMutex
}

func New(consumer K3Consumer) K3Analytics {
	K3LogInfo("K3Analytics init")
	return K3Analytics{
		consumer:   consumer,
		properties: make(map[string]interface{}),
		mutex:      new(sync.RWMutex),
	}
}

func (ka *K3Analytics) track(accountId, dataType, eventName, eventId string, properties map[string]interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			K3LogError("K3Analytics track panic: %v \n data: %+v", r, properties)
		}
	}()

	return nil
}
