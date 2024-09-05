//go:build wireinject
// +build wireinject

package cmd

import (
	"context"

	"github.com/google/wire"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/health"
	"github.com/webitel/webitel-wfm/infra/shutdown"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/handler"
	"github.com/webitel/webitel-wfm/internal/service"
	"github.com/webitel/webitel-wfm/internal/storage"
)

var wireResourceSet = wire.NewSet(
	sqlStorage, wire.Bind(new(cluster.Store), new(*cluster.Cluster)),
	inmemoryCache, serviceDiscovery, auth, webitelEngine, pubsubConn, webitelLogger, audit,
)

var wireHandlersSet = wire.NewSet(
	storage.NewPauseTemplate,
	service.NewPauseTemplate, wire.Bind(new(service.PauseTemplateManager), new(*storage.PauseTemplate)),
	handler.NewPauseTemplate, wire.Bind(new(handler.PauseTemplateManager), new(*service.PauseTemplate)),

	storage.NewShiftTemplate,
	service.NewShiftTemplate, wire.Bind(new(service.ShiftTemplateManager), new(*storage.ShiftTemplate)),
	handler.NewShiftTemplate, wire.Bind(new(handler.ShiftTemplateManager), new(*service.ShiftTemplate)),

	storage.NewWorkingCondition,
	service.NewWorkingCondition, wire.Bind(new(service.WorkingConditionManager), new(*storage.WorkingCondition)),
	handler.NewWorkingCondition, wire.Bind(new(handler.WorkingConditionManager), new(*service.WorkingCondition)),

	storage.NewAgentWorkingConditions,
	service.NewAgentWorkingConditions, wire.Bind(new(service.AgentWorkingConditionsManager), new(*storage.AgentWorkingConditions)),
	handler.NewAgentWorkingConditions, wire.Bind(new(handler.AgentWorkingConditionsManager), new(*service.AgentWorkingConditions)),

	storage.NewAgentAbsence,
	service.NewAgentAbsence, wire.Bind(new(service.AgentAbsenceManager), new(*storage.AgentAbsence)),
	handler.NewAgentAbsence, wire.Bind(new(handler.AgentAbsenceManager), new(*service.AgentAbsence)),

	storage.NewForecastCalculation,
	service.NewForecastCalculation, wire.Bind(new(service.ForecastCalculationManager), new(*storage.ForecastCalculation)),
	handler.NewForecastCalculation, wire.Bind(new(handler.ForecastCalculationManager), new(*service.ForecastCalculation)),
)

func initResources(context.Context, *config.Config, *wlog.Logger, *health.CheckRegistry, *shutdown.Tracker) (*resources, error) {
	wire.Build(wireResourceSet, wire.Struct(new(resources), "*"))

	return &resources{}, nil
}

func initHandlers(*wlog.Logger, *resources, cluster.ForecastStore) (*handlers, error) {
	wire.Build(wireHandlersSet,
		wire.FieldsOf(new(*resources), "cache", "storage", "engine", "audit"),
		wire.Struct(new(handlers), "*"),
	)

	return &handlers{}, nil
}
