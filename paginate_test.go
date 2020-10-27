package gorm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/andviro/goldie"
	gormigrate "github.com/go-gormigrate/gormigrate/v2"
	"github.com/go-mixins/log"
	"github.com/go-mixins/log/logrus"
	"gorm.io/driver/sqlite"
	g "gorm.io/gorm"

	"github.com/go-mixins/gorm/v2"
)

type testItem struct {
	UID        string `gorm:"primary_key"`
	Value      string
	AccessTime time.Time `gorm:"index"`
}

var backend *gorm.Backend

var logger = logrus.New()

func TestMain(m *testing.M) {
	flag.Parse()
	b := &gorm.Backend{
		Driver:      sqlite.Open(":memory:"),
		Migrate:     true,
		UseLogMixin: true,
	}
	b.Debug = testing.Verbose()
	if b.Debug {
		logger.SetLevel(logrus.DebugLevel)
	}
	if err := b.Connect(
		&gormigrate.Migration{
			ID: "initial",
			Migrate: func(tx *g.DB) error {
				return tx.AutoMigrate(&testItem{})
			},
		},
	); err != nil {
		panic(err)
	}
	backend = b.WithContext(log.With(context.Background(), logger))
	data, err := ioutil.ReadFile("testdata/fixtures.json")
	if err != nil {
		panic(err)
	}
	var tis []testItem
	json.Unmarshal(data, &tis)
	for i, ti := range tis {
		ti.AccessTime = time.Unix(int64(i/2*2), 0)
		b.DB.Create(&ti)
	}
	os.Exit(m.Run())
}

func TestPaginateUID(t *testing.T) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	var reqs []*gorm.Pagination
	data, err := ioutil.ReadFile("testdata/uid.json")
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(data, &reqs)
	if err != nil {
		t.Fatal(err)
	}
	enc.SetIndent("", "\t")
	p := &gorm.Paginator{FieldName: "UID", Debug: true}
	for _, tc := range reqs {
		var res []testItem
		backend.DB.Scopes(p.Scope(tc)).Find(&res)
		results, pgn := p.Paginate(res, tc)
		enc.Encode(tc)
		enc.Encode(pgn)
		enc.Encode(results)
	}
	goldie.Assert(t, "pagination-uid", buf.Bytes())
}

func TestPaginateTime(t *testing.T) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	var reqs []*gorm.Pagination
	data, err := ioutil.ReadFile("testdata/time.json")
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(data, &reqs)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	enc.SetIndent("", "\t")
	p := &gorm.Paginator{FieldName: "AccessTime", TieBreakField: "UID", Debug: true, IsTime: true}
	for _, tc := range reqs {
		var res []testItem
		if err := backend.DB.Scopes(p.Scope(tc)).Find(&res).Error; err != nil {
			t.Fatalf("%+v", err)
		}
		results, pgn := p.Paginate(res, tc)
		enc.Encode(tc)
		enc.Encode(pgn)
		enc.Encode(results)
	}
	goldie.Assert(t, "pagination-time", buf.Bytes())
}
