package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	mockhealth "github.com/webitel/webitel-wfm/gen/go/mocks/health"
)

func TestCheckRegistry_Register(t *testing.T) {
	log := wlog.NewLogger(&wlog.LoggerConfiguration{})
	registry := NewCheckRegistry(log)
	check := mockhealth.NewMockCheck(t)
	registry.Register(check)

	assert.Contains(t, registry.GetChecks(), check)
}

func TestCheckRegistry_RegisterFunc(t *testing.T) {
	log := wlog.NewLogger(&wlog.LoggerConfiguration{})
	registry := NewCheckRegistry(log)
	checkFunc := func(ctx context.Context) error { return nil }

	registry.RegisterFunc("test-check", checkFunc)
	assert.Len(t, registry.GetChecks(), 1)
}

func TestCheckRegistry_GetChecks(t *testing.T) {
	log := wlog.NewLogger(&wlog.LoggerConfiguration{})
	registry := NewCheckRegistry(log)
	check := mockhealth.NewMockCheck(t)
	registry.Register(check)
	checks := registry.GetChecks()

	assert.Len(t, checks, 1)
	assert.Equal(t, check, checks[0])
}

func TestCheckRegistry_RunAll(t *testing.T) {
	log := wlog.NewLogger(&wlog.LoggerConfiguration{})
	registry := NewCheckRegistry(log)
	check := mockhealth.NewMockCheck(t)
	checkResult := []CheckResult{{Name: "test-check", Err: nil}}

	check.On("HealthCheck", mock.Anything).Return(checkResult)
	registry.Register(check)

	ctx := context.Background()
	results := registry.RunAll(ctx)
	assert.Len(t, results, 1)
	assert.Equal(t, "test-check", results[0].Name)
	assert.NoError(t, results[0].Err)
}

func TestCheckRegistry_RunAll_WithTimeout(t *testing.T) {
	log := wlog.NewLogger(&wlog.LoggerConfiguration{})
	registry := NewCheckRegistry(log)
	check := mockhealth.NewMockCheck(t)
	checkResult := []CheckResult{{Name: "test-check", Err: nil}}

	check.On("HealthCheck", mock.Anything).Return(checkResult)
	registry.Register(check)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond) // Ensure the context times out
	results := registry.RunAll(ctx)
	assert.Len(t, results, 1)
	assert.Equal(t, "health-checks.run", results[0].Name)
	assert.Equal(t, context.DeadlineExceeded, results[0].Err)
}

func TestCheckRegistry_RunAll_WithPanic(t *testing.T) {
	log := wlog.NewLogger(&wlog.LoggerConfiguration{})
	registry := NewCheckRegistry(log)
	check := mockhealth.NewMockCheck(t)

	check.On("HealthCheck", mock.Anything).Run(func(args mock.Arguments) {
		panic("test panic")
	}).Return(nil)
	registry.Register(check)

	ctx := context.Background()
	results := registry.RunAll(ctx)
	assert.Len(t, results, 0)
}
