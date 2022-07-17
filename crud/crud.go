package crud

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/go-mixins/gorm/v3"
	"github.com/oleiade/reflections"
	g "gorm.io/gorm"
)

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

var splitRe = regexp.MustCompile(`\s*[;,]\s*`)

func (b Basic[A]) Find(pgn gorm.Pagination, opts ...func(*g.DB) *g.DB) ([]A, *gorm.Pagination, error) {
	var (
		res []A
		elt A
	)
	p := &gorm.Paginator[A]{Debug: b.Debug, IsTime: true}
	fields, err := reflections.FieldsDeep(&elt)
	if err != nil {
		return nil, nil, err
	}
	for _, f := range fields {
		t, err := reflections.GetFieldTag(&elt, f, `paginate`)
		if err != nil {
			return nil, nil, err
		}
		if t == "" {
			continue
		}
		options := splitRe.Split(t, -1)
		switch options[0] {
		case "key":
			p.FieldName = f
			for _, o := range options {
				switch o {
				case "reverse":
					p.Reverse = true
				case "isTime":
					p.IsTime = true
				}
			}
		case "tieBreak":
			p.TieBreakField = f
		}
	}
	if p.FieldName == "" {
		return nil, nil, fmt.Errorf("key field for %T must be tagged", elt)
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
