package app

import (
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v3-unstable"
)

var (
	// App is the default application instance.
	App = New(filepath.Base(os.Args[0]), "")
)

// Run the given module using the global Application instance.
func Run(module interface{}) {
	FatalIfError(App.Run(module), "")
}

// Help sets the global Application help.
func Help(help string) *Application {
	return App.Help(help)
}

// Install a module into the global Application instance.
func Install(modules ...interface{}) *Application {
	return App.Install(modules...)
}

// Errorf prints a consistent error message to stderr.
func Errorf(format string, args ...interface{}) {
	kingpin.Errorf(format, args...)
}

// Fatalf prints an error message to stderr and terminates the application
// with a non-zero status.
func Fatalf(format string, args ...interface{}) {
	App.Fatalf(format, args...)
}

// FatalUsage prints command-line usage information the terminates with a non-
// zero status.
func FatalUsage(format string, args ...interface{}) {
	App.FatalUsage(format, args...)
}

// FatalIfError checks if err is present and if so, terminates with the given message.
func FatalIfError(err error, format string, args ...interface{}) {
	App.FatalIfError(err, format, args...)
}
