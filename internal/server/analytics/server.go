package analytics

import (
	"context"
	"fmt"
	"math"

	analyzerv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/analyzer/v1"
	gatewayv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/gateway/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	gatewayv1.UnimplementedAnalyticsServiceServer

	analyzer analyzerv1.AnalyzerServiceClient
}

func New(analyzer analyzerv1.AnalyzerServiceClient) *Server {
	return &Server{
		UnimplementedAnalyticsServiceServer: gatewayv1.UnimplementedAnalyticsServiceServer{},
		analyzer:                            analyzer,
	}
}

func (s *Server) GetChart(
	ctx context.Context,
	req *gatewayv1.GetChartRequest,
) (*gatewayv1.GetChartResponse, error) {
	projectID, err := toInt32(req.GetProjectId())
	if err != nil {
		return nil, err
	}

	resp, err := s.analyzer.GetChart(ctx, &analyzerv1.GetChartRequest{
		ProjectId: projectID,
		ChartType: analyzerv1.ChartType(req.GetChartType()),
	})
	if err != nil {
		return nil, fmt.Errorf("analyzer.GetChart: %w", err)
	}

	return &gatewayv1.GetChartResponse{Data: resp.GetData()}, nil
}

func (s *Server) GetStats(
	ctx context.Context,
	req *gatewayv1.GetStatsRequest,
) (*gatewayv1.GetStatsResponse, error) {
	projectID, err := toInt32(req.GetProjectId())
	if err != nil {
		return nil, err
	}

	resp, err := s.analyzer.GetStats(ctx, &analyzerv1.GetStatsRequest{ProjectId: projectID})
	if err != nil {
		return nil, fmt.Errorf("analyzer.GetStats: %w", err)
	}

	return &gatewayv1.GetStatsResponse{
		Stats: statsToGateway(req.GetProjectId(), resp.GetStats()),
	}, nil
}

func (s *Server) CompareProjects(
	ctx context.Context,
	req *gatewayv1.CompareProjectsRequest,
) (*gatewayv1.CompareProjectsResponse, error) {
	idA, err := toInt32(req.GetProjectIdA())
	if err != nil {
		return nil, err
	}

	idB, err := toInt32(req.GetProjectIdB())
	if err != nil {
		return nil, err
	}

	resp, err := s.analyzer.CompareProjects(ctx, &analyzerv1.CompareProjectsRequest{
		ProjectIdA: idA,
		ProjectIdB: idB,
	})
	if err != nil {
		return nil, fmt.Errorf("analyzer.CompareProjects: %w", err)
	}

	return &gatewayv1.CompareProjectsResponse{
		ProjectA: statsToGateway(req.GetProjectIdA(), resp.GetProjectA()),
		ProjectB: statsToGateway(req.GetProjectIdB(), resp.GetProjectB()),
	}, nil
}

func statsToGateway(projectID int64, src *analyzerv1.Stats) *gatewayv1.ProjectStats {
	stats := &gatewayv1.ProjectStats{
		ProjectId:            projectID,
		CountTotal:           0,
		CountOpen:            0,
		CountClosed:          0,
		CountReopened:        0,
		CountResolved:        0,
		CountInProgress:      0,
		TotalDurationClosed:  0,
		CountCreatedLastWeek: 0,
	}
	if src == nil {
		return stats
	}

	stats.CountTotal = src.GetCountTotal()
	stats.CountOpen = src.GetCountOpen()
	stats.CountClosed = src.GetCountClosed()
	stats.CountReopened = src.GetCountReopened()
	stats.CountResolved = src.GetCountResolved()
	stats.CountInProgress = src.GetCountInProgress()

	return stats
}

func toInt32(projectID int64) (int32, error) {
	if projectID < math.MinInt32 || projectID > math.MaxInt32 {
		//nolint:wrapcheck // pass status.Error through to grpc-gateway error handler
		return 0, status.Errorf(
			codes.InvalidArgument,
			"project id %d out of int32 range",
			projectID,
		)
	}

	return int32(projectID), nil
}
