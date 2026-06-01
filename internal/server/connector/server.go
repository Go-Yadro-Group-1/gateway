package connector

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	analyzerv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/analyzer/v1"
	connectorv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/connector/v1"
	gatewayv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/gateway/v1"
)

// projectsCacheTTL bounds how long the full Jira project list is reused before
// it is re-fetched from the connector. The list is large and changes rarely,
// while every catalog page/search request re-fetches it upstream (~2s), so a
// short TTL removes the repeated full load without serving stale data for long.
const projectsCacheTTL = 60 * time.Second

type projectsCacheEntry struct {
	projects  []*connectorv1.JiraProject
	fetchedAt time.Time
}

type Server struct {
	gatewayv1.UnimplementedConnectorServiceServer

	connector connectorv1.ConnectorServiceClient
	analyzer  analyzerv1.AnalyzerServiceClient

	mu           sync.Mutex
	projectCache map[string]projectsCacheEntry
}

func New(
	connector connectorv1.ConnectorServiceClient,
	analyzer analyzerv1.AnalyzerServiceClient,
) *Server {
	//nolint:exhaustruct // mu keeps its zero value; embedded Unimplemented set explicitly
	return &Server{
		UnimplementedConnectorServiceServer: gatewayv1.UnimplementedConnectorServiceServer{},
		connector:                           connector,
		analyzer:                            analyzer,
		projectCache:                        make(map[string]projectsCacheEntry),
	}
}

const defaultPageSize int32 = 50

func (s *Server) ListJiraProjects(
	ctx context.Context,
	req *gatewayv1.ListJiraProjectsRequest,
) (*gatewayv1.ListJiraProjectsResponse, error) {
	// Connector's upstream Jira endpoint (/rest/api/2/project) is non-paginated
	// and ignores startAt/maxResults, so it returns the full project list every
	// time. Fetch it (cached) and slice here until connector grows real pagination.
	upstream, err := s.availableProjects(ctx, req.GetSearch())
	if err != nil {
		return nil, err
	}

	// Build a set of project keys known to analyzer. A single call per request.
	// On failure, degrade gracefully: all existence flags stay false.
	knownKeys := s.fetchKnownKeys(ctx)

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
		SyncId:     resp.GetSyncId(),
		Status:     resp.GetStatus(),
	}, nil
}

func (s *Server) GetSyncStatus(
	ctx context.Context,
	req *gatewayv1.GetSyncStatusRequest,
) (*gatewayv1.GetSyncStatusResponse, error) {
	resp, err := s.connector.GetSyncStatus(ctx, &connectorv1.GetSyncStatusRequest{
		SyncId: req.GetSyncId(),
	})
	if err != nil {
		return nil, fmt.Errorf("connector.GetSyncStatus: %w", err)
	}

	return &gatewayv1.GetSyncStatusResponse{
		SyncId:     resp.GetSyncId(),
		State:      gatewayv1.SyncState(resp.GetState()),
		Processed:  resp.GetProcessed(),
		Total:      resp.GetTotal(),
		Error:      resp.GetError(),
		ProjectKey: resp.GetProjectKey(),
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

// availableProjects returns the full upstream Jira project list for a search
// query, served from a short-lived in-memory cache. The slow upstream call is
// made outside the lock so concurrent requests don't serialize on it; the
// existence flags layered on top (see fetchKnownKeys) are intentionally not
// cached and stay fresh per request.
func (s *Server) availableProjects(
	ctx context.Context,
	search string,
) ([]*connectorv1.JiraProject, error) {
	s.mu.Lock()
	entry, ok := s.projectCache[search]
	s.mu.Unlock()

	if ok && time.Since(entry.fetchedAt) < projectsCacheTTL {
		return entry.projects, nil
	}

	//nolint:exhaustruct // upstream ignores Limit/Page; only SearchQuery is meaningful here
	resp, err := s.connector.GetAvailableProjects(ctx, &connectorv1.GetAvailableProjectsRequest{
		SearchQuery: search,
	})
	if err != nil {
		return nil, fmt.Errorf("connector.GetAvailableProjects: %w", err)
	}

	projects := resp.GetProjects()

	s.mu.Lock()
	s.projectCache[search] = projectsCacheEntry{projects: projects, fetchedAt: time.Now()}
	s.mu.Unlock()

	return projects, nil
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
