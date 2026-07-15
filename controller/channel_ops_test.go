package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetChannelOps_ReturnsRetryTimes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := common.RetryTimes
	common.RetryTimes = 7
	t.Cleanup(func() { common.RetryTimes = old })

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/channel/ops", nil)

	GetChannelOps(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			RetryTimes int `json:"retry_times"`
		} `json:"data"`
	}
	require.NoError(t, common.DecodeJson(recorder.Body, &response))
	require.True(t, response.Success)
	require.Equal(t, 7, response.Data.RetryTimes)
}
