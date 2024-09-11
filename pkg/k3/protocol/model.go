package protocol

// Data 需要提交给日志存储服务器的数据接口
type Data struct {
	IsComplex    bool                   `json:"-"` // properties are nested or not
	AccountId    string                 `json:"account_id,omitempty"`
	DistinctId   string                 `json:"distinct_id,omitempty"`
	Type         string                 `json:"type"`
	Time         string                 `json:"time"`
	EventName    string                 `json:"event_name,omitempty"`
	EventId      string                 `json:"event_id,omitempty"`
	FirstCheckId string                 `json:"first_check_id,omitempty"`
	Ip           string                 `json:"ip,omitempty"`
	UUID         string                 `json:"uuid,omitempty"`
	AppId        string                 `json:"app_id,omitempty"`
	Properties   map[string]interface{} `json:"properties"`
}

type K3Consumer interface {
	Add(data Data) error
	Flush() error
	Close() error
}
