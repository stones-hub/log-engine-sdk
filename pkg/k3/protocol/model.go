package protocol

import (
	"fmt"
	"time"
)

// Data 需要提交给日志存储服务器的数据接口
type Data struct {
	UUID       string                 `json:"uuid,omitempty"`       // 日志唯一ID
	AccountId  string                 `json:"account_id,omitempty"` // 账户ID
	AppId      string                 `json:"app_id,omitempty"`     // APPID
	Ip         string                 `json:"ip,omitempty"`         // 日志来源ID
	Timestamp  time.Time              `json:"Timestamp"`            // 日志时间
	IndexName  string                 `json:"index_name,omitempty"` // 所读文件的索引标识
	Properties map[string]interface{} `json:"properties"`           // 日志具体内容
}

func (d *Data) String() string {
	return fmt.Sprintf("UUID:%s, AccountId:%s, AppId:%s, Ip:%s, Timestamp:%v, IndexName:%s, Properties:%v", d.UUID, d.AccountId, d.AppId, d.Ip, d.Timestamp, d.IndexName, d.Properties)
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
