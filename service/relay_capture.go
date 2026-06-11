package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

const (
	defaultRelayCaptureMaxRecords    = 100
	defaultRelayCaptureMaxBodyBytes  = int64(1 << 20)
	defaultRelayCaptureMaxTotalBytes = int64(50 << 20)
)

type RelayCaptureState struct {
	Enabled       bool  `json:"enabled"`
	StartedAt     int64 `json:"started_at,omitempty"`
	StoppedAt     int64 `json:"stopped_at,omitempty"`
	RecordCount   int   `json:"record_count"`
	MaxRecords    int   `json:"max_records"`
	MaxBodyBytes  int64 `json:"max_body_bytes"`
	MaxTotalBytes int64 `json:"max_total_bytes"`
	TotalBytes    int64 `json:"total_bytes"`
}

type RelayCaptureRecordsPage struct {
	Total int                  `json:"total"`
	Items []RelayCaptureRecord `json:"items"`
}

type RelayCaptureRecord struct {
	ID          string                    `json:"id"`
	StartedAt   int64                     `json:"started_at"`
	EndedAt     int64                     `json:"ended_at,omitempty"`
	DurationMs  int64                     `json:"duration_ms,omitempty"`
	Route       string                    `json:"route,omitempty"`
	Request     RelayCaptureHTTPRequest   `json:"request"`
	Response    *RelayCaptureHTTPResponse `json:"response,omitempty"`
	Error       string                    `json:"error,omitempty"`
	Truncated   bool                      `json:"truncated"`
	StoredBytes int64                     `json:"stored_bytes"`
}

type RelayCaptureHTTPRequest struct {
	Method            string              `json:"method"`
	URL               string              `json:"url"`
	Path              string              `json:"path"`
	Query             string              `json:"query,omitempty"`
	Proto             string              `json:"proto,omitempty"`
	Host              string              `json:"host,omitempty"`
	RemoteAddr        string              `json:"remote_addr,omitempty"`
	Headers           map[string][]string `json:"headers"`
	Body              string              `json:"body,omitempty"`
	BodyBase64        bool                `json:"body_base64,omitempty"`
	BodyTruncated     bool                `json:"body_truncated,omitempty"`
	BodyBytes         int64               `json:"body_bytes"`
	CapturedBodyBytes int64               `json:"captured_body_bytes"`
}

type RelayCaptureHTTPResponse struct {
	StatusCode        int                 `json:"status_code"`
	Headers           map[string][]string `json:"headers"`
	Body              string              `json:"body,omitempty"`
	BodyBase64        bool                `json:"body_base64,omitempty"`
	BodyTruncated     bool                `json:"body_truncated,omitempty"`
	BodyBytes         int64               `json:"body_bytes"`
	CapturedBodyBytes int64               `json:"captured_body_bytes"`
}

type RelayCaptureRecordContext struct {
	record *RelayCaptureRecord
}

type relayCaptureStore struct {
	mu            sync.RWMutex
	enabled       bool
	startedAt     int64
	stoppedAt     int64
	records       []RelayCaptureRecord
	totalBytes    int64
	maxRecords    int
	maxBodyBytes  int64
	maxTotalBytes int64
}

var (
	relayCaptureIDCounter atomic.Uint64
	relayCaptureStoreInst = &relayCaptureStore{
		maxRecords:    defaultRelayCaptureMaxRecords,
		maxBodyBytes:  defaultRelayCaptureMaxBodyBytes,
		maxTotalBytes: defaultRelayCaptureMaxTotalBytes,
	}
)

func StartRelayCapture() RelayCaptureState {
	relayCaptureStoreInst.mu.Lock()
	defer relayCaptureStoreInst.mu.Unlock()

	now := time.Now().UnixMilli()
	relayCaptureStoreInst.enabled = true
	relayCaptureStoreInst.startedAt = now
	relayCaptureStoreInst.stoppedAt = 0
	return relayCaptureStoreInst.stateLocked()
}

func StopRelayCapture() RelayCaptureState {
	relayCaptureStoreInst.mu.Lock()
	defer relayCaptureStoreInst.mu.Unlock()

	relayCaptureStoreInst.enabled = false
	relayCaptureStoreInst.stoppedAt = time.Now().UnixMilli()
	return relayCaptureStoreInst.stateLocked()
}

