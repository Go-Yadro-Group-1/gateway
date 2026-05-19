package connector

import (
	"context"
	"fmt"

	connectorv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/connector/v1"
	gatewayv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/gateway/v1"
)

type Server struct {
	gatewayv1.UnimplementedConnectorServiceServer

	connector connectorv1.ConnectorServiceClient
}

func New(connector connectorv1.ConnectorServiceClient) *Server {
	return &Server{
		UnimplementedConnectorServiceServer: gatewayv1.UnimplementedConnectorServiceServer{},
		connector:                           connector,
	}
}

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

	projects := make([]*gatewayv1.JiraProject, 0, len(resp.GetProjects()))
	for _, p := range resp.GetProjects() {
		projects = append(projects, &gatewayv1.JiraProject{
			Key:     p.GetKey(),
			Name:    p.GetTitle(),
			SelfUrl: p.GetSelf(),
			// Existence requires analyzer.ListProjects; always false until that RPC ships.
			Existence: false,
		})
	}

	return &gatewayv1.ListJiraProjectsResponse{
		Projects: projects,
		PageInfo: &gatewayv1.PageInfo{
			CurrentPage: req.GetPage(),
			PageCount:   pageCount(resp.GetTotal(), req.GetLimit()),
			TotalCount:  resp.GetTotal(),
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

func pageCount(total, limit int32) int32 {
	if limit <= 0 {
		return 0
	}

	return (total + limit - 1) / limit
}
