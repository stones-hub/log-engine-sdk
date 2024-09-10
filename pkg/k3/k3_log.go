package k3

import (
	"fmt"
	"log"
	"time"
)

// SDK_LOG_PREFIX is the prefix of log
const SDK_LOG_PREFIX = "[K3SDK] "

// K3Logger is a logger interface
type K3Logger interface {
	Print(message string)
}

var (
	// current log level
	CurrentLogLevel = K3LogLevelOFF
	// custom logger
	LogInstance K3Logger
)

// SetCustomLogger set custom logger
func SetCustomLogger(logger K3Logger) {
	if logger != nil {
		LogInstance = logger
	}
}

type K3LogLevel int

const (
	K3LogLevelOFF K3LogLevel = iota
	K3LogLevelERROR
	K3LogLevelWARN
	K3LogLevelINFO
	K3LogLevelDEBUG
)

// SetLogLevel set log level
func SetLogLevel(level K3LogLevel) {
	if level < K3LogLevelOFF || level > K3LogLevelDEBUG {
		log.Printf("Invalid log level: %d", level)
		return
	}
	CurrentLogLevel = level
}

// K3Log print log
func K3Log(level K3LogLevel, format string, v ...interface{}) {
	if level > CurrentLogLevel {
		return
	}

	var baseMessage string

	switch level {
	case K3LogLevelERROR:
		baseMessage = "[Error] "
		break
	case K3LogLevelWARN:
		baseMessage = "[Warn] "
		break
	case K3LogLevelINFO:
		baseMessage = "[Info] "
		break
	case K3LogLevelDEBUG:
		baseMessage = "[Debug] "
		break
	default:
		baseMessage = "[Info] "
		break
	}

	if LogInstance != nil {
		msg := fmt.Sprintf(SDK_LOG_PREFIX+baseMessage+format+"\n", v...)
		LogInstance.Print(msg)
	} else {
		logTime := fmt.Sprintf("[%v]", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Printf(logTime+SDK_LOG_PREFIX+baseMessage+format+"\n", v...)
	}
}

func K3LogDebug(format string, v ...interface{}) {
	K3Log(K3LogLevelDEBUG, format, v...)
}

func K3LogInfo(format string, v ...interface{}) {
	K3Log(K3LogLevelINFO, format, v...)
}

func K3LogWarn(format string, v ...interface{}) {
	K3Log(K3LogLevelWARN, format, v...)

}

func K3LogError(format string, v ...interface{}) {
	K3Log(K3LogLevelERROR, format, v...)
}
