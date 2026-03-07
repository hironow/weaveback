package weave_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hironow/rest/pkg/weave"
	"github.com/hironow/rest/pkg/weave/gen"
)

func TestAddReaction_EmptyEmoji(t *testing.T) {
	// given
	t.Setenv("WANDB_API_KEY", "test-key")
	client, err := weave.NewClient("https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// when
	err = client.AddReaction(context.Background(), "project-1", "call-1", "")

	// then
	if err == nil {
		t.Fatal("expected error for empty emoji, got nil")
	}
	if !strings.Contains(err.Error(), "emoji") {
		t.Errorf("error should mention emoji, got: %v", err)
	}
}

func TestAddReaction_TooLongEmoji(t *testing.T) {
	// given
	t.Setenv("WANDB_API_KEY", "test-key")
	client, err := weave.NewClient("https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	longEmoji := strings.Repeat("a", 65)

	// when
	err = client.AddReaction(context.Background(), "project-1", "call-1", longEmoji)

	// then
	if err == nil {
		t.Fatal("expected error for emoji longer than 64 chars, got nil")
	}
}

func TestAddReaction_InvalidCharsEmoji(t *testing.T) {
	tests := []struct {
		name  string
		emoji string
	}{
		{"space", "thumbs up"},
		{"unicode", "日本語"},
		{"special chars", "heart!@#"},
		{"dot", "thumbs.up"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			t.Setenv("WANDB_API_KEY", "test-key")
			client, err := weave.NewClient("https://example.com")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// when
			err = client.AddReaction(context.Background(), "project-1", "call-1", tt.emoji)

			// then
			if err == nil {
				t.Fatalf("expected error for invalid emoji %q, got nil", tt.emoji)
			}
		})
	}
}

func TestAddReaction_ValidEmoji(t *testing.T) {
	tests := []struct {
		name  string
		emoji string
	}{
		{"simple", "thumbsup"},
		{"with hyphen", "thumbs-up"},
		{"with underscore", "thumbs_up"},
		{"with digits", "100"},
		{"max length", strings.Repeat("a", 64)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			t.Setenv("WANDB_API_KEY", "test-key")

			var capturedBody gen.FeedbackCreateReq
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewDecoder(r.Body).Decode(&capturedBody)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(gen.FeedbackCreateRes{})
			}))
			defer srv.Close()

			client, err := weave.NewClient(srv.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// when
			err = client.AddReaction(context.Background(), "project-1", "call-1", tt.emoji)

			// then
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if capturedBody.ProjectID != "project-1" {
				t.Errorf("ProjectID = %q, want %q", capturedBody.ProjectID, "project-1")
			}
			if capturedBody.WeaveRef != "call-1" {
				t.Errorf("WeaveRef = %q, want %q", capturedBody.WeaveRef, "call-1")
			}
			if capturedBody.FeedbackType != "emoji" {
				t.Errorf("FeedbackType = %q, want %q", capturedBody.FeedbackType, "emoji")
			}
			emojiVal, ok := capturedBody.Payload["emoji"]
			if !ok {
				t.Fatal("expected payload to contain 'emoji' key")
			}
			if emojiVal != tt.emoji {
				t.Errorf("payload emoji = %q, want %q", emojiVal, tt.emoji)
			}
		})
	}
}

func TestAddNote_EmptyNote(t *testing.T) {
	// given
	t.Setenv("WANDB_API_KEY", "test-key")
	client, err := weave.NewClient("https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// when
	err = client.AddNote(context.Background(), "project-1", "call-1", "")

	// then
	if err == nil {
		t.Fatal("expected error for empty note, got nil")
	}
	if !strings.Contains(err.Error(), "note") {
		t.Errorf("error should mention note, got: %v", err)
	}
}

func TestAddNote_ValidNote(t *testing.T) {
	// given
	t.Setenv("WANDB_API_KEY", "test-key")

	var capturedBody gen.FeedbackCreateReq
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(gen.FeedbackCreateRes{})
	}))
	defer srv.Close()

	client, err := weave.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// when
	err = client.AddNote(context.Background(), "project-1", "call-1", "This is a test note")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedBody.ProjectID != "project-1" {
		t.Errorf("ProjectID = %q, want %q", capturedBody.ProjectID, "project-1")
	}
	if capturedBody.WeaveRef != "call-1" {
		t.Errorf("WeaveRef = %q, want %q", capturedBody.WeaveRef, "call-1")
	}
	if capturedBody.FeedbackType != "comment" {
		t.Errorf("FeedbackType = %q, want %q", capturedBody.FeedbackType, "comment")
	}
	noteVal, ok := capturedBody.Payload["note"]
	if !ok {
		t.Fatal("expected payload to contain 'note' key")
	}
	if noteVal != "This is a test note" {
		t.Errorf("payload note = %q, want %q", noteVal, "This is a test note")
	}
}

func TestAddNote_PassesThroughAPIError(t *testing.T) {
	// given
	t.Setenv("WANDB_API_KEY", "test-key")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"detail":[{"loc":["body","project_id"],"msg":"field required","type":"value_error.missing"}]}`))
	}))
	defer srv.Close()

	client, err := weave.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// when
	err = client.AddNote(context.Background(), "project-1", "call-1", "test note")

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
}

func TestAddReaction_PassesThroughAPIError(t *testing.T) {
	// given
	t.Setenv("WANDB_API_KEY", "test-key")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"detail":"unknown emoji"}`))
	}))
	defer srv.Close()

	client, err := weave.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// when
	err = client.AddReaction(context.Background(), "project-1", "call-1", "thumbsup")

	// then
	if err == nil {
		t.Fatal("expected error for 400 response, got nil")
	}
	var clientErr *weave.ClientError
	if !weave.AsClientError(err, &clientErr) {
		t.Fatalf("expected ClientError, got %T: %v", err, err)
	}
	if clientErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", clientErr.StatusCode, http.StatusBadRequest)
	}
}
