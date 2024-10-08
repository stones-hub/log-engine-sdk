package protocol

type LogLevel string

const (
	Error LogLevel = "error"
	Warn  LogLevel = "warn"
	Info  LogLevel = "info"
	Debug LogLevel = "debug"
)

type LogType string

const (
	NetWork    LogType = "network"
	System     LogType = "system"
	Business   LogType = "business"
	Components LogType = "components"
)

type LogName string

const (
	Nginx         LogName = "nginx"
	ALB           LogName = "alb"
	Application   LogName = "application"
	Mysql         LogName = "mysql"
	Redis         LogName = "redis"
	VM            LogName = "vm"
	MTR           LogName = "mtr"
	AlertWebhook  LogName = "alert_webhook"
	Telegram      LogName = "telegram"
	VMAgent       LogName = "vm_agent"
	VMAlert       LogName = "vm_alert"
	VMStorage     LogName = "vm_storage"
	HostKernel    LogName = "host_kernel"
	HostCron      LogName = "host_cron"
	HostAuth      LogName = "host_auth"
	HostMaillog   LogName = "host_maillog"
	HostDmessage  LogName = "host_dmessage"
	HostMessage   LogName = "host_message"
	HostSecurelog LogName = "host_securelog"
	K8SLog        LogName = "k8s_log"
)

type ElasticSearchData struct {
	UUID      string                 // 日志唯一ID， elk document id
	Timestamp string                 // 日志产生时间 "2024-10-01 12:00:00 "
	Protocol  string                 // 协议"HTTP/1.1"
	Domain    string                 // 域名
	LogIp     string                 // 日志来源IP
	HostIp    string                 // 日志落盘IP
	HostName  string                 // 日志落盘主机名
	LogLevel  LogLevel               // 日志级别
	Org       string                 // 事业部
	Project   string                 // 业务线 业务中心， 公司系统, 投放等
	LogType   LogType                // 日志类型
	LogName   LogName                // 日志名称
	Data      map[string]interface{} // 扩展字段
}

type EventId int

const (
	UserLoginID EventId = 1 << iota
	UserRegisterID
	RoleLoginID
	RoleRegisterID
	PaymentID
)

type EventName string

const (
	UserLogin    EventName = "user_login"
	UserRegister EventName = "user_register"
	RoleLogin    EventName = "role_login"
	RoleRegister EventName = "role_register"
	Payment      EventName = "payment"
)
