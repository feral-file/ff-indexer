package log

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var defaultLogger *zap.Logger

func Initialize(level string, isDebug bool) error {
	log, err := New(level, isDebug)
	if err != nil {
		return err
	}

	defaultLogger = log
	return nil
}

func New(level string, isDebug bool) (*zap.Logger, error) {
	var config zap.Config

	if isDebug {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// override log level by configuration
	l := zap.ErrorLevel
	switch strings.ToUpper(level) {
	case "TRACE", "DEBUG":
		l = zap.DebugLevel
	case "INFO":
		l = zap.InfoLevel
	case "WARN":
		l = zap.WarnLevel
	}

	config.Level = zap.NewAtomicLevelAt(l)

	return config.Build()
}

func mustDefaultLogger() *zap.Logger {
	if defaultLogger == nil {
		panic("use indexer logger without initializing")
	}

	return defaultLogger
}

func Debug(msg string, fields ...zap.Field) {
	mustDefaultLogger().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	mustDefaultLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	mustDefaultLogger().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	mustDefaultLogger().Error(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	mustDefaultLogger().Panic(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	mustDefaultLogger().Fatal(msg, fields...)
}

func Sugar() *zap.SugaredLogger {
	return mustDefaultLogger().Sugar()
}
