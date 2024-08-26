package main

import (
	_ "github.com/google/wire"

	"github.com/webitel/webitel-wfm/cmd"
)

//go:generate buf generate --template buf.gen.yaml

//go:generate go run github.com/vektra/mockery/v2@latest
//go:generate go run github.com/google/wire/cmd/wire@latest gen ./cmd

func main() {
	if err := cmd.Run(); err != nil {
		return
	}
}
