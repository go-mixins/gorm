package gorm

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

// Paginator provides cursor-based paginator
type Paginator struct {
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
}

// GetPageSize returns pagination size
func (p *Pagination) GetPageSize() int {
	if p == nil {
		return 0
	}
	return p.PageSize
}

type scope struct {
	rev, neg, hasOffset bool
	field, tie          string
	cursor              map[string]interface{}
}

func (p *Paginator) cursor(val reflect.Value, idx int) string {
	elt := val.Index(idx)
	if elt.Kind() == reflect.Ptr {
		elt = elt.Elem()
	}
	res := map[string]interface{}{
		"f": elt.FieldByName(p.FieldName).Interface(),
	}
	if p.TieBreakField != "" {
		res["t"] = elt.FieldByName(p.TieBreakField).Interface()
	}
	jd, _ := json.Marshal(res)
	if p.Debug {
		return string(jd)
	}
	return base64.RawURLEncoding.EncodeToString(jd)
}

// Paginate query result according to parameter
func (p *Paginator) Paginate(src interface{}, pgn *Pagination) (interface{}, *Pagination) {
	if src == nil || pgn == nil {
		return src, nil
	}
	value := reflect.ValueOf(src)
	kind := value.Kind()
	if kind == reflect.Ptr {
		value = value.Elem()
		kind = value.Kind()
	}
	if (kind != reflect.Array && kind != reflect.Slice) || value.Len() == 0 {
		return src, nil
	}
	res := &Pagination{
		ThisPageToken: pgn.ThisPageToken,
	}
	reverse := p.isReverse(pgn)
	n := value.Len()
	dest := value
	if n != 0 {
		if pgn.PageSize != 0 && n > pgn.PageSize {
			res.PageSize = n - 1
			if !reverse {
				res.NextPageToken = p.cursor(value, n-2)
			} else {
				res.PrevPageToken = "-" + p.cursor(value, n-2)
			}
			dest = value.Slice(0, n-1)
		}
	}
	n = dest.Len()
	res.PageSize = n
	if reverse {
		valueType := value.Type()
		for i := n/2 - 1; i >= 0; i-- {
			opp := n - 1 - i
			val := reflect.New(valueType.Elem()).Elem()
			val.Set(dest.Index(opp))
			dest.Index(opp).Set(dest.Index(i))
			dest.Index(i).Set(val)
		}
	}
	return dest.Interface(), res
}

func (p *Paginator) isReverse(pgn *Pagination) bool {
	if pgn.ThisPageToken == "" {
		return false
	}
	return strings.HasPrefix(pgn.ThisPageToken, "-")
}

func (p *scope) order(db *gorm.DB) *gorm.DB {
	if p.rev {
		db = db.Order(p.field + " DESC")
		if p.tie != "" {
			db = db.Order(p.tie + " DESC")
		}
		return db
	}
	db = db.Order(p.field + " ASC")
	if p.tie != "" {
		db = db.Order(p.tie + " ASC")
	}
	return db
}

func (p *scope) reverse(db *gorm.DB) *gorm.DB {
	if p.rev {
		db = db.Order(p.field + " ASC")
		if p.tie != "" {
			db = db.Order(p.tie + " ASC")
		}
		return db
	}
	db = db.Order(p.field + " DESC")
	if p.tie != "" {
		db = db.Order(p.tie + " DESC")
	}
	return db
}

// SELECT t.column FROM Table t WHERE t.column > ? OR (t.column = ? AND t.id > ?) ORDER BY t.column, t.id FETCH FIRST 10 ROWS ONLY

func (p *scope) forward(db *gorm.DB) *gorm.DB {
	if p.rev {
		db = db.Where(p.field+" < ?", p.cursor["f"])
		if p.cursor["t"] != nil {
			db = db.Or(p.field+" = ? AND "+p.tie+" < ?", p.cursor["f"], p.cursor["t"])
		}
		return db
	}
	db = db.Where(p.field+" > ?", p.cursor["f"])
	if p.cursor["t"] != nil {
		db = db.Or(p.field+" = ? AND "+p.tie+" > ?", p.cursor["f"], p.cursor["t"])
	}
	return db
}

func (p *scope) backward(db *gorm.DB) *gorm.DB {
	if p.rev {
		db = db.Where(p.field+" > ?", p.cursor["f"])
		if p.cursor["t"] != nil {
			db = db.Or(p.field+" = ? AND "+p.tie+" > ?", p.cursor["f"], p.cursor["t"])
		}
		return db
	}
	db = db.Where(p.field+" < ?", p.cursor["f"])
	if p.cursor["t"] != nil {
		db = db.Or(p.field+" = ? AND "+p.tie+" < ?", p.cursor["f"], p.cursor["t"])
	}
	return db
}

func (p *Paginator) scope(pgn *Pagination) *scope {
	res := &scope{
		rev:   p.Reverse,
		field: gorm.ToDBName(p.FieldName),
		tie:   gorm.ToDBName(p.TieBreakField),
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
func (p *Paginator) Scope(pgn *Pagination) func(*gorm.DB) *gorm.DB {
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
