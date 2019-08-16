package gorm

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-mixins/log"
)

type logger struct {
	log.ContextLogger
}

func newLogger(ctx context.Context) logger {
	if ctx == nil {
		return logger{}
	}
	return logger{log.Get(ctx)}
}

func (l logger) Print(vals ...interface{}) {
	if l.ContextLogger == nil {
		return
	}
	var res []string
	for _, v := range vals {
		res = append(res, fmt.Sprint(v))
	}
	l.ContextLogger.Info(strings.Join(res, " "))
}
