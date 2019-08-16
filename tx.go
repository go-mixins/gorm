package gorm

import (
	"github.com/hashicorp/go-multierror"
	"golang.org/x/xerrors"
)

func (b *Backend) Begin() *Backend {
	res := &Backend{
		DB:      b.DB.Begin(),
		context: b.context,
	}
	res.DB.SetLogger(newLogger(res.context))
	return res
}

func (b *Backend) End(rErr error) error {
	if e := recover(); e != nil {
		rErr = multierror.Append(rErr, xerrors.Errorf("panic: %+v", e))
		defer panic(rErr)
	}
	if rErr != nil {
		if err := b.DB.Rollback().Error; err != nil {
			return multierror.Append(rErr, xerrors.Errorf("rolling back: %w", err))
		}
		return rErr
	}
	if err := b.DB.Commit().Error; err != nil {
		return xerrors.Errorf("committing changes: %w", err)
	}
	return nil
}