func ClearRelayCaptureRecords() RelayCaptureState {
	relayCaptureStoreInst.mu.Lock()
	defer relayCaptureStoreInst.mu.Unlock()

	relayCaptureStoreInst.records = nil
	relayCaptureStoreInst.totalBytes = 0
	return relayCaptureStoreInst.stateLocked()
}

func GetRelayCaptureState() RelayCaptureState {
	relayCaptureStoreInst.mu.RLock()
	defer relayCaptureStoreInst.mu.RUnlock()

	return relayCaptureStoreInst.stateLocked()
}

func ListRelayCaptureRecords(offset int, limit int) RelayCaptureRecordsPage {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	relayCaptureStoreInst.mu.RLock()
	defer relayCaptureStoreInst.mu.RUnlock()

	total := len(relayCaptureStoreInst.records)
	if offset >= total {
		return RelayCaptureRecordsPage{Total: total, Items: []RelayCaptureRecord{}}
	}

	end := offset + limit
	if end > total {
		end = total
	}

	items := make([]RelayCaptureRecord, 0, end-offset)
	for i := total - 1 - offset; i >= total-end; i-- {
		items = append(items, relayCaptureStoreInst.records[i])
	}

	return RelayCaptureRecordsPage{Total: total, Items: items}
}

func IsRelayCaptureEnabled() bool {
	relayCaptureStoreInst.mu.RLock()
	defer relayCaptureStoreInst.mu.RUnlock()

	return relayCaptureStoreInst.enabled
}

func GetRelayCaptureMaxBodyBytes() int64 {
	relayCaptureStoreInst.mu.RLock()
	defer relayCaptureStoreInst.mu.RUnlock()

	return relayCaptureStoreInst.maxBodyBytes
}

func BeginRelayCaptureRecord(c *gin.Context) *RelayCaptureRecordContext {
	if c == nil || c.Request == nil || !IsRelayCaptureEnabled() {
		return nil
	}

	requestURL := c.Request.URL.RequestURI()
	if requestURL == "" {
		requestURL = c.Request.URL.String()
	}

	record := &RelayCaptureRecord{
		ID:        nextRelayCaptureRecordID(),
		StartedAt: time.Now().UnixMilli(),
		Request: RelayCaptureHTTPRequest{
			Method:     c.Request.Method,
			URL:        requestURL,
			Path:       c.Request.URL.Path,
			Query:      c.Request.URL.RawQuery,
			Proto:      c.Request.Proto,
			Host:       c.Request.Host,
			RemoteAddr: c.Request.RemoteAddr,
			Headers:    cloneHeader(c.Request.Header),
		},
	}

	return &RelayCaptureRecordContext{record: record}
}

func FinishRelayCaptureRecord(ctx *RelayCaptureRecordContext, route string, requestBody []byte, requestBodySize int64, requestBodyTruncated bool, statusCode int, responseHeader http.Header, responseBody []byte, responseBodySize int64, responseBodyTruncated bool, errText string) {
	if ctx == nil || ctx.record == nil {
		return
	}

	now := time.Now().UnixMilli()
	record := *ctx.record
	record.Route = route
	record.EndedAt = now
	record.DurationMs = now - record.StartedAt
	record.Error = errText

	reqBody, reqBase64 := encodeRelayCaptureBody(requestBody)
	record.Request.Body = reqBody
	record.Request.BodyBase64 = reqBase64
	record.Request.BodyTruncated = requestBodyTruncated
	record.Request.BodyBytes = requestBodySize
	record.Request.CapturedBodyBytes = int64(len(requestBody))

	respBody, respBase64 := encodeRelayCaptureBody(responseBody)
	record.Response = &RelayCaptureHTTPResponse{
		StatusCode:        statusCode,
		Headers:           cloneHeader(responseHeader),
		Body:              respBody,
		BodyBase64:        respBase64,
		BodyTruncated:     responseBodyTruncated,
		BodyBytes:         responseBodySize,
		CapturedBodyBytes: int64(len(responseBody)),
	}

	record.Truncated = record.Request.BodyTruncated || record.Response.BodyTruncated
	record.StoredBytes = estimateRelayCaptureRecordBytes(&record)

	relayCaptureStoreInst.addRecord(record)
}

func (s *relayCaptureStore) addRecord(record RelayCaptureRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = append(s.records, record)
	s.totalBytes += record.StoredBytes
	s.trimLocked()
}

