//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

//go:generate wire

package main

import (
	"ReservoirFloodPrediction/internal/biz"
	"ReservoirFloodPrediction/internal/conf"
	"ReservoirFloodPrediction/internal/data"
	"ReservoirFloodPrediction/internal/server"
	"ReservoirFloodPrediction/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// Provider function to extract Server config from Bootstrap
func provideServerConfig(bc *conf.Bootstrap) *conf.Server {
	return bc.Server
}

// Provider function to extract Data config from Bootstrap
func provideDataConfig(bc *conf.Bootstrap) *conf.Data {
	return bc.Data
}

// Provider function to extract Biz config from Bootstrap
func provideBizConfig(bc *conf.Bootstrap) *conf.Biz {
	if bc == nil {
		return nil
	}
	return bc.Biz
}

// wireApp init kratos application.
func wireApp(*conf.Bootstrap, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		provideServerConfig,
		provideDataConfig,
		provideBizConfig,
		server.ProviderSet,
		data.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		newApp,
	))
}
