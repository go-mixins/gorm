package gorm

import (
	"context"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/jellevandenhooff/slogctx"
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

type slogLogger logger.LogLevel

func (slogLogger) LogMode(l logger.LogLevel) logger.Interface {
	return slogLogger(l)
}

func (l slogLogger) Info(ctx context.Context, f string, v ...interface{}) {
	if logger.LogLevel(l) < logger.Info {
		return
	}
	ctx = slogctx.WithAttrs(ctx, "caller", fileWithLineNum())
	slogctx.Info(ctx, f, v...)
}

func (l slogLogger) Warn(ctx context.Context, f string, v ...interface{}) {
	if logger.LogLevel(l) < logger.Warn {
		return
	}
	ctx = slogctx.WithAttrs(ctx, "caller", fileWithLineNum())
	slogctx.Warn(ctx, f, v...)
}

func (l slogLogger) Error(ctx context.Context, f string, v ...interface{}) {
	if logger.LogLevel(l) < logger.Error {
		return
	}
	ctx = slogctx.WithAttrs(ctx, "caller", fileWithLineNum())
	slogctx.Error(ctx, f, nil, v...)
}

func (l slogLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if logger.LogLevel(l) < logger.Info && err == nil {
		return
	}
	dt := time.Since(begin)
	sql, rows := fc()
	ctx = slogctx.WithAttrs(ctx, "caller", fileWithLineNum(), "duration", dt.Milliseconds(), "rows", rows)
	if err != nil {
		slogctx.Error(ctx, "%s", err, sql)
		return
	}
	slogctx.Debug(ctx, "%s", sql)
}
