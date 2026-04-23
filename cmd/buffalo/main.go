package main

import (
	"fmt"
	"os"

	"github.com/massonsky/buffalo/internal/cli"
	"github.com/massonsky/buffalo/pkg/errors"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(errors.ExitCode(err))
	}
}
