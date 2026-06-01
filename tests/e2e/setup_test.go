//go:build e2e

package e2e_test

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	analyzerv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/analyzer/v1"
	connectorv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/connector/v1"
	gatewayv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/gateway/v1"
	"github.com/Go-Yadro-Group-1/gateway/internal/server/analytics"
	connectorserver "github.com/Go-Yadro-Group-1/gateway/internal/server/connector"
	"github.com/Go-Yadro-Group-1/gateway/internal/server/projects"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1 << 20 // 1 MB

// gatewayUnderTest bundles all handles needed to drive a test.
type gatewayUnderTest struct {
	Server          *httptest.Server
	AnalyzerStub    *analyzerStub
	ConnectorStub   *connectorStub
}

// setupGateway spins up:
//   - two in-process gRPC servers backed by configurable stubs
//   - three gateway server implementations wired to those stubs
//   - a runtime.ServeMux with chi middleware on top
//   - an httptest.Server
//
// Call t.Cleanup on the returned teardown automatically via testing.TB.Cleanup.
func setupGateway(t *testing.T) *gatewayUnderTest {
	t.Helper()

	// --- analyzer bufconn ---
	analyzerLis := bufconn.Listen(bufSize)
	aStub := newAnalyzerStub()
	analyzerGRPC := grpc.NewServer()
	analyzerv1.RegisterAnalyzerServiceServer(analyzerGRPC, aStub)

	go func() {
		_ = analyzerGRPC.Serve(analyzerLis)
	}()

	// --- connector bufconn ---
	connectorLis := bufconn.Listen(bufSize)
	cStub := newConnectorStub()
	connectorGRPC := grpc.NewServer()
	connectorv1.RegisterConnectorServiceServer(connectorGRPC, cStub)

	go func() {
		_ = connectorGRPC.Serve(connectorLis)
	}()

	// --- gRPC client connections via bufconn ---
	// passthrough:/// resolver is required so grpc.NewClient resolves the
	// address to a single endpoint and hands it to the custom ContextDialer.
	analyzerConn, err := grpc.NewClient(
		"passthrough:///bufnet-analyzer",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			return analyzerLis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	connectorConn, err := grpc.NewClient(
		"passthrough:///bufnet-connector",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			return connectorLis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	// --- gateway server implementations ---
	analyzerClient := analyzerv1.NewAnalyzerServiceClient(analyzerConn)
	connectorClient := connectorv1.NewConnectorServiceClient(connectorConn)

	analyticsSrv := analytics.New(analyzerClient)
	connectorSrv := connectorserver.New(connectorClient, analyzerClient)
	projectsSrv := projects.New(analyticsSrv, analyzerClient)

	// --- grpc-gateway mux ---
	mux := runtime.NewServeMux()
	ctx := context.Background()

	require.NoError(t, gatewayv1.RegisterAnalyticsServiceHandlerServer(ctx, mux, analyticsSrv))
	require.NoError(t, gatewayv1.RegisterConnectorServiceHandlerServer(ctx, mux, connectorSrv))
	require.NoError(t, gatewayv1.RegisterProjectsServiceHandlerServer(ctx, mux, projectsSrv))

	// --- chi router (mirrors internal/httpserver.New) ---
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(requestLogger(slog.Default()))
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	router.Mount("/", mux)

	// --- httptest server ---
	srv := httptest.NewServer(router)

	t.Cleanup(func() {
		srv.Close()
		analyzerGRPC.Stop()
		connectorGRPC.Stop()
		_ = analyzerConn.Close()
		_ = connectorConn.Close()
		_ = analyzerLis.Close()
		_ = connectorLis.Close()
	})

	return &gatewayUnderTest{
		Server:        srv,
		AnalyzerStub:  aStub,
		ConnectorStub: cStub,
	}
}

// requestLogger mirrors internal/httpserver's requestLogger. It additionally
// echoes the request ID into the response header before delegating to next,
// so that e2e assertions can verify the RequestID middleware is active.
func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			reqID := middleware.GetReqID(req.Context())
			// Set before calling next so the header is included in the response.
			w.Header().Set("X-Request-Id", reqID)

			wrapped := middleware.NewWrapResponseWriter(w, req.ProtoMajor)
			next.ServeHTTP(wrapped, req)

			log.InfoContext(req.Context(), "http request",
				"request_id", reqID,
				"method", req.Method,
				"path", req.URL.Path,
				"status", wrapped.Status(),
			)
		})
	}
}
