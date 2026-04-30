package main

import (
	"os"
	"path/filepath"

	"github.com/shadowbook/nightward/internal/cli"
)

func main() {
	os.Exit(cli.RunWithName(filepath.Base(os.Args[0]), os.Args[1:], os.Stdout, os.Stderr))
}
