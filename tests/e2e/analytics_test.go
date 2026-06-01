//go:build e2e

package e2e_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	analyzerv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/analyzer/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getStatsRespJSON matches gateway.v1.GetStatsResponse JSON envelope.
type getStatsRespJSON struct {
	Stats projectStatsJSON `json:"stats"`
}

// getChartRespJSON matches gateway.v1.GetChartResponse JSON envelope.
// grpc-gateway serialises []byte fields as base64-encoded strings.
type getChartRespJSON struct {
	Data string `json:"data"` // base64-encoded bytes
}

// compareProjectsRespJSON matches gateway.v1.CompareProjectsResponse JSON.
type compareProjectsRespJSON struct {
	ProjectA projectStatsJSON `json:"projectA"`
	ProjectB projectStatsJSON `json:"projectB"`
}

func TestAnalytics_GetStats_MapsFields(t *testing.T) {
	t.Parallel()

	gw := setupGateway(t)
	gw.AnalyzerStub.setStatsResp(&analyzerv1.GetStatsResponse{
		Stats: &analyzerv1.Stats{
			CountTotal:      100,
			CountOpen:       20,
			CountClosed:     80,
			CountReopened:   5,
			CountResolved:   10,
			CountInProgress: 3,
		},
	})

	resp, err := http.Get(gw.Server.URL + "/api/v1/analytics/stats?projectId=1")
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-Id"))

	var body getStatsRespJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	assert.Equal(t, "1", body.Stats.ProjectID) // int64 → string in protobuf JSON
	assert.Equal(t, int32(100), body.Stats.CountTotal)
	assert.Equal(t, int32(20), body.Stats.CountOpen)
	assert.Equal(t, int32(80), body.Stats.CountClosed)
}

func TestAnalytics_GetChart_DataIsBase64Encoded(t *testing.T) {
	t.Parallel()

	rawJSON := []byte(`{"type":"bar","data":[1,2,3]}`)

	gw := setupGateway(t)
	gw.AnalyzerStub.setChartData(rawJSON)

	resp, err := http.Get(gw.Server.URL + "/api/v1/analytics/charts?projectId=1&chartType=1")
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body getChartRespJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	// grpc-gateway encodes []byte as standard base64; decode and compare to original bytes.
	decoded, err := base64.StdEncoding.DecodeString(body.Data)
	require.NoError(t, err, "data field must be valid base64")
	assert.Equal(t, rawJSON, decoded, "decoded chart data must equal what the stub returned")
}

func TestAnalytics_CompareProjects_ReturnsBothStatsBlocks(t *testing.T) {
	t.Parallel()

	gw := setupGateway(t)
	gw.AnalyzerStub.setCompareResp(&analyzerv1.CompareProjectsResponse{
		ProjectA: &analyzerv1.Stats{
			CountTotal:  10,
			CountClosed: 7,
		},
		ProjectB: &analyzerv1.Stats{
			CountTotal:  20,
			CountClosed: 15,
		},
	})

	resp, err := http.Get(gw.Server.URL + "/api/v1/analytics/compare?projectIdA=1&projectIdB=2")
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body compareProjectsRespJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	assert.Equal(t, "1", body.ProjectA.ProjectID) // int64 → string in protobuf JSON
	assert.Equal(t, int32(10), body.ProjectA.CountTotal)
	assert.Equal(t, int32(7), body.ProjectA.CountClosed)

	assert.Equal(t, "2", body.ProjectB.ProjectID) // int64 → string in protobuf JSON
	assert.Equal(t, int32(20), body.ProjectB.CountTotal)
	assert.Equal(t, int32(15), body.ProjectB.CountClosed)
}
