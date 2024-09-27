package protocol

// ElkLogData 存入到elk的数据日志
type ElkLogData struct {
	Host      string    // 来源主机
	Ip        string    // 来源ip
	LogType   string    // 日志类型
	EventId   EventID   // 事件id
	EventName EventName // 事件名称
}

type LogType string

const (
	Nginx LogType = "nginx"
	Mysql LogType = "mysql"
	Redis LogType = "redis"
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
