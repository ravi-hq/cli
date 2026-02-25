// Package main is the entry point for the Ravi CLI application.
//
// The application is built with the API base URL injected at build time:
//
//	make build API_URL=https://ravi.app
//
// Run with --help to see available commands.
package main

import (
	"fmt"
	"os"

	"github.com/ravi-hq/cli/internal/output"
	"github.com/ravi-hq/cli/pkg/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		output.Current.PrintError(err)
		os.Exit(1)
	}
	fmt.Println()
}
