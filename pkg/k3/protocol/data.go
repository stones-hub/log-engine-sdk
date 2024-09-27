package protocol

/*
用于 elk 日志格式
*/

// ElkLogData 存入到elk的数据日志
type ElkLogData struct {
	UUID      string    // uuid
	Timestamp string    // 日志生成时间 "2024-10-01 12:01:00"
	HostName  string    // 来源主机
	IP        string    // 来源ip
	LogLevel  LogLevel  // 日志级别
	LogType   LogType   // 日志类型
	EventId   EventID   // 事件id
	EventName EventName // 事件名称
	Data      interface{}
}

type LogLevel string

const (
	Error LogLevel = "error"
	Warn  LogLevel = "warn"
	Info  LogLevel = "info"
	Debug LogLevel = "debug"
)

type LogType string

const (
	Nginx LogType = "nginx"
	Mysql LogType = "mysql"
	Redis LogType = "redis"
	Trace LogType = "trace"
)

type EventName string

const (
	UserLogin    EventName = "user_login"
	UserRegister EventName = "user_register"
	RoleLogin    EventName = "role_login"
	RoleRegister EventName = "role_register"
)

type EventID int

const (
	UserLoginID EventID = 1 << iota
	UserRegisterID
	RoleLoginID
	RoleRegisterID
)
