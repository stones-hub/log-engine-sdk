package protocol

import "time"

type ElasticSearchData struct {
	UUID       string     `json:"_id,omitempty"`        // 日志唯一ID， elk document id  √
	LogLevel   string     `json:"log_level,omitempty"`  // 日志级别 warn, error, debug, info
	HostName   string     `json:"host_name,omitempty"`  // 日志落盘主机名 √
	HostIp     string     `json:"host_ip,omitempty"`    // 日志落盘IP √
	TraceId    string     `json:"trace_id,omitempty"`   // 追踪ID
	Domain     string     `json:"domain,omitempty"`     // 域名
	Protocol   string     `json:"protocol,omitempty"`   // 协议"HTTP/1.1"
	HttpCode   int        `json:"http_code,omitempty"`  // http状态码
	ClientIp   string     `json:"client_ip,omitempty"`  // 日志来源IP
	Org        string     `json:"org,omitempty"`        // 事业部, gnfx, tapplus, zganme等
	Project    string     `json:"project,omitempty"`    // 业务线 ywzx， company, tf, sdk4,等
	CodeName   string     `json:"code_name,omitempty"`  // 代码仓库标识, user_service, nginx, mysql, redis等
	LogSrc     string     `json:"log_src,omitempty"`    // 日志源(内部调用default/sms/phone等)
	EventId    int        `json:"event_id,omitempty"`   // 日志事件ID
	EventName  string     `json:"event_name,omitempty"` // 日志事件名称(每种日志唯一)
	Timestamp  time.Time  `json:"Timestamp,omitempty"`  // 日志产生时间 "2024-10-01 12:00:00 " √
	ExtendData ExtendData `json:"extend_data"`          // 扩展字段
}

type ExtendData struct {
	Uid      string                 `json:"uid,omitempty"`       // 用户ID
	GameName string                 `json:"game_name,omitempty"` // 游戏名
	Amount   int64                  `json:"amount,omitempty"`    // 充值金额
	Currency string                 `json:"currency,omitempty"`  // 货币类型
	Language string                 `json:"language,omitempty"`  // 日志来源系统 （PHP, GO, JAVA, NODEJS, Android, IOS, Linux
	Version  string                 `json:"version,omitempty"`   // 代码版本号
	Code     int                    `json:"code,omitempty"`      // 业务的状态码
	Content  map[string]interface{} `json:"content,omitempty"`   // 自定义给日志打印方，随意key-value的方式即可
}
