package gorm

import (
	"context"
	"time"

	"github.com/go-mixins/log"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type Printer func(string, ...interface{})

func (p Printer) Printf(format string, vals ...interface{}) {
	if p == nil {
		return
	}
	p(format, vals...)
}

type ctxLogger logger.LogLevel

func (ctxLogger) LogMode(l logger.LogLevel) logger.Interface {
	return ctxLogger(l)
}

func (l ctxLogger) Info(ctx context.Context, f string, v ...interface{}) {
	if logger.LogLevel(l) < logger.Info {
		return
	}
	log.Get(ctx).WithContext(log.M{
		"line": utils.FileWithLineNum(),
	}).
		Infof(f, v...)
}

func (l ctxLogger) Warn(ctx context.Context, f string, v ...interface{}) {
	if logger.LogLevel(l) < logger.Warn {
		return
	}
	log.Get(ctx).WithContext(log.M{
		"line": utils.FileWithLineNum(),
	}).
		Warnf(f, v...)
}

func (l ctxLogger) Error(ctx context.Context, f string, v ...interface{}) {
	if logger.LogLevel(l) < logger.Error {
		return
	}
	log.Get(ctx).WithContext(log.M{
		"line": utils.FileWithLineNum(),
	}).
		Errorf(f, v...)
}

func (l ctxLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if logger.LogLevel(l) < logger.Info && err == nil {
		return
	}
	dt := time.Since(begin)
	sql, rows := fc()
	logger := log.Get(ctx).WithContext(log.M{
		"line":     utils.FileWithLineNum(),
		"duration": dt.Milliseconds(),
		"rows":     rows,
	})
	if err != nil {
		logger = logger.WithContext(log.M{
			"error": err.Error(),
		})
	}
	logger.Debugf("%s", sql)
}
