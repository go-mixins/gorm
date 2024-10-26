package gorm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/oleiade/reflections"

	"gorm.io/gorm"
)

// Paginator provides cursor-based paginator
type Paginator[A any] struct {
	FieldName     string
	TieBreakField string
	Reverse       bool
	Debug         bool
	IsTime        bool
}

// Pagination contains pagination fields
type Pagination struct {
	PageSize      int    `json:"page_size"`
	NextPageToken string `json:"next_page_token"`
	ThisPageToken string `json:"this_page_token"`
	PrevPageToken string `json:"prev_page_token"`
	table         string
}

func (p Pagination) WithTable(tableName string) Pagination {
	res := p
	res.table = tableName
	return res
}

// GetPageSize returns pagination size
func (p *Pagination) GetPageSize() int {
	if p == nil {
		return 0
	}
	return p.PageSize
}

type scope struct {
	rev, neg, hasOffset     bool
	dbField, dbTie, dbTable string
	cursor                  map[string]interface{}
}

var splitRe = regexp.MustCompile(`\s*[;,]\s*`)

func NewPaginator[A any]() (*Paginator[*A], error) {
	var elt A
	p := &Paginator[*A]{}
	fields, err := reflections.FieldsDeep(&elt)
	if err != nil {
		return nil, err
	}
	for _, f := range fields {
		t, err := reflections.GetFieldTag(&elt, f, `paginate`)
		if err != nil {
			return nil, err
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
		return nil, fmt.Errorf("key field for %T must be tagged", elt)
	}
	return p, nil
}

func (p *Paginator[A]) cursor(elt A) string {
	f, _ := reflections.GetField(elt, p.FieldName)
	res := map[string]interface{}{
		"f": f,
	}
	if p.TieBreakField != "" {
		t, _ := reflections.GetField(elt, p.TieBreakField)
		res["t"] = t
	}
	jd, _ := json.Marshal(res)
	if p.Debug {
		return string(jd)
	}
	return base64.RawURLEncoding.EncodeToString(jd)
}

// Paginate query result according to parameter
func (p *Paginator[A]) Paginate(src []A, pgn *Pagination) ([]A, *Pagination) {
	if len(src) == 0 || pgn == nil {
		return src, nil
	}
	res := &Pagination{
		ThisPageToken: pgn.ThisPageToken,
	}
	reverse := pgn.isReverse()
	n := len(src)
	dest := src
	if n != 0 && pgn.PageSize != 0 {
		if n > pgn.PageSize {
			res.PageSize = n - 1
			if !reverse {
				res.NextPageToken = p.cursor(src[n-2])
				if res.ThisPageToken != "" {
					res.PrevPageToken = "-" + p.cursor(src[0])
				}
			} else {
				res.PrevPageToken = "-" + p.cursor(src[n-2])
				res.NextPageToken = p.cursor(src[0])
			}
			dest = src[0 : n-1]
		}
		if !reverse {
			if res.ThisPageToken != "" {
				res.PrevPageToken = "-" + p.cursor(src[0])
			}
		} else {
			res.NextPageToken = p.cursor(src[0])
		}
	}
	n = len(dest)
	res.PageSize = n
	if reverse {
		for i := n/2 - 1; i >= 0; i-- {
			opp := n - 1 - i
			dest[opp], dest[i] = dest[i], dest[opp]
		}
	}
	return dest, res
}

func (pgn *Pagination) isReverse() bool {
	if pgn.ThisPageToken == "" {
		return false
	}
	return strings.HasPrefix(pgn.ThisPageToken, "-")
}

func (p *scope) order(db *gorm.DB) *gorm.DB {
	if p.rev {
		db = db.Order(p.field(db) + " DESC")
		if p.dbTie != "" {
			db = db.Order(p.tie(db) + " DESC")
		}
		return db
	}
	db = db.Order(p.field(db) + " ASC")
	if p.dbTie != "" {
		db = db.Order(p.tie(db) + " ASC")
	}
	return db
}

func (p *scope) reverse(db *gorm.DB) *gorm.DB {
	if p.rev {
		db = db.Order(p.field(db) + " ASC")
		if p.dbTie != "" {
			db = db.Order(p.tie(db) + " ASC")
		}
		return db
	}
	db = db.Order(p.field(db) + " DESC")
	if p.dbTie != "" {
		db = db.Order(p.tie(db) + " DESC")
	}
	return db
}

// SELECT t.column FROM Table t WHERE t.column > ? OR (t.column = ? AND t.id > ?) ORDER BY t.column, t.id FETCH FIRST 10 ROWS ONLY

func (p *scope) forward(db *gorm.DB) *gorm.DB {
	if p.rev {
		db = db.Where(p.field(db)+" < ?", p.cursor["f"])
		if p.cursor["t"] != nil {
			db = db.Or(p.field(db)+" = ? AND "+p.tie(db)+" < ?", p.cursor["f"], p.cursor["t"])
		}
		return db
	}
	db = db.Where(p.field(db)+" > ?", p.cursor["f"])
	if p.cursor["t"] != nil {
		db = db.Or(p.field(db)+" = ? AND "+p.tie(db)+" > ?", p.cursor["f"], p.cursor["t"])
	}
	return db
}

func (p *scope) backward(db *gorm.DB) *gorm.DB {
	if p.rev {
		db = db.Where(p.field(db)+" > ?", p.cursor["f"])
		if p.cursor["t"] != nil {
			db = db.Or(p.field(db)+" = ? AND "+p.tie(db)+" > ?", p.cursor["f"], p.cursor["t"])
		}
		return db
	}
	db = db.Where(p.field(db)+" < ?", p.cursor["f"])
	if p.cursor["t"] != nil {
		db = db.Or(p.field(db)+" = ? AND "+p.tie(db)+" < ?", p.cursor["f"], p.cursor["t"])
	}
	return db
}

func (s *scope) field(db *gorm.DB) string {
	res := db.NamingStrategy.ColumnName("", s.dbField)
	if s.dbTable != "" {
		res = fmt.Sprintf(`%s."%s"`, s.dbTable, res)
	}
	return res
}

func (s *scope) tie(db *gorm.DB) string {
	res := db.NamingStrategy.ColumnName("", s.dbTie)
	if s.dbTable != "" {
		res = fmt.Sprintf(`%s."%s"`, s.dbTable, res)
	}
	return res
}

func (p *Paginator[A]) scope(pgn *Pagination) *scope {
	res := &scope{
		rev:     p.Reverse,
		dbField: p.FieldName,
		dbTie:   p.TieBreakField,
		dbTable: pgn.table,
	}
	if pgn == nil {
		return res
	}
	res.neg = strings.HasPrefix(pgn.ThisPageToken, "-")
	var data []byte
	if p.Debug {
		data = []byte(strings.TrimPrefix(pgn.ThisPageToken, "-"))
	} else {
		data, _ = base64.RawURLEncoding.DecodeString(strings.TrimPrefix(pgn.ThisPageToken, "-"))
	}
	json.Unmarshal(data, &res.cursor)
	if p.IsTime && res.cursor != nil {
		d, _ := res.cursor["f"].(string)
		res.cursor["f"], _ = time.Parse(time.RFC3339Nano, d)
	}
	res.neg = res.neg && res.cursor != nil
	return res
}

// Scope provides proper offset and ordering for cursor db
func (p *Paginator[A]) Scope(pgn *Pagination) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		s := p.scope(pgn)
		if n := pgn.GetPageSize(); n != 0 {
			db = db.Limit(int(n) + 1)
		}
		switch {
		case s.neg:
			db = s.reverse(s.backward(db))
		case s.cursor != nil:
			db = s.forward(db)
			fallthrough
		default:
			db = s.order(db)
		}
		return db
	}
}
