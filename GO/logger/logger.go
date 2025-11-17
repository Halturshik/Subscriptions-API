package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

var (
	infoLogger  = log.New(os.Stdout, "", 0)
	warnLogger  = log.New(os.Stdout, "", 0)
	errorLogger = log.New(os.Stderr, "", 0)
)

func Info(msg string, args ...any) {
	infoLogger.Printf("[%s] [INFO]  %s", time.Now().Format("2006-01-02 15:04:05"), format(msg, args...))
}

func Warn(msg string, args ...any) {
	warnLogger.Printf("[%s] [WARN]  %s", time.Now().Format("2006-01-02 15:04:05"), format(msg, args...))
}

func Error(msg string, args ...any) {
	errorLogger.Printf("[%s] [ERROR] %s", time.Now().Format("2006-01-02 15:04:05"), format(msg, args...))
}

func format(msg string, args ...any) string {
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}
