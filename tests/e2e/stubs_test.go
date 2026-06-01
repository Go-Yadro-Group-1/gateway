//go:build e2e

package e2e_test

import (
	"context"
	"sync"

	analyzerv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/analyzer/v1"
	connectorv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/connector/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// analyzerStub is a minimal, thread-safe in-process implementation of
// analyzerv1.AnalyzerServiceServer. Tests configure responses via the exported
// fields before making HTTP calls.
type analyzerStub struct {
	analyzerv1.UnimplementedAnalyzerServiceServer

	mu sync.Mutex

	// GetChart
	chartData []byte
	chartErr  error

	// GetStats
	statsResp *analyzerv1.GetStatsResponse
	statsErr  error

	// CompareProjects
	compareResp *analyzerv1.CompareProjectsResponse
	compareErr  error

	// ListProjects
	listProjectsResp *analyzerv1.ListProjectsResponse
	listProjectsErr  error

	// DeleteProject
	deleteProjectErr error

	// captured calls
	deletedProjectIDs []int64
}

func newAnalyzerStub() *analyzerStub {
	return &analyzerStub{}
}

func (s *analyzerStub) setChartData(data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.chartData = data
	s.chartErr = nil
}

func (s *analyzerStub) setStatsResp(r *analyzerv1.GetStatsResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.statsResp = r
	s.statsErr = nil
}

func (s *analyzerStub) setStatsErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.statsErr = err
}

func (s *analyzerStub) setCompareResp(r *analyzerv1.CompareProjectsResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.compareResp = r
	s.compareErr = nil
}

func (s *analyzerStub) setListProjectsResp(r *analyzerv1.ListProjectsResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.listProjectsResp = r
	s.listProjectsErr = nil
}

func (s *analyzerStub) setListProjectsErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.listProjectsErr = err
}

func (s *analyzerStub) setDeleteProjectErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deleteProjectErr = err
}

func (s *analyzerStub) getDeletedIDs() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]int64, len(s.deletedProjectIDs))
	copy(out, s.deletedProjectIDs)

	return out
}

func (s *analyzerStub) GetChart(_ context.Context, _ *analyzerv1.GetChartRequest) (*analyzerv1.GetChartResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.chartErr != nil {
		return nil, s.chartErr
	}

	return &analyzerv1.GetChartResponse{Data: s.chartData}, nil
}

func (s *analyzerStub) GetStats(_ context.Context, _ *analyzerv1.GetStatsRequest) (*analyzerv1.GetStatsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.statsErr != nil {
		return nil, s.statsErr
	}

	if s.statsResp != nil {
		return s.statsResp, nil
	}

	return &analyzerv1.GetStatsResponse{}, nil
}

func (s *analyzerStub) CompareProjects(_ context.Context, _ *analyzerv1.CompareProjectsRequest) (*analyzerv1.CompareProjectsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.compareErr != nil {
		return nil, s.compareErr
	}

	if s.compareResp != nil {
		return s.compareResp, nil
	}

	return &analyzerv1.CompareProjectsResponse{}, nil
}

func (s *analyzerStub) ListProjects(_ context.Context, _ *analyzerv1.ListProjectsRequest) (*analyzerv1.ListProjectsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listProjectsErr != nil {
		return nil, s.listProjectsErr
	}

	if s.listProjectsResp != nil {
		return s.listProjectsResp, nil
	}

	return &analyzerv1.ListProjectsResponse{}, nil
}

func (s *analyzerStub) DeleteProject(_ context.Context, req *analyzerv1.DeleteProjectRequest) (*analyzerv1.DeleteProjectResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.deleteProjectErr != nil {
		return nil, s.deleteProjectErr
	}

	s.deletedProjectIDs = append(s.deletedProjectIDs, req.GetProjectId())

	return &analyzerv1.DeleteProjectResponse{}, nil
}

// connectorStub is a minimal in-process ConnectorServiceServer.
type connectorStub struct {
	connectorv1.UnimplementedConnectorServiceServer

	mu sync.Mutex

	// GetAvailableProjects
	availableResp *connectorv1.GetAvailableProjectsResponse
	availableErr  error

	// DownloadProject
	downloadResp *connectorv1.DownloadProjectResponse
	downloadErr  error

	// captured calls
	downloadedKeys []string
}

func newConnectorStub() *connectorStub {
	return &connectorStub{}
}

func (s *connectorStub) setAvailableProjects(r *connectorv1.GetAvailableProjectsResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.availableResp = r
	s.availableErr = nil
}

func (s *connectorStub) setDownloadResp(r *connectorv1.DownloadProjectResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.downloadResp = r
	s.downloadErr = nil
}

func (s *connectorStub) getDownloadedKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]string, len(s.downloadedKeys))
	copy(out, s.downloadedKeys)

	return out
}

func (s *connectorStub) GetAvailableProjects(_ context.Context, _ *connectorv1.GetAvailableProjectsRequest) (*connectorv1.GetAvailableProjectsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.availableErr != nil {
		return nil, s.availableErr
	}

	if s.availableResp != nil {
		return s.availableResp, nil
	}

	return &connectorv1.GetAvailableProjectsResponse{}, nil
}

func (s *connectorStub) DownloadProject(_ context.Context, req *connectorv1.DownloadProjectRequest) (*connectorv1.DownloadProjectResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.downloadErr != nil {
		return nil, s.downloadErr
	}

	s.downloadedKeys = append(s.downloadedKeys, req.GetProjectKey())

	if s.downloadResp != nil {
		return s.downloadResp, nil
	}

	return &connectorv1.DownloadProjectResponse{}, nil
}

// notFoundError returns a gRPC status error with codes.NotFound.
func notFoundError(msg string) error {
	return status.Error(codes.NotFound, msg)
}

// internalError returns a gRPC status error with codes.Internal.
func internalError(msg string) error {
	return status.Error(codes.Internal, msg)
}
