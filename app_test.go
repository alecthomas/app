package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type DB string
type DBURI string

type testModuleA struct {
	Flag string
}

func (t *testModuleA) Configure(config Configurator) error {
	config.Flag("test", "").StringVar(&t.Flag)
	return nil
}
func (t *testModuleA) ProvideDB(uri DBURI) DB { return DB(fmt.Sprintf("DB:%s:%s", uri, t.Flag)) }

type testModuleB struct{}

func (t *testModuleB) Configure(config Configurator) error { return nil }
func (t *testModuleB) ProvideURI() DBURI                   { return DBURI("postgres://127.0.0.1") }

func TestAppConfigureProvideInject(t *testing.T) {
	run := false
	var db DB
	app := New("", "").
		Install(&testModuleA{}, &testModuleB{})

	err := app.RunWithArgs([]string{}, func(db_ DB) error {
		run = true
		db = db_
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, run)
	assert.Equal(t, DB("DB:postgres://127.0.0.1:"), db)

	err = app.RunWithArgs([]string{"--test=flag"}, func(db_ DB) error {
		run = true
		db = db_
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, run)
	assert.Equal(t, DB("DB:postgres://127.0.0.1:flag"), db)
}
