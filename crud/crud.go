package crud

import (
	"fmt"

	"github.com/go-mixins/gorm/v3"
	g "gorm.io/gorm"
)

type Versioned struct {
	Version int
}

type Basic[M interface {
	Versioned
	GetVersion() int
	SetVersion(int)
}] gorm.Backend

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
	ver := upd.GetVersion()
	upd.SetVersion(ver + 1)
	q = q.Where(`version = ?`, ver)
	if err := q.Updates(upd).Error; gorm.UniqueViolation(err) {
		return ErrFound
	} else if gorm.NotFound(err) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("updating %T: %+v", upd, err)
	}
	if q.RowsAffected == 0 {
		return ErrConcurrency
	}
	return nil
}
