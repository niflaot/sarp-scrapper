package logger

import (
	"os"
	"time"

	uberzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(config Config) (*uberzap.Logger, error) {
	return newWithSink(config, zapcore.Lock(os.Stdout), zapcore.Lock(os.Stderr))
}

func newWithSink(config Config, output, errorOutput zapcore.WriteSyncer) (*uberzap.Logger, error) {
	level, err := config.ParseLevel()
	if err != nil {
		return nil, err
	}
	encoderConfig := zapcore.EncoderConfig{
		TimeKey: "time", LevelKey: "level", NameKey: "logger", CallerKey: "caller",
		MessageKey: "message", StacktraceKey: "stacktrace", LineEnding: zapcore.DefaultLineEnding,
		EncodeTime:     zapcore.TimeEncoderOfLayout(time.RFC3339Nano),
		EncodeDuration: zapcore.StringDurationEncoder, EncodeCaller: zapcore.ShortCallerEncoder,
	}
	var encoder zapcore.Encoder
	if config.JSON {
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}
	return uberzap.New(zapcore.NewCore(encoder, output, level), uberzap.ErrorOutput(errorOutput), uberzap.AddCaller()), nil
}
