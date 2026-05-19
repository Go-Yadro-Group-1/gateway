package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	gatewayv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/gateway/v1"
	analyzerclient "github.com/Go-Yadro-Group-1/gateway/internal/clients/analyzer"
	connectorclient "github.com/Go-Yadro-Group-1/gateway/internal/clients/connector"
	"github.com/Go-Yadro-Group-1/gateway/internal/config"
	"github.com/Go-Yadro-Group-1/gateway/internal/server/analytics"
	connectorserver "github.com/Go-Yadro-Group-1/gateway/internal/server/connector"
	"github.com/Go-Yadro-Group-1/gateway/internal/server/projects"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type App struct {
	mux *runtime.ServeMux

	analyzer  *analyzerclient.Client
	connector *connectorclient.Client
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	analyzerCli, err := analyzerclient.Dial(cfg.Analyzer.Address)
	if err != nil {
		return nil, fmt.Errorf("dial analyzer: %w", err)
	}

	connectorCli, err := connectorclient.Dial(cfg.Connector.Address)
	if err != nil {
		_ = analyzerCli.Close()

		return nil, fmt.Errorf("dial connector: %w", err)
	}

	analyticsSrv := analytics.New(analyzerCli.Analyzer())
	connectorSrv := connectorserver.New(connectorCli.Connector())
	projectsSrv := projects.New(analyticsSrv)

	mux := runtime.NewServeMux()

	err = registerHandlers(ctx, mux, analyticsSrv, connectorSrv, projectsSrv)
	if err != nil {
		_ = analyzerCli.Close()
		_ = connectorCli.Close()

		return nil, err
	}

	return &App{
		mux:       mux,
		analyzer:  analyzerCli,
		connector: connectorCli,
	}, nil
}

func registerHandlers(
	ctx context.Context,
	mux *runtime.ServeMux,
	analyticsSrv gatewayv1.AnalyticsServiceServer,
	connectorSrv gatewayv1.ConnectorServiceServer,
	projectsSrv gatewayv1.ProjectsServiceServer,
) error {
	err := gatewayv1.RegisterAnalyticsServiceHandlerServer(ctx, mux, analyticsSrv)
	if err != nil {
		return fmt.Errorf("register analytics handler: %w", err)
	}

	err = gatewayv1.RegisterConnectorServiceHandlerServer(ctx, mux, connectorSrv)
	if err != nil {
		return fmt.Errorf("register connector handler: %w", err)
	}

	err = gatewayv1.RegisterProjectsServiceHandlerServer(ctx, mux, projectsSrv)
	if err != nil {
		return fmt.Errorf("register projects handler: %w", err)
	}

	return nil
}

func (a *App) Handler() http.Handler {
	return a.mux
}

func (a *App) Close() error {
	return errors.Join(a.analyzer.Close(), a.connector.Close())
}
