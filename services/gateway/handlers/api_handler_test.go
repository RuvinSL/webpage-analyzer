// package handlers

// import (
// 	"bytes"
// 	"encoding/json"
// 	"errors"
// 	"net/http"
// 	"net/http/httptest"
// 	"strings"
// 	"testing"
// 	"time"

// 	"github.com/RuvinSL/webpage-analyzer/pkg/mocks"
// 	"github.com/RuvinSL/webpage-analyzer/pkg/models"
// 	"github.com/golang/mock/gomock"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func TestNewAPIHandler(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	assert.NotNil(t, handler)
// 	assert.Equal(t, mockAnalyzerClient, handler.analyzerClient)
// 	assert.Equal(t, mockLogger, handler.logger)
// 	assert.Equal(t, mockMetrics, handler.metrics)
// }

// func TestAPIHandler_AnalyzeURL_Success(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	expectedResult := &models.AnalysisResult{
// 		URL:         "https://example.com",
// 		Title:       "Example Domain",
// 		HTMLVersion: "HTML5",
// 		Headings: map[string][]string{
// 			"h1": {"Example Domain"},
// 		},
// 		Links:        []models.Link{},
// 		HasLoginForm: false,
// 	}

// 	// Set up expectations
// 	mockLogger.EXPECT().Info("Processing analysis request", "url", "https://example.com").Times(1)
// 	mockAnalyzerClient.EXPECT().Analyze(gomock.Any(), "https://example.com").Return(expectedResult, nil).Times(1)

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	// Create request
// 	reqBody := models.AnalysisRequest{URL: "https://example.com"}
// 	jsonData, _ := json.Marshal(reqBody)
// 	req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(jsonData))
// 	req.Header.Set("Content-Type", "application/json")

// 	// Create response recorder
// 	w := httptest.NewRecorder()

// 	// Execute
// 	handler.AnalyzeURL(w, req)

// 	// Assertions
// 	assert.Equal(t, http.StatusOK, w.Code)
// 	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

// 	var result models.AnalysisResult
// 	err := json.NewDecoder(w.Body).Decode(&result)
// 	require.NoError(t, err)
// 	assert.Equal(t, expectedResult.URL, result.URL)
// 	assert.Equal(t, expectedResult.Title, result.Title)
// 	assert.Equal(t, expectedResult.HTMLVersion, result.HTMLVersion)
// }

// func TestAPIHandler_AnalyzeURL_InvalidJSON(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	// Set up expectations
// 	mockLogger.EXPECT().Error("Failed to parse request", "error", gomock.Any()).Times(1)
// 	mockLogger.EXPECT().Error("Failed to encode error response", "error", gomock.Any()).AnyTimes()

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	// Create request with invalid JSON
// 	req := httptest.NewRequest("POST", "/analyze", strings.NewReader("invalid json"))
// 	req.Header.Set("Content-Type", "application/json")

// 	w := httptest.NewRecorder()
// 	handler.AnalyzeURL(w, req)

// 	assert.Equal(t, http.StatusBadRequest, w.Code)

// 	var errorResp models.ErrorResponse
// 	err := json.NewDecoder(w.Body).Decode(&errorResp)
// 	require.NoError(t, err)
// 	assert.Equal(t, "Invalid request format", errorResp.Error)
// 	assert.Equal(t, http.StatusBadRequest, errorResp.StatusCode)
// }

// func TestAPIHandler_AnalyzeURL_EmptyURL(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	// Set up expectations
// 	mockLogger.EXPECT().Error("Failed to encode error response", "error", gomock.Any()).AnyTimes()

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	// Create request with empty URL
// 	reqBody := models.AnalysisRequest{URL: ""}
// 	jsonData, _ := json.Marshal(reqBody)
// 	req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(jsonData))
// 	req.Header.Set("Content-Type", "application/json")

// 	w := httptest.NewRecorder()
// 	handler.AnalyzeURL(w, req)

// 	assert.Equal(t, http.StatusBadRequest, w.Code)

