package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var globalLogger *zap.Logger

// Init 初始化日志系统
func Init(level string, development bool) error {
	var config zap.Config

	if development {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
	}

	// 设置日志级别
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	// 设置输出格式
	config.Encoding = "console"
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	logger, err := config.Build(
		zap.AddCallerSkip(1), // 跳过一层调用栈，显示真实调用位置
	)
	if err != nil {
		return err
	}

	globalLogger = logger
	return nil
}

// GetLogger 获取全局 logger
func GetLogger() *zap.Logger {
	if globalLogger == nil {
		// 如果未初始化，使用默认配置
		globalLogger, _ = zap.NewProduction()
	}
	return globalLogger
}

// Sync 刷新日志缓冲区
func Sync() {
	if globalLogger != nil {
		_ = globalLogger.Sync()
	}
}

// 便捷方法
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

func Debugf(template string, args ...interface{}) {
	GetLogger().Sugar().Debugf(template, args...)
}

func Infof(template string, args ...interface{}) {
	GetLogger().Sugar().Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	GetLogger().Sugar().Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	GetLogger().Sugar().Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	GetLogger().Sugar().Fatalf(template, args...)
	os.Exit(1)
}
