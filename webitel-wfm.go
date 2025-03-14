package main

import (
	_ "github.com/google/wire"

	"github.com/webitel/webitel-wfm/cmd"
)

//go:generate buf generate --template buf.gen.yaml
//go:generate mockery
//go:generate wire gen ./cmd

func main() {
	if err := cmd.Run(); err != nil {
		return
	}
}
