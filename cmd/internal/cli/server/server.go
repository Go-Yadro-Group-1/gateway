package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Go-Yadro-Group-1/gateway/internal/app"
	"github.com/Go-Yadro-Group-1/gateway/internal/config"
	"github.com/Go-Yadro-Group-1/gateway/internal/httpserver"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultConfig   = "config/dev.yaml"
	shutdownTimeout = 15 * time.Second
)

//nolint:exhaustruct
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "serve",
		Short:        "Start the Jira Gateway HTTP server",
		Long:         "Start the Jira Gateway HTTP server.",
		RunE:         run,
		SilenceUsage: true,
	}

	cmd.Flags().String("config", defaultConfig, "path to config file")

	return cmd
}

func run(cmd *cobra.Command, _ []string) error {
	// Best-effort: load a local .env if present. godotenv does not override
	// variables already set in the environment, so platform-injected env
	// (e.g. Timeweb) keeps precedence; a missing file is not an error.
	_ = godotenv.Load()

	cfg, err := loadConfig(cmd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := newLogger(cfg.App.LogLevel)

	rootCtx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application, err := app.New(rootCtx, cfg)
	if err != nil {
		return fmt.Errorf("init app: %w", err)
	}

	defer func() {
		closeErr := application.Close()
		if closeErr != nil {
			logger.Warn("app close", slog.String("error", closeErr.Error()))
		}
	}()

	srv, err := httpserver.New(httpserver.Config{
		Host:    cfg.HTTP.Host,
		Port:    cfg.HTTP.Port,
		Handler: application.Handler(),
		Logger:  logger,
	})
	if err != nil {
		return fmt.Errorf("build http server: %w", err)
	}

	logger.Info("starting HTTP server",
		slog.String("addr", srv.Addr),
		slog.String("analyzer", cfg.Analyzer.Address),
		slog.String("connector", cfg.Connector.Address),
	)

	return serve(rootCtx, srv, logger)
}

func serve(ctx context.Context, srv *http.Server, logger *slog.Logger) error {
	errCh := make(chan error, 1)

	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err

			return
		}

		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown requested", slog.String("reason", ctx.Err().Error()))

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		//nolint:contextcheck // parent ctx is already cancelled; need a fresh deadline
		err := srv.Shutdown(shutdownCtx)
		if err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}

		return <-errCh
	case err := <-errCh:
		return err
	}
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	cfgFile, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, fmt.Errorf("get config flag: %w", err)
	}

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	viper.AutomaticEnv()

	err = config.BindEnvs()
	if err != nil {
		return nil, fmt.Errorf("bind envs: %w", err)
	}

	// A missing config file is not fatal: the whole config can be supplied via
	// the environment (.env). Only surface real read/parse errors.
	err = viper.ReadInConfig()
	if err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level

	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       lvl,
		ReplaceAttr: nil,
	}))
}
