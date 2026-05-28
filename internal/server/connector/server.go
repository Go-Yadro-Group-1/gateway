package connector

import (
	"context"
	"fmt"
	"log/slog"

	analyzerv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/analyzer/v1"
	connectorv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/connector/v1"
	gatewayv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/gateway/v1"
)

type Server struct {
	gatewayv1.UnimplementedConnectorServiceServer

	connector connectorv1.ConnectorServiceClient
	analyzer  analyzerv1.AnalyzerServiceClient
}

func New(
	connector connectorv1.ConnectorServiceClient,
	analyzer analyzerv1.AnalyzerServiceClient,
) *Server {
	return &Server{
		UnimplementedConnectorServiceServer: gatewayv1.UnimplementedConnectorServiceServer{},
		connector:                           connector,
		analyzer:                            analyzer,
	}
}

const defaultPageSize int32 = 50

func (s *Server) ListJiraProjects(
	ctx context.Context,
	req *gatewayv1.ListJiraProjectsRequest,
) (*gatewayv1.ListJiraProjectsResponse, error) {
	resp, err := s.connector.GetAvailableProjects(ctx, &connectorv1.GetAvailableProjectsRequest{
		SearchQuery: req.GetSearch(),
		Limit:       req.GetLimit(),
		Page:        req.GetPage(),
	})
	if err != nil {
		return nil, fmt.Errorf("connector.GetAvailableProjects: %w", err)
	}

	// Build a set of project keys known to analyzer. A single call per request.
	// On failure, degrade gracefully: all existence flags stay false.
	knownKeys := s.fetchKnownKeys(ctx)

	// Connector's upstream Jira endpoint (/rest/api/2/project) is non-paginated
	// and ignores startAt/maxResults, so it returns the full project list every time.
	// Slice it here until connector grows real pagination.
	upstream := resp.GetProjects()
	page, limit := normalizePaging(req.GetPage(), req.GetLimit())
	start, end := pageBounds(int32(len(upstream)), page, limit) //nolint:gosec

	projects := make([]*gatewayv1.JiraProject, 0, end-start)
	for _, p := range upstream[start:end] {
		_, exists := knownKeys[p.GetKey()]
		projects = append(projects, &gatewayv1.JiraProject{
			Key:       p.GetKey(),
			Name:      p.GetTitle(),
			SelfUrl:   p.GetSelf(),
			Existence: exists,
		})
	}

	return &gatewayv1.ListJiraProjectsResponse{
		Projects: projects,
		PageInfo: &gatewayv1.PageInfo{
			TotalCount: int32(len(upstream)),        //nolint:gosec
			IsLast:     end == int32(len(upstream)), //nolint:gosec
		},
	}, nil
}

func (s *Server) SyncProject(
	ctx context.Context,
	req *gatewayv1.SyncProjectRequest,
) (*gatewayv1.SyncProjectResponse, error) {
	resp, err := s.connector.DownloadProject(ctx, &connectorv1.DownloadProjectRequest{
		ProjectKey: req.GetProjectKey(),
	})
	if err != nil {
		return nil, fmt.Errorf("connector.DownloadProject: %w", err)
	}

	return &gatewayv1.SyncProjectResponse{
		ProjectKey: req.GetProjectKey(),
		Message:    resp.GetMessage(),
	}, nil
}

// fetchKnownKeys calls analyzer.ListProjects and returns a set of project keys (key → project_id).
// On any error it logs and returns an empty map so the caller can degrade gracefully.
func (s *Server) fetchKnownKeys(ctx context.Context) map[string]int64 {
	resp, err := s.analyzer.ListProjects(ctx, &analyzerv1.ListProjectsRequest{})
	if err != nil {
		slog.WarnContext(
			ctx,
			"analyzer.ListProjects failed; existence will be false for all projects",
			"err", err,
		)

		return map[string]int64{}
	}

	keys := make(map[string]int64, len(resp.GetProjects()))
	for _, p := range resp.GetProjects() {
		keys[p.GetProjectKey()] = p.GetProjectId()
	}

	return keys
}

func normalizePaging(page, limit int32) (int32, int32) {
	if page < 0 {
		page = 0
	}

	if limit <= 0 {
		limit = defaultPageSize
	}

	return page, limit
}

func pageBounds(total, page, limit int32) (int32, int32) {
	start := page * limit
	if start >= total {
		return total, total
	}

	end := min(start+limit, total)

	return start, end
}