func (s *relayCaptureStore) trimLocked() {
	for s.maxRecords > 0 && len(s.records) > s.maxRecords {
		s.totalBytes -= s.records[0].StoredBytes
		s.records = s.records[1:]
	}
	for s.maxTotalBytes > 0 && len(s.records) > 1 && s.totalBytes > s.maxTotalBytes {
		s.totalBytes -= s.records[0].StoredBytes
		s.records = s.records[1:]
	}
	if s.totalBytes < 0 {
		s.totalBytes = 0
	}
}

func (s *relayCaptureStore) stateLocked() RelayCaptureState {
	return RelayCaptureState{
		Enabled:       s.enabled,
		StartedAt:     s.startedAt,
		StoppedAt:     s.stoppedAt,
		RecordCount:   len(s.records),
		MaxRecords:    s.maxRecords,
		MaxBodyBytes:  s.maxBodyBytes,
		MaxTotalBytes: s.maxTotalBytes,
		TotalBytes:    s.totalBytes,
	}
}

func nextRelayCaptureRecordID() string {
	seq := relayCaptureIDCounter.Add(1)
	return fmt.Sprintf("rc_%d_%d", time.Now().UnixNano(), seq)
}

func cloneHeader(header http.Header) map[string][]string {
	cloned := make(map[string][]string, len(header))
	for key, values := range header {
		copied := make([]string, len(values))
		copy(copied, values)
		cloned[key] = copied
	}
	return cloned
}

func encodeRelayCaptureBody(body []byte) (string, bool) {
	if len(body) == 0 {
		return "", false
	}
	if utf8.Valid(body) && isMostlyPrintableText(body) {
		return string(body), false
	}
	return base64.StdEncoding.EncodeToString(body), true
}

func isMostlyPrintableText(body []byte) bool {
	if len(body) == 0 {
		return true
	}
	control := 0
	for _, r := range string(body) {
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if r < 0x20 {
			control++
		}
	}
	return control*10 <= utf8.RuneCount(body)
}

func estimateRelayCaptureRecordBytes(record *RelayCaptureRecord) int64 {
	if record == nil {
		return 0
	}
	total := int64(len(record.ID) + len(record.Route) + len(record.Error))
	total += estimateHeaderBytes(record.Request.Headers)
	total += int64(len(record.Request.Method) + len(record.Request.URL) + len(record.Request.Path) + len(record.Request.Query) + len(record.Request.Proto) + len(record.Request.Host) + len(record.Request.RemoteAddr) + len(record.Request.Body))
	if record.Response != nil {
		total += estimateHeaderBytes(record.Response.Headers)
		total += int64(len(record.Response.Body))
	}
	return total
}

func estimateHeaderBytes(headers map[string][]string) int64 {
	var total int64
	for key, values := range headers {
		total += int64(len(key))
		for _, value := range values {
			total += int64(len(value))
		}
	}
	return total
}

type RelayCaptureResponseWriter struct {
	gin.ResponseWriter
	body      bytes.Buffer
	maxBytes  int64
	bodyBytes int64
	truncated bool
}

func NewRelayCaptureResponseWriter(writer gin.ResponseWriter, maxBytes int64) *RelayCaptureResponseWriter {
	return &RelayCaptureResponseWriter{ResponseWriter: writer, maxBytes: maxBytes}
}

func (w *RelayCaptureResponseWriter) Write(data []byte) (int, error) {
	w.capture(data)
	return w.ResponseWriter.Write(data)
}

func (w *RelayCaptureResponseWriter) WriteString(data string) (int, error) {
	w.capture([]byte(data))
	return w.ResponseWriter.WriteString(data)
}

func (w *RelayCaptureResponseWriter) Flush() {
	w.ResponseWriter.Flush()
}

func (w *RelayCaptureResponseWriter) CapturedBody() []byte {
	body := w.body.Bytes()
	copied := make([]byte, len(body))
	copy(copied, body)
	return copied
}

func (w *RelayCaptureResponseWriter) BodyTruncated() bool {
	return w.truncated
}

func (w *RelayCaptureResponseWriter) BodyBytes() int64 {
	return w.bodyBytes
}

func (w *RelayCaptureResponseWriter) capture(data []byte) {
	if len(data) == 0 {
		return
	}
	w.bodyBytes += int64(len(data))
	if w.maxBytes <= 0 {
		w.truncated = true
		return
	}
	remaining := int(w.maxBytes) - w.body.Len()
	if remaining <= 0 {
		w.truncated = true
		return
	}
	if len(data) > remaining {
		_, _ = w.body.Write(data[:remaining])
		w.truncated = true
		return
	}
	_, _ = w.body.Write(data)
}
