package projects

import (
	"context"
	"fmt"

	gatewayv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/gateway/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	gatewayv1.UnimplementedProjectsServiceServer

	analytics gatewayv1.AnalyticsServiceServer
}

func New(analytics gatewayv1.AnalyticsServiceServer) *Server {
	return &Server{
		UnimplementedProjectsServiceServer: gatewayv1.UnimplementedProjectsServiceServer{},
		analytics:                          analytics,
	}
}

func (s *Server) ListProjects(
	_ context.Context,
	_ *emptypb.Empty,
) (*gatewayv1.ListProjectsResponse, error) {
	//nolint:wrapcheck // pass status.Error through to grpc-gateway error handler
	return nil, status.Error(codes.Unimplemented, "analyzer.ListProjects RPC is not yet available")
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
	_ context.Context,
	_ *gatewayv1.DeleteProjectRequest,
) (*emptypb.Empty, error) {
	//nolint:wrapcheck // pass status.Error through to grpc-gateway error handler
	return nil, status.Error(codes.Unimplemented, "analyzer.DeleteProject RPC is not yet available")
}
