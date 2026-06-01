//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	analyzerv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/analyzer/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// projectSummaryJSON matches the grpc-gateway JSON shape of gateway.v1.ProjectSummary.
// grpc-gateway uses protobuf JSON encoding: int64 fields are serialised as strings.
type projectSummaryJSON struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type listProjectsRespJSON struct {
	Projects []projectSummaryJSON `json:"projects"`
}

// projectStatsJSON matches gateway.v1.ProjectStats camelCase JSON.
// int64 fields (project_id) are strings in protobuf JSON.
type projectStatsJSON struct {
	ProjectID   string `json:"projectId"` // int64 → string in protobuf JSON
	CountTotal  int32  `json:"countTotal"`
	CountOpen   int32  `json:"countOpen"`
	CountClosed int32  `json:"countClosed"`
}

func TestProjects_ListProjects_ReturnsProjectsFromAnalyzer(t *testing.T) {
	t.Parallel()

	gw := setupGateway(t)
	gw.AnalyzerStub.setListProjectsResp(&analyzerv1.ListProjectsResponse{
		Projects: []*analyzerv1.ProjectInfo{
			{ProjectId: 1, ProjectKey: "ALPHA"},
			{ProjectId: 2, ProjectKey: "BETA"},
		},
	})

	resp, err := http.Get(gw.Server.URL + "/api/v1/projects")
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-Id"))

	var body listProjectsRespJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	require.Len(t, body.Projects, 2)
	assert.Equal(t, "1", body.Projects[0].ID)
	assert.Equal(t, "ALPHA", body.Projects[0].Key)
	assert.Equal(t, "2", body.Projects[1].ID)
	assert.Equal(t, "BETA", body.Projects[1].Key)
}

func TestProjects_GetProjectStats_ReturnsMappedFields(t *testing.T) {
	t.Parallel()

	gw := setupGateway(t)
	gw.AnalyzerStub.setStatsResp(&analyzerv1.GetStatsResponse{
		Stats: &analyzerv1.Stats{
			CountTotal:  42,
			CountOpen:   10,
			CountClosed: 32,
		},
	})

	resp, err := http.Get(gw.Server.URL + "/api/v1/projects/7")
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body projectStatsJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	assert.Equal(t, "7", body.ProjectID) // int64 → string in protobuf JSON
	assert.Equal(t, int32(42), body.CountTotal)
	assert.Equal(t, int32(10), body.CountOpen)
	assert.Equal(t, int32(32), body.CountClosed)
}

func TestProjects_GetProjectStats_NotFound(t *testing.T) {
	t.Parallel()

	gw := setupGateway(t)
	gw.AnalyzerStub.setStatsErr(notFoundError("project not found"))

	resp, err := http.Get(gw.Server.URL + "/api/v1/projects/99")
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestProjects_DeleteProject_ForwardsID(t *testing.T) {
	t.Parallel()

	gw := setupGateway(t)

	req, err := http.NewRequest(http.MethodDelete, gw.Server.URL+"/api/v1/projects/5", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	// grpc-gateway maps empty proto response to HTTP 200 with "{}" body
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	ids := gw.AnalyzerStub.getDeletedIDs()
	require.Len(t, ids, 1)
	assert.Equal(t, int64(5), ids[0], "stub must receive project_id=5")
}

func TestProjects_DeleteProject_NotFound(t *testing.T) {
	t.Parallel()

	gw := setupGateway(t)
	gw.AnalyzerStub.setDeleteProjectErr(notFoundError("project not found"))

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/projects/404", gw.Server.URL), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
