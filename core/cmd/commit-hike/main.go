// Command commit-hike is the engine behind the Commit Hike IDE plugins.
package main

import (
	"fmt"
	"os"

	"github.com/Shchusia/commit-hike/core/internal/cli"
	"github.com/Shchusia/commit-hike/core/internal/store"
)

// version is set at build time: -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() {
	dir, err := store.DefaultDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version, dir))
}
