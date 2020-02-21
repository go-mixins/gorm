package gorm

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-mixins/log"
)

type logger struct {
	printer func(...interface{})
}

func newLogger(ctx context.Context, level logLevel) logger {
	res := logger{}
	if ctx != nil {
		logger := log.Get(ctx)
		switch level {
		case LogInfo:
			res.printer = logger.Info
		default:
			res.printer = logger.Debug
		}
	}
	return res
}

func (l logger) Print(vals ...interface{}) {
	if l.printer == nil {
		return
	}
	var res []string
	for _, v := range vals {
		res = append(res, fmt.Sprint(v))
	}
	l.printer(strings.Join(res, " "))
}
