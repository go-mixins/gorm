package crud

import (
	"errors"
	"fmt"

	"github.com/go-mixins/gorm/v4"
	g "gorm.io/gorm"
)

type Basic[A any] gorm.Backend

type CRUD[A any] interface {
	Create(src *A, opts ...func(*g.DB) *g.DB) error
	Update(upd A, opts ...func(*g.DB) *g.DB) error
	Get(conds ...interface{}) (*A, error)
	Delete(conds ...interface{}) error
	Find(pgn gorm.Pagination, opts ...func(*g.DB) *g.DB) ([]*A, *gorm.Pagination, error)
}

type Tx[A any] interface {
	CRUD[A]
	End(rErr error) error
}

func (b *Basic[A]) Transact(f func(tx CRUD[A]) error) (rErr error) {
	tx := b.Begin()
	defer func() {
		rErr = tx.End(rErr)
	}()
	return f(tx)
}

func (b *Basic[A]) Begin() Tx[A] {
	backend := (*gorm.Backend)(b).Begin()
	return (*Basic[A])(backend)
}

func (b *Basic[A]) End(rErr error) error {
	backend := (*gorm.Backend)(b)
	return backend.End(rErr)
}

func (b *Basic[A]) Create(src *A, opts ...func(*g.DB) *g.DB) error {
	q := b.DB
	for _, opt := range opts {
		q = opt(q)
	}
	if err := q.Create(src).Error; errors.Is(err, g.ErrDuplicatedKey) {
		return ErrFound
	} else if err != nil {
		return fmt.Errorf("creating %T: %+v", src, err)
	}
	return nil
}

func (b *Basic[A]) Update(upd A, opts ...func(*g.DB) *g.DB) error {
	q := b.DB.Model(upd)
	for _, opt := range opts {
		q = opt(q)
	}
	if err := q.Updates(upd).Error; errors.Is(err, g.ErrDuplicatedKey) {
		return ErrFound
	} else if errors.Is(err, g.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("updating %T: %+v", upd, err)
	} else if q.RowsAffected == 0 {
		return ErrUpdateNotApplied
	}
	return nil
}

func (b *Basic[A]) Get(conds ...interface{}) (*A, error) {
	var dest A
	q := b.DB.Model(dest)
	if err := q.First(&dest, conds...).Error; errors.Is(err, g.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("reading %T: %+v", dest, err)
	}
	return &dest, nil
}

func (b *Basic[A]) Delete(conds ...interface{}) error {
	var dest A
	if err := b.DB.Delete(&dest, conds...).Error; errors.Is(err, g.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("deleting %T: %+v", dest, err)
	}
	return nil
}

func (b *Basic[A]) Find(pgn gorm.Pagination, opts ...func(*g.DB) *g.DB) ([]*A, *gorm.Pagination, error) {
	var res []*A
	p, err := gorm.NewPaginator[A]()
	if err != nil {
		return nil, nil, err
	}
	q := b.DB.Scopes(p.Scope(&pgn))
	for _, o := range opts {
		q = o(q)
	}
	if err := q.Find(&res).Error; err != nil {
		return nil, nil, err
	}
	results, resPgn := p.Paginate(res, &pgn)
	return results, resPgn, nil
}
