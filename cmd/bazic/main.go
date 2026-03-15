package main

import (
	"os"

	"baziclang/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args, cli.Options{
		BinaryName:     "bazic",
		DefaultBackend: "llvm",
		Version:        "v0.2.0",
	}))
}
