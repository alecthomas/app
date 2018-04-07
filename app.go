// Package app is a main entry point for modular applications. Each application consists of a set of Modules providing
// features, and an application module. Types provided by modules can be used by other modules.
//
// It is an opinionated framework, relying on gopkg.in/alecthomas/kingpin.v3-unstable for command- line management and
// github.com/alecthomas/inject for injection. See those modules for details on defining flags and implementing provider
// methods, respectively.
//
// The application lifecycle is as follows:
//
// 1. Construct an application with New().
//
// 2. Install() any modules.
//
// 3. Call Run() with the "main" module.
//
// 4. An injector is created.
//
// 5. Module construction...
//
// 5.1. Each module (including main) will be installed into the injector.
//
// 5.2. If the module implements the Configurable interface, its Configure() method will be called.
//
// 5.3. Finally, the module will be passed to kingpin.Application.Struct() to provide configuration support.
//
// 6. Kingpin is called to parse the command-line and insert values into the modules.
//
// 7. Each module's Start() method (if any) is called via the injector, injecting parameters from modules.
//
// 8. The "main".Start() is called to run the application.
//
// 9. When "main".Start() returns, run each module's Stop() method (if any).
//
//
// Here is a basic example app:
//
//		type Application struct {
//			Debug bool `help:"Enable debug logging."`
//		}
//
//		// Start the application.
//		func (a *Application) Start(db *mgo.Database) error {
//			// ...
//		}
//
//		func main() {
//			app.Help(help).
//				Install(
//					&mongo.Module{},
//					&http.Module{Bind: ":8080"},
//					&auth.Module{},
//					&healthcheck.Module{},
//				).
//				Run(&Application{})
//		}
//
package app

import (
	"fmt"
	"os"
	"reflect"

	"gopkg.in/alecthomas/kingpin.v3-unstable"

	"github.com/alecthomas/inject"
)

// Binder for injector.
type Binder = inject.SafeBinder

// A Configurable module.
type Configurable interface {
	// Configure the module.
	//
	// "binder" may be used to explicitly add bindings to the injector.
	Configure(binder Binder) error
}

// Application object.
type Application struct {
	*kingpin.Application
	modules []interface{}
}

// New creates a new Application instance.
func New(name, help string) *Application {
	a := &Application{
		Application: kingpin.New(name, help),
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
func (a *Application) Install(modules ...interface{}) *Application {
	a.modules = append(a.modules, modules...)
	return a
}

// Run the given application module's Start(...) method.
//
// Its arguments will be obtained from the installed modules.
func (a *Application) Run(module interface{}) error {
	return a.RunWithArgs(os.Args[1:], module)
}

// RunWithArgs the given application module's Start(...) method.
//
// Its arguments will be obtained from the installed modules.
func (a *Application) RunWithArgs(args []string, module interface{}) error {
	start := reflect.ValueOf(module).MethodByName("Start")
	if !start.IsValid() {
		return fmt.Errorf("no Start(...) method on application module")
	}
	injector := inject.SafeNew()
	if err := injector.Bind(a); err != nil {
		return err
	}
	// Configure modules.
	modules := []interface{}{}
	modules = append(modules, a.modules...)
	modules = append(modules, module)
	for _, module := range modules {
		if err := injector.Install(module); err != nil {
			return err
		}
		if configurable, ok := module.(Configurable); ok {
			if err := configurable.Configure(injector); err != nil {
				return err
			}
		}
		if err := a.Struct(module); err != nil {
			return err
		}
	}
	// Parse arguments.
	command, err := a.Parse(args)
	if err != nil {
		return err
	}
	if err = injector.Bind(SelectedCommand(command)); err != nil {
		return err
	}
	// Call module Start(...) methods.
	for _, module := range modules[:len(modules)-1] {
		mv := reflect.ValueOf(module)
		method := mv.MethodByName("Start")
		if method.IsValid() {
			if _, err = injector.Call(method.Interface()); err != nil {
				return err
			}
		}
	}
	// Run application.
	_, err = injector.Call(start.Interface())
	// Call module Stop(...) methods in reverse.
	for i := len(a.modules) - 1; i >= 0; i-- {
		mv := reflect.ValueOf(a.modules[i])
		method := mv.MethodByName("Stop")
		if method.IsValid() {
			// Don't check for errors, as there's not much we can do.
			injector.Call(method.Interface())
		}
	}
	return err
}
