// Command cost-cap-api is the CLI entrypoint.
package main

import (
	"fmt"
	"os"

	"github.com/carvalhocaio/cost-cap-api/internal/app"
)

func main() {
	if err := app.Run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", app.Name, err)
		os.Exit(1)
	}
}
