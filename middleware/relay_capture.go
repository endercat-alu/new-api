package middleware

import (
	"io"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func RelayCapture() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !service.IsRelayCaptureEnabled() || !isRelayCapturePath(c.Request.URL.Path) {
			c.Next()
			return
		}

		captureCtx := service.BeginRelayCaptureRecord(c)
		captureWriter := service.NewRelayCaptureResponseWriter(c.Writer, service.GetRelayCaptureMaxBodyBytes())
		c.Writer = captureWriter

		c.Next()

		requestBody, requestBodySize, requestBodyTruncated := captureStoredRequestBody(c, service.GetRelayCaptureMaxBodyBytes())
		statusCode := c.Writer.Status()
		if statusCode == 0 {
			statusCode = 200
		}
		service.FinishRelayCaptureRecord(
			captureCtx,
			c.FullPath(),
			requestBody,
			requestBodySize,
			requestBodyTruncated,
			statusCode,
			c.Writer.Header(),
			captureWriter.CapturedBody(),
			captureWriter.BodyBytes(),
			captureWriter.BodyTruncated(),
			c.Errors.String(),
		)
	}
}

func isRelayCapturePath(path string) bool {
	if path == "" {
		return false
	}
	if strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/v1beta/") || strings.HasPrefix(path, "/pg/") || strings.HasPrefix(path, "/mj/") || strings.HasPrefix(path, "/suno/") {
		return true
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) >= 2 && parts[1] == "mj"
}

func captureStoredRequestBody(c *gin.Context, maxBytes int64) ([]byte, int64, bool) {
	if c == nil || maxBytes <= 0 {
		return nil, 0, false
	}
	value, exists := c.Get(common.KeyBodyStorage)
	if !exists || value == nil {
		return nil, 0, false
	}
	storage, ok := value.(common.BodyStorage)
	if !ok || storage == nil {
		return nil, 0, false
	}

	size := storage.Size()
	current, err := storage.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, size, false
	}
	if _, err = storage.Seek(0, io.SeekStart); err != nil {
		return nil, size, false
	}
	defer func() {
		_, _ = storage.Seek(current, io.SeekStart)
	}()

	data, err := io.ReadAll(io.LimitReader(storage, maxBytes+1))
	if err != nil {
		return nil, size, false
	}
	truncated := size > maxBytes || int64(len(data)) > maxBytes
	if int64(len(data)) > maxBytes {
		data = data[:maxBytes]
	}
	return data, size, truncated
}
