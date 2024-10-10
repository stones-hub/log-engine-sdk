package protocol

type LogLevel string

const (
	Error LogLevel = "error"
	Warn  LogLevel = "warn"
	Info  LogLevel = "info"
	Debug LogLevel = "debug"
)

type ElasticSearchData struct {
	UUID       string     // 日志唯一ID， elk document id
	Timestamp  string     // 日志产生时间 "2024-10-01 12:00:00 "
	Protocol   string     // 协议"HTTP/1.1"
	Domain     string     // 域名
	LogIp      string     // 日志来源IP
	HostIp     string     // 日志落盘IP
	HostName   string     // 日志落盘主机名
	LogLevel   LogLevel   // 日志级别
	Org        string     // 事业部
	Project    string     // 业务线 业务中心， 公司系统, 投放等
	ExtendData ExtendData // 扩展字段
}

type ExtendData struct {
	Uid      string
	GameName string
	Amount   int64
	Content  map[string]interface{} // 自定义给日志打印方，随意key-value的方式即可
}