// 	var errorResp models.ErrorResponse
// 	err := json.NewDecoder(w.Body).Decode(&errorResp)
// 	require.NoError(t, err)
// 	assert.Equal(t, "URL is required", errorResp.Error)
// }

// func TestAPIHandler_AnalyzeURL_AnalyzerError(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	// Set up expectations
// 	mockLogger.EXPECT().Info("Processing analysis request", "url", "https://example.com").Times(1)
// 	mockLogger.EXPECT().Error("Analysis failed", "url", "https://example.com", "error", gomock.Any()).Times(1)
// 	mockLogger.EXPECT().Error("Failed to encode error response", "error", gomock.Any()).AnyTimes()
// 	mockAnalyzerClient.EXPECT().Analyze(gomock.Any(), "https://example.com").Return(nil, errors.New("service unavailable")).Times(1)

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	reqBody := models.AnalysisRequest{URL: "https://example.com"}
// 	jsonData, _ := json.Marshal(reqBody)
// 	req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(jsonData))

// 	w := httptest.NewRecorder()
// 	handler.AnalyzeURL(w, req)

// 	assert.Equal(t, http.StatusInternalServerError, w.Code)

// 	var errorResp models.ErrorResponse
// 	err := json.NewDecoder(w.Body).Decode(&errorResp)
// 	require.NoError(t, err)
// 	assert.Contains(t, errorResp.Error, "Analysis failed: service unavailable")
// }

// func TestAPIHandler_AnalyzeURL_ContextTimeout(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	// Set up expectations
// 	mockLogger.EXPECT().Info("Processing analysis request", "url", "https://example.com").Times(1)
// 	mockLogger.EXPECT().Error("Analysis failed", "url", "https://example.com", "error", gomock.Any()).Times(1)
// 	mockLogger.EXPECT().Error("Failed to encode error response", "error", gomock.Any()).AnyTimes()
// 	mockAnalyzerClient.EXPECT().Analyze(gomock.Any(), "https://example.com").Return(nil, errors.New("context deadline exceeded")).Times(1)

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	reqBody := models.AnalysisRequest{URL: "https://example.com"}
// 	jsonData, _ := json.Marshal(reqBody)
// 	req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(jsonData))

// 	w := httptest.NewRecorder()
// 	handler.AnalyzeURL(w, req)

// 	assert.Equal(t, http.StatusGatewayTimeout, w.Code)

// 	var errorResp models.ErrorResponse
// 	err := json.NewDecoder(w.Body).Decode(&errorResp)
// 	require.NoError(t, err)
// 	assert.Equal(t, "Analysis timeout", errorResp.Error)
// }

// func TestAPIHandler_AnalyzeURL_EncodingError(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	// Create a result that will cause JSON encoding to fail
// 	resultWithCyclicRef := &models.AnalysisResult{
// 		URL: "https://example.com",
// 	}

// 	// Set up expectations
// 	mockLogger.EXPECT().Info("Processing analysis request", "url", "https://example.com").Times(1)
// 	mockLogger.EXPECT().Error("Failed to encode response", "error", gomock.Any()).Times(1)
// 	mockAnalyzerClient.EXPECT().Analyze(gomock.Any(), "https://example.com").Return(resultWithCyclicRef, nil).Times(1)

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	reqBody := models.AnalysisRequest{URL: "https://example.com"}
// 	jsonData, _ := json.Marshal(reqBody)
// 	req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(jsonData))

// 	// Use a response writer that will fail on write
// 	w := &failingResponseWriter{ResponseRecorder: httptest.NewRecorder()}
// 	handler.AnalyzeURL(w, req)

// 	assert.Equal(t, http.StatusOK, w.Code)
// }

// func TestAPIHandler_BatchAnalyze_Success(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	urls := []string{"https://example.com", "https://test.com"}

// 	result1 := &models.AnalysisResult{
// 		URL:   "https://example.com",
// 		Title: "Example",
// 	}
// 	result2 := &models.AnalysisResult{
// 		URL:   "https://test.com",
// 		Title: "Test",
// 	}

