package server

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

func NewLogger(debug bool) *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetReportCaller(true) // 启用调用源信息

	formatter := &logrus.TextFormatter{
		TimestampFormat:  time.DateTime,      // 时间格式化为 "2006-01-02 15:04:05"
		FullTimestamp:    true,               // 显示完整时间
		CallerPrettyfier: formatCallerSource, // 自定义调用源格式
	}
	logger.SetFormatter(formatter)
	if debug {
		logger.SetLevel(logrus.DebugLevel) // 设置默认日志级别
	} else {
		logger.SetLevel(logrus.InfoLevel) // 设置默认日志级别
	}

	return logger
}

// 处理调用源信息，格式化为 "文件名:行号"
func formatCallerSource(frame *runtime.Frame) (function string, file string) {
	fileName := filepath.Base(frame.File)
	return "", fmt.Sprintf("%s:%d", fileName, frame.Line)
}
