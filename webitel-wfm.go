package main

import (
	_ "github.com/google/wire"

	"github.com/webitel/webitel-wfm/cmd"
)

//go:generate go tool buf generate --template buf.gen.yaml
//go:generate go tool mockery
//go:generate go tool wire gen ./cmd

func main() {
	if err := cmd.Run(); err != nil {
		return
	}
}
