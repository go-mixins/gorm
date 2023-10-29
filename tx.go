package gorm

import (
	"errors"
	"fmt"
)

func (b *Backend) Begin() *Backend {
	res := new(Backend)
	*res = *b
	res.DB = b.DB.Begin()
	return res
}

func (b *Backend) End(rErr error) error {
	if e := recover(); e != nil {
		rErr = errors.Join(rErr, fmt.Errorf("panic: %+v", e))
		defer panic(rErr)
	}
	if rErr != nil {
		if err := b.DB.Rollback().Error; err != nil {
			return errors.Join(rErr, fmt.Errorf("rolling back: %+v", err))
		}
		return rErr
	} else if err := b.DB.Commit().Error; err != nil {
		return fmt.Errorf("committing changes: %w", err)
	}
	return nil
}
