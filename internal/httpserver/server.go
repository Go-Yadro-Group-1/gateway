package httpserver

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const readHeaderTimeout = 10 * time.Second

//go:embed openapi/gateway.swagger.json
var openapiFS embed.FS

type Config struct {
	Host    string
	Port    int
	Handler http.Handler
	Logger  *slog.Logger
}

func New(cfg Config) (*http.Server, error) {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(requestLogger(cfg.Logger))

	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	openapi, err := fs.Sub(openapiFS, "openapi")
	if err != nil {
		return nil, fmt.Errorf("embed openapi: %w", err)
	}

	router.Handle("/openapi.json", openapiHandler(openapi))

	router.Mount("/", cfg.Handler)

	//nolint:exhaustruct // unset http.Server fields keep their zero defaults
	return &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}, nil
}

func openapiHandler(spec fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		http.ServeFileFS(w, req, spec, "gateway.swagger.json")
	}
}

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			wrapped := middleware.NewWrapResponseWriter(w, req.ProtoMajor)

			next.ServeHTTP(wrapped, req)

			log.LogAttrs(req.Context(), slog.LevelInfo, "http request",
				slog.String("request_id", middleware.GetReqID(req.Context())),
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", wrapped.Status()),
				slog.Int("bytes", wrapped.BytesWritten()),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}
