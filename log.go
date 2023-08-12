package gorm

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

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

func (b *Backend) LogMode(l logger.LogLevel) logger.Interface {
	return &Backend{
		Logger:   b.Logger,
		loglevel: l,
	}
}

func (b *Backend) logger() *slog.Logger {
	if b.Logger != nil {
		return b.Logger
	}
	return slog.Default()
}

func (b *Backend) level() logger.LogLevel {
	if b.loglevel != 0 {
		return b.loglevel
	}
	return logger.Warn
}

func (b *Backend) Info(ctx context.Context, f string, v ...interface{}) {
	if b.level() < logger.Info {
		return
	}
	b.logger().InfoContext(ctx, fmt.Sprintf(f, v...), "caller", fileWithLineNum())
}

func (b *Backend) Warn(ctx context.Context, f string, v ...interface{}) {
	if b.level() < logger.Warn {
		return
	}
	b.logger().WarnContext(ctx, fmt.Sprintf(f, v...), "caller", fileWithLineNum())
}

func (b *Backend) Error(ctx context.Context, f string, v ...interface{}) {
	if b.level() < logger.Error {
		return
	}
	b.logger().ErrorContext(ctx, fmt.Sprintf(f, v...), "caller", fileWithLineNum())
}

func (b *Backend) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if b.level() < logger.Info && err == nil {
		return
	}
	dt := time.Since(begin)
	sql, rows := fc()
	attrs := []any{
		"caller",
		fileWithLineNum(),
		"duration", dt.Milliseconds(),
		"rows", rows,
	}
	if err != nil {
		attrs = append(attrs, "error", err)
		b.logger().ErrorContext(ctx, sql, attrs...)
		return
	}
	b.logger().DebugContext(ctx, sql, attrs...)
}
