package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type DB string
type DBURI string

type testModuleA struct {
	Test string `help:"A flag."`
}

func (t *testModuleA) ProvideDB(uri DBURI) DB { return DB(fmt.Sprintf("DB:%s:%s", uri, t.Test)) }

type testModuleB struct{}

func (t *testModuleB) Configure(config Configurator) error { return nil }
func (t *testModuleB) ProvideURI() DBURI                   { return DBURI("postgres://127.0.0.1") }

type testApp struct {
	run        bool
	configured bool
	db         DB
}

func (t *testApp) Configure(config Configurator) error {
	t.configured = true
	return nil
}

func (t *testApp) Start(db DB) error {
	t.run = true
	t.db = db
	return nil
}

func TestAppConfigureProvideInject(t *testing.T) {
	app := New("", "").
		Install(&testModuleA{}, &testModuleB{})

	myApp := &testApp{}
	err := app.RunWithArgs([]string{}, myApp)
	assert.NoError(t, err)
	assert.True(t, myApp.run)
	assert.True(t, myApp.configured)
	assert.Equal(t, DB("DB:postgres://127.0.0.1:"), myApp.db)

	myApp = &testApp{}
	err = app.RunWithArgs([]string{"--test=flag"}, myApp)
	assert.NoError(t, err)
	assert.True(t, myApp.run)
	assert.True(t, myApp.configured)
	assert.Equal(t, DB("DB:postgres://127.0.0.1:flag"), myApp.db)
}
