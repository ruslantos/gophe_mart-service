package logger

import (
	"sync"

	"go.uber.org/zap"
)

var (
	once     sync.Once
	instance *zap.Logger
)

func Get() *zap.Logger {
	once.Do(func() {
		var err error
		instance, err = zap.NewProduction()
		if err != nil {
			panic(err)
		}
	})
	return instance
}

func Sync() {
	if instance != nil {
		_ = instance.Sync()
	}
}
