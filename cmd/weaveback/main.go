package main

import (
	"os"

	"github.com/hironow/rest/cmd/weaveback/cli"
)

func main() {
	code := cli.Run(os.Args, os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}
