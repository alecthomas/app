# Modular application framework for Go [![](https://godoc.org/github.com/alecthomas/app?status.svg)](http://godoc.org/github.com/alecthomas/app) [![Build Status](https://travis-ci.org/alecthomas/app.png)](https://travis-ci.org/alecthomas/app)

Each application consists of a set of Modules providing features, and a function that runs the
application. Modules can provide instances of types. Those types can be used by other modules or
the application run function as parameters.

This is generally not a useful package for typical Go applications. It is
intended for large code bases where multiple applications are composed from
separate modules.

It is an opinionated framework, relying on Kingpin for command-line management.

Under typical usage, packages will export modules. In this situation a typical main file might look
like:

```go
package main

import (
  "gopkg.in/mgo.v2"
  "github.com/alecthomas/app"

  "myapp/user"
  "myapp/mongo"
)

func run(manager *user.UserManager) error {
  user, err := manager.GetUser("alec")
  return err
}

func main() {
  app.
    Install(&mongo.Module{}, &user.Module{}).
    Run(run)
}
```

Here's what each module might look like.

`myapp/mongo/module.go`

```go
package mongo

import (
  "gopkg.in/mgo.v2"
  "github.com/alecthomas/app"
)

// Configures and provides a Mongo session.
type Module struct {
  URI string
}

func (m *Module) Configure(config app.Configurator) error {
  config.Flag("mongo-uri", "Mongo URI").Required().StringVar(&m.URI)
  return nil
}

func (m *Module) ProvideMongoSession() (*mgo.Session, error) {
  return mgo.Dial(m.URI)
}

// The session is automatically provided by ProvideMongoSession().
func (m *Module) ProvideMongoDB(session *mgo.Session) *mgo.DB {
  return session.DB("db")
}
```

`myapp/user/module.go`


```go
package user

import "myapp/mongo"

// Provide a Mongo-backed user management type.
type Module struct {}

func (u *Module) Configure(config app.Configurator) error {
  return nil
}

func (u *Module) Dependencies() []app.Module {
  return []app.Module{&MongoModule{}}
}

func (u *Module) ProvideUserManager(session *mgo.Session) (*user.UserManager, error) {
  return New(session)
}
```

Modules themselves can declare other modules as dependencies, which must be explicitly installed
for the application to start.
