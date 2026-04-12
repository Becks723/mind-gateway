package logging

import (
	"log/slog"
	"os"
)

// Logger 封装应用的结构化日志能力
type Logger struct {
	logger *slog.Logger // logger 表示底层 slog 日志实例
}

// NewLogger 创建新的结构化日志记录器
func NewLogger(level string) *Logger {
	// 解析日志等级并构造 JSON 日志处理器
	logLevel := parseLevel(level)
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	return &Logger{
		logger: slog.New(handler),
	}
}

// Info 记录信息级日志
func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Error 记录错误级日志
func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// parseLevel 将字符串日志等级转换为 slog 等级
func parseLevel(level string) slog.Level {
	// 按输入等级返回对应的 slog 等级
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
