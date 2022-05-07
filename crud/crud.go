package crud

import (
	"errors"
	"fmt"

	"github.com/go-mixins/gorm/v3"
	g "gorm.io/gorm"
)

type Versioned struct {
	Version int
}

type Basic[M any] gorm.Backend

func (b Basic[A]) Create(src *A, opts ...func(*g.DB) *g.DB) error {
	q := b.DB
	for _, opt := range opts {
		q = opt(q)
	}
	if err := q.Create(src).Error; gorm.UniqueViolation(err) {
		return ErrFound
	} else if err != nil {
		return fmt.Errorf("creating %T: %+v", src, err)
	}
	return nil
}

func (b Basic[A]) Update(upd A, opts ...func(*g.DB) *g.DB) error {
	q := b.DB.Model(upd)
	for _, opt := range opts {
		q = opt(q)
	}
	if err := q.Updates(upd).Error; gorm.UniqueViolation(err) {
		return ErrFound
	} else if gorm.NotFound(err) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("updating %T: %+v", upd, err)
	} else if q.RowsAffected == 0 {
		return ErrUpdateNotApplied
	}
	return nil
}

func (b Basic[A]) Get(conds ...interface{}) (*A, error) {
	var dest A
	q := b.DB.Model(dest)
	if err := q.First(&dest, conds...).Error; errors.Is(err, g.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("reading %T: %+v", dest, err)
	}
	return &dest, nil
}

func (b Basic[A]) Delete(conds ...interface{}) error {
	var dest A
	if err := b.DB.Delete(&dest, conds...).Error; errors.Is(err, g.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("deleting %T: %+v", dest, err)
	}
	return nil
}