// 	// Set up expectations
// 	mockAnalyzerClient.EXPECT().Analyze(gomock.Any(), "https://example.com").Return(result1, nil).Times(1)
// 	mockAnalyzerClient.EXPECT().Analyze(gomock.Any(), "https://test.com").Return(result2, nil).Times(1)

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	reqBody := models.BatchAnalysisRequest{URLs: urls}
// 	jsonData, _ := json.Marshal(reqBody)
// 	req := httptest.NewRequest("POST", "/batch-analyze", bytes.NewReader(jsonData))

// 	w := httptest.NewRecorder()
// 	handler.BatchAnalyze(w, req)

// 	assert.Equal(t, http.StatusOK, w.Code)

// 	var response models.BatchAnalysisResult
// 	err := json.NewDecoder(w.Body).Decode(&response)
// 	require.NoError(t, err)
// 	assert.Len(t, response.Results, 2)
// 	assert.Len(t, response.Errors, 0)
// 	assert.Equal(t, "https://example.com", response.Results[0].URL)
// 	assert.Equal(t, "https://test.com", response.Results[1].URL)
// 	assert.Greater(t, response.TotalTime, time.Duration(0))
// }

// func TestAPIHandler_BatchAnalyze_WithErrors(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	urls := []string{"https://example.com", "https://invalid.com"}

// 	result1 := &models.AnalysisResult{
// 		URL:   "https://example.com",
// 		Title: "Example",
// 	}

// 	// Set up expectations
// 	mockAnalyzerClient.EXPECT().Analyze(gomock.Any(), "https://example.com").Return(result1, nil).Times(1)
// 	mockAnalyzerClient.EXPECT().Analyze(gomock.Any(), "https://invalid.com").Return(nil, errors.New("invalid URL")).Times(1)

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	reqBody := models.BatchAnalysisRequest{URLs: urls}
// 	jsonData, _ := json.Marshal(reqBody)
// 	req := httptest.NewRequest("POST", "/batch-analyze", bytes.NewReader(jsonData))

// 	w := httptest.NewRecorder()
// 	handler.BatchAnalyze(w, req)

// 	assert.Equal(t, http.StatusOK, w.Code)

// 	var response models.BatchAnalysisResult
// 	err := json.NewDecoder(w.Body).Decode(&response)
// 	require.NoError(t, err)
// 	assert.Len(t, response.Results, 1)
// 	assert.Len(t, response.Errors, 1)
// 	assert.Equal(t, "https://example.com", response.Results[0].URL)
// 	assert.Equal(t, "invalid URL", response.Errors[0].Error)
// 	assert.Contains(t, response.Errors[0].Details, "https://invalid.com")
// }

// func TestAPIHandler_BatchAnalyze_EmptyURLs(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	mockLogger.EXPECT().Error("Failed to encode error response", "error", gomock.Any()).AnyTimes()

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	reqBody := models.BatchAnalysisRequest{URLs: []string{}}
// 	jsonData, _ := json.Marshal(reqBody)
// 	req := httptest.NewRequest("POST", "/batch-analyze", bytes.NewReader(jsonData))

// 	w := httptest.NewRecorder()
// 	handler.BatchAnalyze(w, req)

// 	assert.Equal(t, http.StatusBadRequest, w.Code)

// 	var errorResp models.ErrorResponse
// 	err := json.NewDecoder(w.Body).Decode(&errorResp)
// 	require.NoError(t, err)
// 	assert.Equal(t, "At least one URL is required", errorResp.Error)
// }

// func TestAPIHandler_BatchAnalyze_TooManyURLs(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	mockLogger.EXPECT().Error("Failed to encode error response", "error", gomock.Any()).AnyTimes()

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	// Create request with more than 100 URLs
// 	urls := make([]string, 101)
// 	for i := 0; i < 101; i++ {
// 		urls[i] = "https://example.com"
// 	}

// 	reqBody := models.BatchAnalysisRequest{URLs: urls}
// 	jsonData, _ := json.Marshal(reqBody)
// 	req := httptest.NewRequest("POST", "/batch-analyze", bytes.NewReader(jsonData))

