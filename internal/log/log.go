package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(environment string, debug bool) *zap.SugaredLogger {
	var logger *zap.Logger

	if environment == "prod" {
		errorUnlessEnabled := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			// true: log message at this level
			// false: skip message at this level
			return level >= zapcore.ErrorLevel || debug
		})

		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			os.Stdout,
			errorUnlessEnabled,
		)

		return zap.New(core).Sugar()
	}

	logger, _ = zap.NewDevelopment()
	return logger.Sugar()
}
