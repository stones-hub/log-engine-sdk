package protocol

import "time"

// Data 需要提交给日志存储服务器的数据接口
type Data struct {
	UUID       string                 `json:"uuid,omitempty"`       // 日志唯一ID
	AccountId  string                 `json:"account_id,omitempty"` // 账户ID
	AppId      string                 `json:"app_id,omitempty"`     // APPID
	Ip         string                 `json:"ip,omitempty"`         // 日志来源ID
	Timestamp  time.Time              `json:"Timestamp"`            // 日志时间
	EventName  string                 `json:"event_name,omitempty"` // 所读文件路径string
	Properties map[string]interface{} `json:"properties"`           // 日志具体内容
}

type K3Consumer interface {
	Add(data Data) error
	Flush() error
	Close() error
}

type Sender interface {
	Send(data []Data) error
	Close() error
}
