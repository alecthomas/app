// Package app is a main entry point for modular applications. Each application consists of a set
// of Modules providing features, and a function that runs the application. The types provided by
// the modules can be used by other modules, or by the application run function as parameters.
//
// It is an opinionated framework, relying on Kingpin for command-line management.
//
//		func run(db *mgo.DB) error {
//			return nil
//		}
//
//		func main() {
//			modules := []app.Module{
//				&mongo.Module{},
//				&http.Module{Bind: ":8080"},
//				&auth.Module{},
//				&healthcheck.Module{},
//			}
//			app.Help(help).
//				Install(modules...).
//				Run(run)
//		}
//
// Modules themselves can declare other modules as dependencies which must be explicitly installed
// for the application to start.
//
// See the documentation for Module for more information.
package app

import (
	"fmt"
	"os"
	"reflect"

	"github.com/alecthomas/inject"
	"gopkg.in/alecthomas/kingpin.v3-unstable"
)

// An application Module has four distinct purposes:
//
// 1. Declare other Modules it depends on.
//
// If a module implements the ModuleDependencies interface, any modules it returns must be
// explicitly installed for the application to start.
//
// 		func (m *MyModule) Requires() []app.Module {
// 			return []app.Module{&http.Module{}}
// 		}
//
// 2. Configure the Application instance (add flags, commands, or arguments) by implementing:
//
// 		Configure(app.Configurator) error
//
// 3. Provide instances of types required by the application, or other modules.
//
// Provider methods are in the form:
//
// 		Provide[Multi][Sequence]<type>(...) (<type>, error)
//
// Where [Multi] means that a new instance of <type> will be provided each time it is required, and
// [Sequence] means the return value will be provided as []<type>. This allows multiple providers to
// contribute to a single vaule. ... can be any type provided by the application or other modules.
//
// 4. Optionally run some code at startup, just prior to the application entry point.
//
// If the module has a method called "Start", it will be called with any parameters
// provided by other modules.
type Module interface {
	Configure(module Configurator) error
}

// Configurator is passed to the Module.Configure() method to allow modules to add flags.
type Configurator interface {
	Flag(name, help string) *kingpin.FlagClause
	Command(name, help string) *kingpin.CmdClause
}

// ModuleDependencies interface can be implemented to define dependencies for a module.
type ModuleDependencies interface {
	Dependencies() []Module
}

// Application object.
type Application struct {
	*kingpin.Application
	// Modules and their dependencies
	modules map[Module][]Module
}

// New creates a new Application instance.
func New(name, help string) *Application {
	a := &Application{
		Application: kingpin.New(name, help),
		modules:     map[Module][]Module{},
	}
	return a
}

// SelectedCommand is available for injection to module Start() functions, as well as the
// main run() function.
type SelectedCommand string

// Help sets the application help.
func (a *Application) Help(help string) *Application {
	a.Application.Help = help
	return a
}

// Install an application module.
//
// If the module conforms to ModuleDependencies, those modules will be required when the
// application is Run().
func (a *Application) Install(modules ...Module) *Application {
	for _, module := range modules {
		t := reflect.TypeOf(module)
		// Already seen this module?
		for m := range a.modules {
			if reflect.TypeOf(m) == t {
				panic(fmt.Sprintf("%s is already installed", t))
			}
		}
		deps := []Module{}
		if md, ok := module.(ModuleDependencies); ok {
			deps = md.Dependencies()
		}
		a.modules[module] = deps
	}
	return a
}

// Run the application using the given run function and command-line args.
//
// Its arguments will be obtained from the installed modules.
func (a *Application) Run(run interface{}) error {
	return a.RunWithArgs(os.Args[1:], run)
}

// RunWithArgs the application using the given run function.
//
// Its arguments will be obtained from the installed modules.
func (a *Application) RunWithArgs(args []string, run interface{}) error {
	injector := inject.New()
	if err := injector.Bind(a); err != nil {
		return err
	}
	if err := injector.Bind(injector); err != nil {
		return err
	}
	modules, err := topoSortModules(a.modules)
	if err != nil {
		return err
	}
	// Configure modules.
	for _, module := range modules {
		if err = injector.Install(module); err != nil {
			return err
		}
		if err = module.Configure(a); err != nil {
			return err
		}
	}
	// Parse arguments.
	command, err := a.Parse(args)
	if err != nil {
		return err
	}
	// Start modules.
	for _, module := range modules {
		mv := reflect.ValueOf(module)
		method := mv.MethodByName("Start")
		if method.IsValid() {
			if _, err = injector.Call(method.Interface()); err != nil {
				return err
			}
		}
	}
	// Run application.
	if err = injector.Bind(SelectedCommand(command)); err != nil {
		return err
	}
	_, err = injector.Call(run)
	return err
}

// Ensure module dependencies are initialised in correct order.
func topoSortModules(deps map[Module][]Module) ([]Module, error) {
	installed := map[reflect.Type]Module{}
	for m := range deps {
		installed[reflect.TypeOf(m)] = m
	}
	graphSorted := []Module{}
	graphUnsorted := map[reflect.Type][]reflect.Type{}
	for i, d := range deps {
		deps := []reflect.Type{}
		for _, dep := range d {
			deps = append(deps, reflect.TypeOf(dep))
		}
		graphUnsorted[reflect.TypeOf(i)] = deps
	}

	for len(graphUnsorted) != 0 {
		acyclic := false
		for node, edges := range graphUnsorted {
			found := false
			for _, edge := range edges {
				if installed[edge] == nil {
					return nil, fmt.Errorf("module %s is not available to %s", edge, node)
				}
				if _, ok := graphUnsorted[edge]; ok {
					found = true
					break
				}
			}
			if !found {
				acyclic = true
				delete(graphUnsorted, node)
				graphSorted = append(graphSorted, installed[node])
			}

		}
		if !acyclic {
			return nil, fmt.Errorf("cyclic module dependency")
		}
	}
	return graphSorted, nil
}
