package main

import (
	"os"

	"baziclang/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args, cli.Options{
		BinaryName:     "bazc",
		DefaultBackend: "go",
		Version:        "v0.2.0",
	}))
}
