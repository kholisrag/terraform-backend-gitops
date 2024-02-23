package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func init() {
	var err error
	log, _ = zap.NewProduction()
	defer log.Sync()

	config := zap.NewProductionConfig()
	encoderConfig := zap.NewProductionEncoderConfig()
	// enccoderConfig.StacktraceKey = "" // to hide stacktrace
	config.EncoderConfig = encoderConfig

	log, err = config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}
}

func Init(logLevel string) {
	// Customize the log level dynamically
	currentLogLevel := log.Level()
	var newLogLevel zapcore.Level
	switch strings.ToLower(logLevel) {
	case "info", "INFO":
		newLogLevel = zapcore.InfoLevel
	case "warn", "WARN", "warning", "WARNING":
		newLogLevel = zapcore.WarnLevel
	case "error", "ERROR", "err", "ERR":
		newLogLevel = zapcore.ErrorLevel
	case "panic", "PANIC":
		newLogLevel = zapcore.PanicLevel
	case "fatal", "FATAL":
		newLogLevel = zapcore.FatalLevel
	case "debug", "DEBUG":
		newLogLevel = zapcore.DebugLevel
	default:
		newLogLevel = zapcore.InfoLevel
	}

	if newLogLevel.CapitalString() != currentLogLevel.CapitalString() {
		newEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		newCore := zapcore.NewCore(
			newEncoder,
			zapcore.AddSync(os.Stdout),
			zap.NewAtomicLevelAt(newLogLevel),
		)

		log = log.WithOptions(zap.WrapCore(func(zapcore.Core) zapcore.Core {
			log.Sugar().Infof("log level changed to %v", strings.ToLower(newLogLevel.String()))
			return newCore
		}))
	}
}

func GetZapLogger() *zap.Logger {
	return log
}

func Infof(msg string, args ...interface{}) {
	log.Sugar().Infof(msg, args)
}

func Errorf(msg string, args ...interface{}) {
	log.Sugar().Errorf(msg, args)
}

func Fatalf(msg string, args ...interface{}) {
	log.Sugar().Errorf(msg, args)
}

func Debugf(msg string, args ...interface{}) {
	log.Sugar().Debugf(msg, args)
}

func Warnf(msg string, args ...interface{}) {
	log.Sugar().Warnf(msg, args)
}

func Panicf(msg string, args ...interface{}) {
	log.Sugar().Panicf(msg, args)
}

func Info(msg string, fields ...zap.Field) {
	log.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	log.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	log.Fatal(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	log.Debug(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	log.Warn(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	log.Panic(msg, fields...)
}
