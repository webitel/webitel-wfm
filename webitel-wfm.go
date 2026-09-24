package main

import (
	"fmt"
	"os"

	"github.com/webitel/webitel-wfm/cmd"
)

//go:generate go tool buf generate --template buf.gen.yaml
//go:generate go tool mockery
//go:generate go tool wire gen ./cmd

func main() {
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
