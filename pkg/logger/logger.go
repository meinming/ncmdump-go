package logger

import (
	"fmt"
	"os"
	"time"
)

// 定义 ANSI 终端颜色代码
const (
	colorReset = "\033[0m"
	colorInfo  = "\033[34m" // 蓝色 Blue
	colorWarn  = "\033[33m" // 黄色 Yellow
	colorError = "\033[31m" // 红色 Red
	colorDebug = "\033[36m" // 青色 Cyan
)

var DebugEnabled = false

// Logger 结构体，目前作为可扩展的占位符
type Logger struct{}

func getTimestamp() string {
	// Go 的时间格式化非常奇特，必须使用 "2006-01-02 15:04:05" 这个固定时间作为模版
	return time.Now().Format("2006-01-02 15:04:05")
}

// Info 打印一般提示信息
func Info(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	// 输出格式：[时间] [级别] 内容
	fmt.Printf("%s %s[INFO]%s  %s\n", getTimestamp(), colorInfo, colorReset, msg)
}

// Debug 打印调试信息
func Debug(format string, v ...interface{}) {
	if !DebugEnabled {
		return
	}
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s %s[DEBUG]%s %s\n", getTimestamp(), colorDebug, colorReset, msg)
}

// Warn 打印警告信息
func Warn(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s %s[WARN]%s  %s\n", getTimestamp(), colorWarn, colorReset, msg)
}

// Error 打印错误信息
// 错误日志应当通过 os.Stderr（标准错误流）输出，而不是 os.Stdout（标准输出流）
func Error(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	fmt.Fprintf(os.Stderr, "%s %s[ERROR]%s %s\n", getTimestamp(), colorError, colorReset, msg)
}
