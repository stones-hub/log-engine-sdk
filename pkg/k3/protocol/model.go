package protocol

import "time"

// Data 需要提交给日志存储服务器的数据接口
type Data struct {
	AccountId  string                 `json:"account_id,omitempty"` // 账户ID
	AppId      string                 `json:"app_id,omitempty"`     // APPID
	Timestamp  time.Time              `json:"Timestamp"`            // 日志时间
	Ip         string                 `json:"ip,omitempty"`         // 日志来源ID
	UUID       string                 `json:"uuid,omitempty"`       // 日志唯一ID
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