// 	w := httptest.NewRecorder()
// 	handler.BatchAnalyze(w, req)

// 	assert.Equal(t, http.StatusBadRequest, w.Code)

// 	var errorResp models.ErrorResponse
// 	err := json.NewDecoder(w.Body).Decode(&errorResp)
// 	require.NoError(t, err)
// 	assert.Equal(t, "Maximum 100 URLs allowed per batch", errorResp.Error)
// }

// func TestAPIHandler_BatchAnalyze_InvalidJSON(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	mockLogger.EXPECT().Error("Failed to parse batch request", "error", gomock.Any()).Times(1)
// 	mockLogger.EXPECT().Error("Failed to encode error response", "error", gomock.Any()).AnyTimes()

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	req := httptest.NewRequest("POST", "/batch-analyze", strings.NewReader("invalid json"))

// 	w := httptest.NewRecorder()
// 	handler.BatchAnalyze(w, req)

// 	assert.Equal(t, http.StatusBadRequest, w.Code)

// 	var errorResp models.ErrorResponse
// 	err := json.NewDecoder(w.Body).Decode(&errorResp)
// 	require.NoError(t, err)
// 	assert.Equal(t, "Invalid request format", errorResp.Error)
// }

// func TestAPIHandler_BatchAnalyze_EncodingError(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	result := &models.AnalysisResult{
// 		URL:   "https://example.com",
// 		Title: "Example",
// 	}

// 	mockAnalyzerClient.EXPECT().Analyze(gomock.Any(), "https://example.com").Return(result, nil).Times(1)
// 	mockLogger.EXPECT().Error("Failed to encode batch response", "error", gomock.Any()).Times(1)

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	reqBody := models.BatchAnalysisRequest{URLs: []string{"https://example.com"}}
// 	jsonData, _ := json.Marshal(reqBody)
// 	req := httptest.NewRequest("POST", "/batch-analyze", bytes.NewReader(jsonData))

// 	// Use a response writer that will fail on write
// 	w := &failingResponseWriter{ResponseRecorder: httptest.NewRecorder()}
// 	handler.BatchAnalyze(w, req)

// 	assert.Equal(t, http.StatusOK, w.Code)
// }

// func TestAPIHandler_sendError(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	w := httptest.NewRecorder()
// 	handler.sendError(w, "Test error", http.StatusBadRequest)

// 	assert.Equal(t, http.StatusBadRequest, w.Code)
// 	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

// 	var errorResp models.ErrorResponse
// 	err := json.NewDecoder(w.Body).Decode(&errorResp)
// 	require.NoError(t, err)
// 	assert.Equal(t, "Test error", errorResp.Error)
// 	assert.Equal(t, http.StatusBadRequest, errorResp.StatusCode)
// 	assert.NotZero(t, errorResp.Timestamp)
// }

// func TestAPIHandler_sendError_EncodingError(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	mockLogger.EXPECT().Error("Failed to encode error response", "error", gomock.Any()).Times(1)

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	// Use a response writer that will fail on write
// 	w := &failingResponseWriter{ResponseRecorder: httptest.NewRecorder()}
// 	handler.sendError(w, "Test error", http.StatusBadRequest)

// 	assert.Equal(t, http.StatusBadRequest, w.Code)
// }

// // Benchmark tests
// func BenchmarkAPIHandler_AnalyzeURL(b *testing.B) {
// 	ctrl := gomock.NewController(b)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	result := &models.AnalysisResult{
// 		URL:   "https://example.com",
// 		Title: "Example",
// 	}

// 	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
// 	mockAnalyzerClient.EXPECT().Analyze(gomock.Any(), gomock.Any()).Return(result, nil).AnyTimes()

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	reqBody := models.AnalysisRequest{URL: "https://example.com"}
// 	jsonData, _ := json.Marshal(reqBody)

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(jsonData))
// 		w := httptest.NewRecorder()
// 		handler.AnalyzeURL(w, req)
// 	}
// }

// func BenchmarkAPIHandler_BatchAnalyze(b *testing.B) {
// 	ctrl := gomock.NewController(b)
// 	defer ctrl.Finish()

