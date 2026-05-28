package projects

import (
	"context"
	"fmt"

	analyzerv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/analyzer/v1"
	gatewayv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/gateway/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	gatewayv1.UnimplementedProjectsServiceServer

	analytics gatewayv1.AnalyticsServiceServer
	analyzer  analyzerv1.AnalyzerServiceClient
}

func New(
	analytics gatewayv1.AnalyticsServiceServer,
	analyzer analyzerv1.AnalyzerServiceClient,
) *Server {
	return &Server{
		UnimplementedProjectsServiceServer: gatewayv1.UnimplementedProjectsServiceServer{},
		analytics:                          analytics,
		analyzer:                           analyzer,
	}
}

func (s *Server) ListProjects(
	ctx context.Context,
	_ *emptypb.Empty,
) (*gatewayv1.ListProjectsResponse, error) {
	resp, err := s.analyzer.ListProjects(ctx, &analyzerv1.ListProjectsRequest{})
	if err != nil {
		return nil, fmt.Errorf("analyzer.ListProjects: %w", err)
	}

	projects := make([]*gatewayv1.ProjectSummary, 0, len(resp.GetProjects()))
	for _, p := range resp.GetProjects() {
		projects = append(projects, &gatewayv1.ProjectSummary{
			Id:   p.GetProjectId(),
			Key:  p.GetProjectKey(),
			Name: p.GetProjectKey(),
			Url:  "",
		})
	}

	return &gatewayv1.ListProjectsResponse{Projects: projects}, nil
}

func (s *Server) GetProjectStats(
	ctx context.Context,
	req *gatewayv1.GetProjectStatsRequest,
) (*gatewayv1.ProjectStats, error) {
	resp, err := s.analytics.GetStats(ctx, &gatewayv1.GetStatsRequest{ProjectId: req.GetId()})
	if err != nil {
		return nil, fmt.Errorf("analytics.GetStats: %w", err)
	}

	return resp.GetStats(), nil
}

func (s *Server) DeleteProject(
	ctx context.Context,
	req *gatewayv1.DeleteProjectRequest,
) (*emptypb.Empty, error) {
	_, err := s.analyzer.DeleteProject(ctx, &analyzerv1.DeleteProjectRequest{
		ProjectId: req.GetId(),
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			//nolint:wrapcheck // pass status.Error through to grpc-gateway error handler
			return nil, err
		}

		return nil, fmt.Errorf("analyzer.DeleteProject: %w", err)
	}

	return &emptypb.Empty{}, nil
}
