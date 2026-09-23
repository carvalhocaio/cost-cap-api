// Package app contains the application logic behind the CLI entrypoint.
package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

// Name is the program name used in usage and output messages.
const Name = "cost-cap-api"

// version is overridden at build time via -ldflags "-X ...app.version=...".
var version = "dev"

// Version returns the application version.
func Version() string {
	return version
}

// Run parses args (excluding the program name) and executes the application,
// writing regular output to stdout.
func Run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet(Name, flag.ContinueOnError)
	fs.SetOutput(stdout)
	showVersion := fs.Bool("version", false, "print the version and exit")
	name := fs.String("name", "world", "name to greet")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if *showVersion {
		_, err := fmt.Fprintf(stdout, "%s %s\n", Name, Version())
		return err
	}

	_, err := fmt.Fprintf(stdout, "Hello, %s!\n", *name)
	return err
}
