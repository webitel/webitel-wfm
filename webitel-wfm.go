package main

import (
	_ "github.com/google/wire"

	"github.com/webitel/webitel-wfm/cmd"
)

//go:generate go run github.com/bufbuild/buf/cmd/buf@v1.42.0 generate --template buf.gen.yaml
//go:generate go run github.com/vektra/mockery/v2@v2.46.0
//go:generate go run github.com/google/wire/cmd/wire@v0.6.0 gen ./cmd

func main() {
	if err := cmd.Run(); err != nil {
		return
	}
}
