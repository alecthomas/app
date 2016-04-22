package app

import (
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	// App is the default application instance.
	App = New(filepath.Base(os.Args[0]), "")
)

// Run a function using the global Application instance.
func Run(run interface{}) {
	FatalIfError(App.Run(os.Args[1:], run), "")
}

// Help sets the global Application help.
func Help(help string) *Application {
	return App.Help(help)
}

// Install a module into the global Application instance.
func Install(modules ...Module) *Application {
	return App.Install(modules...)
}

// Flag adds a new flag to the application.
func Flag(name, help string) *kingpin.FlagClause {
	return App.Flag(name, help)
}

// Arg adds a new positional argument to the application.
func Arg(name, help string) *kingpin.ArgClause {
	return App.Arg(name, help)
}

// Command adds a new top-level argument to the application.
func Command(name, help string) *kingpin.CmdClause {
	return App.Command(name, help)
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
