package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func TestUpdateOptionRejectsDeprecatedFrontendOption(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/option/", strings.NewReader(`{"key":"theme.frontend","value":"default"}`))
	context.Request.Header.Set("Content-Type", "application/json")

	UpdateOption(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	var response struct {
		Success bool `json:"success"`
	}
	if err := common.DecodeJson(recorder.Body, &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Success {
		t.Fatal("deprecated option update unexpectedly succeeded")
	}
}
