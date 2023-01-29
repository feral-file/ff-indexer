package log

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var DefaultLogger *zap.Logger = InitializeLogger()

func InitializeLogger() *zap.Logger {
	log, err := New()
	if err != nil {
		panic(fmt.Errorf("fail to init zap logger: %s", err.Error()))
	}

	return log
}

func New() (*zap.Logger, error) {
	var config zap.Config

	if viper.GetBool("debug") {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// override log level by configuration
	logLevel := zap.DebugLevel
	switch strings.ToUpper(viper.GetString("log.level")) {
	case "TRACE", "DEBUG":
		logLevel = zap.DebugLevel
	case "INFO":
		logLevel = zap.DebugLevel
	case "WARN":
		logLevel = zap.DebugLevel
	}

	config.Level = zap.NewAtomicLevelAt(logLevel)

	return config.Build()
}

func Debug(msg string, fields ...zap.Field) {
	DefaultLogger.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	DefaultLogger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	DefaultLogger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	DefaultLogger.Error(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	DefaultLogger.Panic(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	DefaultLogger.Fatal(msg, fields...)
}

func Sugar() *zap.SugaredLogger {
	return DefaultLogger.Sugar()
}
