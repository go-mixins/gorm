package crud_test

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"testing"
	"time"

	gormigrate "github.com/go-gormigrate/gormigrate/v2"
	"github.com/go-mixins/log"
	"github.com/go-mixins/log/logrus"
	"gorm.io/driver/sqlite"
	g "gorm.io/gorm"

	"github.com/go-mixins/gorm/v3"
	"github.com/go-mixins/gorm/v3/crud"
)

type testItem struct {
	UID        string `gorm:"primary_key" paginate:"tieBreak"`
	Value      string
	AccessTime time.Time `gorm:"index" paginate:"key;isTime;reverse"`
}

var (
	backend *gorm.Backend
	api     *crud.Basic[testItem]
)

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
	api = (*crud.Basic[testItem])(backend)
	os.Exit(m.Run())
}

func TestCrud_Create_Update(t *testing.T) {
	data, err := ioutil.ReadFile("../testdata/fixtures.json")
	if err != nil {
		panic(err)
	}
	var tis []testItem
	json.Unmarshal(data, &tis)
	for i, ti := range tis {
		if err := api.Create(&ti); err != nil {
			t.Errorf("create %d: %+v", i, err)
		}
	}
	sel := func(q *g.DB) *g.DB {
		return q.Select("Value")
	}
	for i, ti := range tis {
		if err := api.Update(ti, sel); err != nil {
			t.Errorf("update %d: %+v", i, err)
		}
	}
	for i, ti := range tis {
		itm, err := api.Get(`value = ?`, ti.Value)
		if err != nil {
			t.Errorf("get %d: %+v", i, err)
		}
		t.Logf("%+v", itm)
	}
	res, _, err := api.Find(gorm.Pagination{})
	if err != nil {
		t.Errorf("paginate: %+v", err)
	} else if len(res) != len(tis) {
		t.Errorf("must have length %d", len(tis))
	}
	for i, ti := range res {
		if err := api.Delete(`uid = ?`, ti.UID); err != nil {
			t.Errorf("delete %d: %+v", i, err)
		}
	}
}
