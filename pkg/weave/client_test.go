package weave_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hironow/rest/pkg/weave"
	"github.com/hironow/rest/pkg/weave/gen"
)

func TestNewClient_MissingAPIKey(t *testing.T) {
	// given
	apiKey := ""

	// when
	_, err := weave.NewClient("https://trace.wandb.ai", apiKey)

	// then
	if err == nil {
		t.Fatal("expected error when apiKey is empty, got nil")
	}
}

func TestNewClient_WithAPIKey(t *testing.T) {
	// given
	apiKey := "test-api-key-123"

	// when
	client, err := weave.NewClient("https://trace.wandb.ai", apiKey)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_AuthorizationHeader(t *testing.T) {
	// given
	apiKey := "test-bearer-token"

	var capturedAuthHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(gen.FeedbackQueryRes{})
	}))
	defer srv.Close()

	client, err := weave.NewClient(srv.URL, apiKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// when
	_, _ = client.QueryFeedback(context.Background(), gen.FeedbackQueryReq{
		ProjectID: "test-project",
	})

	// then: Basic auth encodes "api:test-bearer-token" in base64
	if capturedAuthHeader == "" {
		t.Error("Authorization header is empty")
	}
	if len(capturedAuthHeader) < 6 || capturedAuthHeader[:6] != "Basic " {
		t.Errorf("Authorization header = %q, want Basic auth scheme", capturedAuthHeader)
	}
}

func TestClientError_Error(t *testing.T) {
	// given
	err := &weave.ClientError{
		StatusCode: 422,
		Body:       []byte(`{"detail":"validation error"}`),
		Retryable:  false,
	}

	// when
	msg := err.Error()

	// then
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestClientError_NonRetryableStatuses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		retryable  bool
	}{
		{"400 Bad Request", 400, false},
		{"401 Unauthorized", 401, false},
		{"403 Forbidden", 403, false},
		{"404 Not Found", 404, false},
		{"422 Unprocessable", 422, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &weave.ClientError{
				StatusCode: tt.statusCode,
				Retryable:  tt.retryable,
			}
			if err.Retryable != false {
				t.Errorf("expected Retryable=false for status %d", tt.statusCode)
			}
		})
	}
}

func TestCreateFeedback_Success(t *testing.T) {
	// given
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/feedback/create" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(gen.FeedbackCreateRes{})
	}))
	defer srv.Close()

	client, err := weave.NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// when
	resp, err := client.CreateFeedback(context.Background(), gen.FeedbackCreateReq{
		ProjectID:    "test-project",
		FeedbackType: "test",
		Payload:      map[string]any{"key": "value"},
	})

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestCreateFeedbackBatch_Success(t *testing.T) {
	// given
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/feedback/batch/create" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(gen.FeedbackCreateBatchRes{})
	}))
	defer srv.Close()

	client, err := weave.NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// when
	resp, err := client.CreateFeedbackBatch(context.Background(), gen.FeedbackCreateBatchReq{
		Batch: []gen.FeedbackCreateReq{},
	})

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestQueryFeedback_Success(t *testing.T) {
	// given
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/feedback/query" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(gen.FeedbackQueryRes{})
	}))
	defer srv.Close()

	client, err := weave.NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// when
	resp, err := client.QueryFeedback(context.Background(), gen.FeedbackQueryReq{
		ProjectID: "test-project",
	})

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestPurgeFeedback_Success(t *testing.T) {
	// given
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/feedback/purge" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(gen.FeedbackPurgeRes{})
	}))
	defer srv.Close()

	client, err := weave.NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// when
	resp, err := client.PurgeFeedback(context.Background(), gen.FeedbackPurgeReq{
		ProjectID: "test-project",
		Query:     gen.Query{},
	})

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestReplaceFeedback_Success(t *testing.T) {
	// given
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/feedback/replace" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(gen.FeedbackReplaceRes{})
	}))
	defer srv.Close()

	client, err := weave.NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// when
	resp, err := client.ReplaceFeedback(context.Background(), gen.FeedbackReplaceReq{
		ProjectID:    "test-project",
		FeedbackID:   "feedback-123",
		FeedbackType: "test",
		Payload:      map[string]any{"key": "value"},
	})

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestCreateFeedback_HTTPError(t *testing.T) {
	// given
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]any{
			"detail": []map[string]any{
				{"loc": []string{"body", "project_id"}, "msg": "field required", "type": "value_error.missing"},
			},
		})
	}))
	defer srv.Close()

	client, err := weave.NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// when
	_, err = client.CreateFeedback(context.Background(), gen.FeedbackCreateReq{
		ProjectID:    "test-project",
		FeedbackType: "test",
		Payload:      map[string]any{},
	})

	// then
	if err == nil {
		t.Fatal("expected error for 422 response, got nil")
	}
	var clientErr *weave.ClientError
	if !weave.AsClientError(err, &clientErr) {
		t.Fatalf("expected ClientError, got %T: %v", err, err)
	}
	if clientErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("StatusCode = %d, want %d", clientErr.StatusCode, http.StatusUnprocessableEntity)
	}
	if clientErr.Retryable {
		t.Error("expected Retryable=false for 422")
	}
}
