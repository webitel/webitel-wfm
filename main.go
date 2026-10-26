package main

import (
	"fmt"
	"os"

	"github.com/webitel/webitel-wfm/cmd"
)

//go:generate go run github.com/bufbuild/buf/cmd/buf@v1.73.0 generate --template buf.gen.yaml
//go:generate go run github.com/vektra/mockery/v2@v2.53.7

func main() {
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