// 	mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 	mockLogger := mocks.NewMockLogger(ctrl)
// 	mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 	result := &models.AnalysisResult{
// 		URL:   "https://example.com",
// 		Title: "Example",
// 	}

// 	mockAnalyzerClient.EXPECT().Analyze(gomock.Any(), gomock.Any()).Return(result, nil).AnyTimes()

// 	handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 	reqBody := models.BatchAnalysisRequest{URLs: []string{"https://example.com", "https://test.com"}}
// 	jsonData, _ := json.Marshal(reqBody)

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		req := httptest.NewRequest("POST", "/batch-analyze", bytes.NewReader(jsonData))
// 		w := httptest.NewRecorder()
// 		handler.BatchAnalyze(w, req)
// 	}
// }

// // Table-driven test for different error scenarios
// func TestAPIHandler_AnalyzeURL_ErrorScenarios(t *testing.T) {
// 	tests := []struct {
// 		name           string
// 		setupMocks     func(*mocks.MockAnalyzerClient, *mocks.MockLogger)
// 		requestBody    models.AnalysisRequest
// 		expectedStatus int
// 		expectedError  string
// 	}{
// 		{
// 			name: "analyzer service error",
// 			setupMocks: func(client *mocks.MockAnalyzerClient, logger *mocks.MockLogger) {
// 				logger.EXPECT().Info(gomock.Any(), gomock.Any()).Times(1)
// 				logger.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
// 				logger.EXPECT().Error("Failed to encode error response", "error", gomock.Any()).AnyTimes()
// 				client.EXPECT().Analyze(gomock.Any(), "https://example.com").Return(nil, errors.New("service error")).Times(1)
// 			},
// 			requestBody:    models.AnalysisRequest{URL: "https://example.com"},
// 			expectedStatus: http.StatusInternalServerError,
// 			expectedError:  "Analysis failed: service error",
// 		},
// 		{
// 			name: "context timeout",
// 			setupMocks: func(client *mocks.MockAnalyzerClient, logger *mocks.MockLogger) {
// 				logger.EXPECT().Info(gomock.Any(), gomock.Any()).Times(1)
// 				logger.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
// 				logger.EXPECT().Error("Failed to encode error response", "error", gomock.Any()).AnyTimes()
// 				client.EXPECT().Analyze(gomock.Any(), "https://example.com").Return(nil, errors.New("context deadline exceeded")).Times(1)
// 			},
// 			requestBody:    models.AnalysisRequest{URL: "https://example.com"},
// 			expectedStatus: http.StatusGatewayTimeout,
// 			expectedError:  "Analysis timeout",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			ctrl := gomock.NewController(t)
// 			defer ctrl.Finish()

// 			mockAnalyzerClient := mocks.NewMockAnalyzerClient(ctrl)
// 			mockLogger := mocks.NewMockLogger(ctrl)
// 			mockMetrics := mocks.NewMockMetricsCollector(ctrl)

// 			tt.setupMocks(mockAnalyzerClient, mockLogger)

// 			handler := NewAPIHandler(mockAnalyzerClient, mockLogger, mockMetrics)

// 			jsonData, _ := json.Marshal(tt.requestBody)
// 			req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(jsonData))
// 			w := httptest.NewRecorder()

// 			handler.AnalyzeURL(w, req)

// 			assert.Equal(t, tt.expectedStatus, w.Code)

// 			var errorResp models.ErrorResponse
// 			err := json.NewDecoder(w.Body).Decode(&errorResp)
// 			require.NoError(t, err)
// 			assert.Equal(t, tt.expectedError, errorResp.Error)
// 		})
// 	}
// }

// // Helper struct for testing encoding errors
// type failingResponseWriter struct {
// 	*httptest.ResponseRecorder
// 	written bool
// }

// func (f *failingResponseWriter) Write(data []byte) (int, error) {
// 	if f.written {
// 		return 0, errors.New("write failed")
// 	}
// 	f.written = true
// 	return f.ResponseRecorder.Write(data)
// }
