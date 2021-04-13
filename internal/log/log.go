package log

import (
	"go.uber.org/zap"
)

func New() *zap.SugaredLogger {
	logger, _ := zap.NewDevelopment()
	return logger.Sugar()
}
