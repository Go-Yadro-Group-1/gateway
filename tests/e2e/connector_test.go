//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	analyzerv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/analyzer/v1"
	connectorv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/connector/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jiraProjectJSON matches gateway.v1.JiraProject camelCase JSON.
type jiraProjectJSON struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	SelfURL   string `json:"selfUrl"`
	Existence bool   `json:"existence"`
}

type pageInfoJSON struct {
	TotalCount int32 `json:"totalCount"`
	IsLast     bool  `json:"isLast"`
}

type listJiraProjectsRespJSON struct {
	Projects []jiraProjectJSON `json:"projects"`
	PageInfo pageInfoJSON      `json:"pageInfo"`
}

// makeConnectorProjects builds n connector JiraProject entries with keys "A1", "A2", …, "An".
func makeConnectorProjects(n int) []*connectorv1.JiraProject {
	projects := make([]*connectorv1.JiraProject, n)
	for i := range n {
		projects[i] = &connectorv1.JiraProject{
			Key:   fmt.Sprintf("A%d", i+1),
			Title: fmt.Sprintf("Project A%d", i+1),
			Self:  fmt.Sprintf("http://jira/A%d", i+1),
		}
	}

	return projects
}

// makeAnalyzerProjects builds analyzer ProjectInfo entries for the given keys.
func makeAnalyzerProjects(keys []string) []*analyzerv1.ProjectInfo {
	infos := make([]*analyzerv1.ProjectInfo, len(keys))
	for i, k := range keys {
		infos[i] = &analyzerv1.ProjectInfo{
			ProjectId:  int64(i + 1),
			ProjectKey: k,
		}
	}

	return infos
}

func TestConnector_ListJiraProjects_Page1_HappyPath(t *testing.T) {
	t.Parallel()

	gw := setupGateway(t)

	// connector returns 25 projects A1..A25
	gw.ConnectorStub.setAvailableProjects(&connectorv1.GetAvailableProjectsResponse{
		Projects: makeConnectorProjects(25),
	})

	// analyzer knows A1, A3, A5, A7, A9 (5 synced projects)
	gw.AnalyzerStub.setListProjectsResp(&analyzerv1.ListProjectsResponse{
		Projects: makeAnalyzerProjects([]string{"A1", "A3", "A5", "A7", "A9"}),
	})

	resp, err := http.Get(gw.Server.URL + "/api/v1/connector/projects?page=0&limit=10")
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body listJiraProjectsRespJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	require.Len(t, body.Projects, 10, "first page of 25 with limit=10 must return 10 items")
	assert.Equal(t, int32(25), body.PageInfo.TotalCount)
	assert.False(t, body.PageInfo.IsLast, "page 0 of 25 with limit=10 is not the last page")

	// A1 (index 0), A3 (index 2), A5 (index 4), A7 (index 6), A9 (index 8) are synced
	synced := map[string]bool{"A1": true, "A3": true, "A5": true, "A7": true, "A9": true}
	for _, p := range body.Projects {
		assert.Equal(t, synced[p.Key], p.Existence,
			"existence mismatch for project %s", p.Key)
	}
}

func TestConnector_ListJiraProjects_Page3_IsLast(t *testing.T) {
	t.Parallel()

	gw := setupGateway(t)

	gw.ConnectorStub.setAvailableProjects(&connectorv1.GetAvailableProjectsResponse{
		Projects: makeConnectorProjects(25),
	})
	gw.AnalyzerStub.setListProjectsResp(&analyzerv1.ListProjectsResponse{})

	resp, err := http.Get(gw.Server.URL + "/api/v1/connector/projects?page=2&limit=10")
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body listJiraProjectsRespJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	assert.Len(t, body.Projects, 5, "page 2 of 25 with limit=10 must have 5 items (25 - 20 = 5)")
	assert.True(t, body.PageInfo.IsLast, "page 2 of 25 with limit=10 is the last page")
	assert.Equal(t, int32(25), body.PageInfo.TotalCount)
}

func TestConnector_ListJiraProjects_AnalyzerDown_GracefulDegradation(t *testing.T) {
	t.Parallel()

	gw := setupGateway(t)

	gw.ConnectorStub.setAvailableProjects(&connectorv1.GetAvailableProjectsResponse{
		Projects: makeConnectorProjects(3),
	})
	// analyzer is unavailable — all existence flags must stay false
	gw.AnalyzerStub.setListProjectsErr(internalError("analyzer unavailable"))

	resp, err := http.Get(gw.Server.URL + "/api/v1/connector/projects?page=0&limit=10")
	require.NoError(t, err)

	defer resp.Body.Close()

	// graceful degradation: still HTTP 200
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body listJiraProjectsRespJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	require.Len(t, body.Projects, 3)
	for _, p := range body.Projects {
		assert.False(t, p.Existence,
			"when analyzer is unavailable existence must be false for all projects, got true for %s", p.Key)
	}
}

func TestConnector_SyncProject_ForwardsProjectKey(t *testing.T) {
	t.Parallel()

	gw := setupGateway(t)
	gw.ConnectorStub.setDownloadResp(&connectorv1.DownloadProjectResponse{
		Message: "sync started",
	})

	body := bytes.NewBufferString(`{"projectKey":"FOO"}`)
	resp, err := http.Post(gw.Server.URL+"/api/v1/connector/projects/sync", "application/json", body)
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	keys := gw.ConnectorStub.getDownloadedKeys()
	require.Len(t, keys, 1)
	assert.Equal(t, "FOO", keys[0], "connector stub must receive projectKey=FOO")
}
