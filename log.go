package gorm

import (
	"context"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-mixins/log"
	"gorm.io/gorm/logger"
)

func fileWithLineNum() string {
	_, file, _, _ := runtime.Caller(2) // XXX fragile!
	dirname, _ := filepath.Split(file)
	for i := 3; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && (!strings.HasPrefix(file, dirname) || strings.HasSuffix(file, "_test.go")) {
			return file + ":" + strconv.FormatInt(int64(line), 10)
		}
	}
	return ""
}

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
		"caller": fileWithLineNum(),
	}).
		Infof(f, v...)
}

func (l ctxLogger) Warn(ctx context.Context, f string, v ...interface{}) {
	if logger.LogLevel(l) < logger.Warn {
		return
	}
	log.Get(ctx).WithContext(log.M{
		"caller": fileWithLineNum(),
	}).
		Warnf(f, v...)
}

func (l ctxLogger) Error(ctx context.Context, f string, v ...interface{}) {
	if logger.LogLevel(l) < logger.Error {
		return
	}
	log.Get(ctx).WithContext(log.M{
		"caller": fileWithLineNum(),
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
		"caller":   fileWithLineNum(),
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
