// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package cmd

import (
	"context"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/health"
	"github.com/webitel/webitel-wfm/infra/pubsub"
	"github.com/webitel/webitel-wfm/infra/server"
	"github.com/webitel/webitel-wfm/infra/shutdown"
	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/infra/webitel/logger"
	"github.com/webitel/webitel-wfm/internal/handler"
	"github.com/webitel/webitel-wfm/internal/service"
	"github.com/webitel/webitel-wfm/internal/storage"
)

// Injectors from wire.go:

func initResources(contextContext context.Context, configConfig *config.Config, wlogLogger *wlog.Logger, checkRegistry *health.CheckRegistry, tracker *shutdown.Tracker) (*resources, error) {
	registry, err := serviceDiscovery(contextContext, configConfig, wlogLogger, checkRegistry, tracker)
	if err != nil {
		return nil, err
	}
	authManager, err := auth(registry, checkRegistry, tracker)
	if err != nil {
		return nil, err
	}
	serverServer, err := server.New(wlogLogger, authManager)
	if err != nil {
		return nil, err
	}
	cluster, err := sqlStorage(contextContext, configConfig, wlogLogger)
	if err != nil {
		return nil, err
	}
	configCache := &configConfig.Cache
	cacheCache, err := cache.New(configCache)
	if err != nil {
		return nil, err
	}
	client, err := engine.New(wlogLogger, registry)
	if err != nil {
		return nil, err
	}
	loggerClient, err := logger.New(wlogLogger, registry)
	if err != nil {
		return nil, err
	}
	configService := loggerClient.ConfigService
	configPubsub := &configConfig.Pubsub
	manager, err := pubsub.New(wlogLogger, configPubsub)
	if err != nil {
		return nil, err
	}
	audit := logger.NewAudit(configService, manager)
	cmdResources := &resources{
		grpcServer: serverServer,
		storage:    cluster,
		cache:      cacheCache,
		authcli:    authManager,
		engine:     client,
		loggercli:  loggerClient,
		audit:      audit,
		registry:   registry,
		ps:         manager,
	}
	return cmdResources, nil
}

func initHandlers(cmdResources *resources, forecastStore cluster.ForecastStore) (*handler.Handlers, error) {
	serverServer := cmdResources.grpcServer
	store := cmdResources.storage
	manager := cmdResources.cache
	pauseTemplate := storage.NewPauseTemplate(store, manager)
	servicePauseTemplate := service.NewPauseTemplate(pauseTemplate)
	handlerPauseTemplate := handler.NewPauseTemplate(serverServer, servicePauseTemplate)
	shiftTemplate := storage.NewShiftTemplate(store)
	serviceShiftTemplate := service.NewShiftTemplate(shiftTemplate)
	handlerShiftTemplate := handler.NewShiftTemplate(serverServer, serviceShiftTemplate)
	workingCondition := storage.NewWorkingCondition(store)
	serviceWorkingCondition := service.NewWorkingCondition(workingCondition)
	handlerWorkingCondition := handler.NewWorkingCondition(serverServer, serviceWorkingCondition)
	agentWorkingConditions := storage.NewAgentWorkingConditions(store)
	client := cmdResources.engine
	serviceAgentWorkingConditions := service.NewAgentWorkingConditions(agentWorkingConditions, client)
	handlerAgentWorkingConditions := handler.NewAgentWorkingConditions(serverServer, serviceAgentWorkingConditions)
	agentAbsence := storage.NewAgentAbsence(store, manager)
	audit := cmdResources.audit
	serviceAgentAbsence := service.NewAgentAbsence(agentAbsence, audit, client)
	handlerAgentAbsence := handler.NewAgentAbsence(serverServer, serviceAgentAbsence)
	forecastCalculation := storage.NewForecastCalculation(store, manager, forecastStore)
	serviceForecastCalculation := service.NewForecastCalculation(forecastCalculation)
	handlerForecastCalculation := handler.NewForecastCalculation(serverServer, serviceForecastCalculation)
	workingSchedule := storage.NewWorkingSchedule(store, manager)
	serviceWorkingSchedule := service.NewWorkingSchedule(workingSchedule, client)
	handlerWorkingSchedule := handler.NewWorkingSchedule(serverServer, serviceWorkingSchedule)
	agentWorkingSchedule := storage.NewAgentWorkingSchedule(store, manager)
	serviceAgentWorkingSchedule := service.NewAgentWorkingSchedule(agentWorkingSchedule, workingSchedule, client)
	handlerAgentWorkingSchedule := handler.NewAgentWorkingSchedule(serverServer, serviceAgentWorkingSchedule)
	handlers := &handler.Handlers{
		PauseTemplate:          handlerPauseTemplate,
		ShiftTemplate:          handlerShiftTemplate,
		WorkingCondition:       handlerWorkingCondition,
		AgentWorkingConditions: handlerAgentWorkingConditions,
		AgentAbsence:           handlerAgentAbsence,
		ForecastCalculation:    handlerForecastCalculation,
		WorkingSchedule:        handlerWorkingSchedule,
		AgentWorkingSchedule:   handlerAgentWorkingSchedule,
	}
	return handlers, nil
}
