# Modular application framework for Go [![](https://godoc.org/github.com/alecthomas/app?status.svg)](http://godoc.org/github.com/alecthomas/app) [![Build Status](https://travis-ci.org/alecthomas/app.png)](https://travis-ci.org/alecthomas/app) [![Gitter chat](https://badges.gitter.im/alecthomas.png)](https://gitter.im/alecthomas/Lobby)

In large monolithic code-bases, multiple applications will typically share
packages, such as those providing core functionality such as monitoring,
logging, database connectors, RPC server and client configuration, etc. This
framework allows each package to export modules that contain configuration
data (in the form of flags, or configurable at construction time) and logic to
create objects from that package. Modules may themselves require values,
allowing for seamless inter- dependencies, but also allowing applications to
provide modules with configuration (eg. a map of monitoring variables).

Each application consists of a set of Modules providing features, and an
application module that uses those features. Types provided by modules can be
used by other modules. Circular dependencies will be detected.

*This is generally not a useful package for typical Go applications. It is
intended for large code bases where multiple applications are composed from
separate modules.*

It is an opinionated framework, relying on
[gopkg.in/alecthomas/kingpin.v3-unstable](https://gopkg.in/alecthomas/kingpin.v3-unstable)
for command- line management and
[github.com/alecthomas/inject](https://github.com/alecthomas/inject)
for injection. See those modules for details on defining flags and implementing
provider methods, respectively.

Modules may (optionally) configure the Application instance by implementing `app.Configurable`:

```go
Configure(app.Configurator) error
```

Flags can be configured here, but it is generally more convenient to use Kingpin's struct
flags (see Kingpin documentation for details).

If the module has a method called `Start(...)`, it will be called with any parameters injected.
Similarly, any methods with a `Stop(...)` method will have it called in reverse order.

Under typical usage, packages will export modules which are composed together
by main packages. For example:

```go
package main

import (
  "gopkg.in/mgo.v2"
  "github.com/alecthomas/app"
  "github.com/prometheus/client_golang/prometheus"

  "myapp/user"
  "myapp/mongo"
  "myapp/monitoring"
)

type Application struct {
  usersCreated prometheus.Counter
  usersDeleted prometheus.Counter
}

func New() *Application {
  return &Application{
    usersCreated: prometheus.NewCounter(prometheus.CounterOpts{Name: "usersCreated"}),
    usersDeleted: prometheus.NewCounter(prometheus.CounterOpts{Name: "usersDeleted"}),
  }
}

func (a *Application) Start(manager *user.UserManager) error {
  user, err := manager.GetUser("alec")
  // Do something...
  a.usersCreated.Inc()
  return err
}

// Provide monitoring variables for the monitoring package to export.
func (a *Application) ProvideMonitoringMapping() map[string]prometheus.Collector {
  return map[string]prometheus.Collector{
    "usersCreated": a.usersCreated,
    "usersDeleted": a.usersDeleted,
  }
}

func main() {
  app.
    Install(&mongo.Module{}, &user.Module{}, &monitoring.Module{}).
    Run(New())
}
```

Here's what each module might look like.

```go
package mongo

import (
  "gopkg.in/mgo.v2"
)

// Configures and provides a Mongo session.
type Module struct {
  URI string `help:"Mongo URI." required:"true"`
  DB string `help:"Mongo DB to connect to." default:"development"`
}

func (m *Module) ProvideMongoSession() (*mgo.Session, error) {
  return mgo.Dial(m.URI)
}

func (m *Module) ProvideMongoDB(session *mgo.Session) *mgo.DB {
  return session.DB(m.DB)
}
```

This module provides a UserManager instance. It also explicitly installs the
MongoModule to ensure it is available. The application may also install
MongoModule in order to configure it, if required.


```go
package user

import "myapp/mongo"

// Provide a Mongo-backed user management type.
type Module struct {}

func (u *Module) Configure(config app.Configurator) error {
  // Ensures that MongoModule is installed and required by this module.
  config.Install(&MongoModule{})
  return nil
}

func (u *Module) ProvideUserManager(session *mgo.Session) (*user.UserManager, error) {
  return New(session)
}
```

A module for starting a HTTP server. Routes can be provided by other modules
(see monitoring example below).

```go
package httpserver

import (
  "net/http"
)

type Route struct {
  Path string
  Handler http.Handler
}

type Module struct {
  HTTPBind string `help:"Bind address for HTTP server." default:":8090"`
}

func (m *Module) ProvideMux() *http.ServeMux {
  return http.DefaultServeMux
}

func (m *Module) Start(routes []Route) error {
  for _, route := range routes {
    http.Handle(route.Path, route.Handler)
  }
  go http.ListenAndServe(m.HTTPBind, nil)
  return nil
}
```

```go
package monitoring

import (
  "net/http"

  "github.com/prometheus/client_golang/prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"

  "myapp/httpserver"
)

type Module struct {}

func (m *Module) ProvideRouteSequence(collectors []prometheus.Collector) []httpserver.Route {
  for _, collector := range collectors {
    prometheus.MustRegister(collector)
  }
  return []httpserver.Route{{"/metrics", promhttp.Handler()}}
}
```
