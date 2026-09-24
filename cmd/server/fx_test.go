package server

import (
	"testing"

	"go.uber.org/fx"

	"github.com/webitel/webitel-wfm/config"
)

func TestValidateApp(t *testing.T) {
	cfg := new(config.Config)
	if err := fx.ValidateApp(MainModule(cfg)); err != nil {
		t.Fatalf("DI container validation failed: %v", err)
	}
}
