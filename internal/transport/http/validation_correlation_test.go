package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidationFailureKeepsRequestCorrelation(t *testing.T) {
	router, _ := testRouter(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/batches", strings.NewReader(`{"batch_id":"","samples":[]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", "request-correlation-001")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("X-Request-ID") != "request-correlation-001" {
		t.Fatalf("response header request id=%q", recorder.Header().Get("X-Request-ID"))
	}
	var response ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.RequestID != "request-correlation-001" {
		t.Fatalf("validation response lost request correlation: %+v", response)
	}
}
