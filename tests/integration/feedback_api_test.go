package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hironow/weaveback/pkg/weave"
	"github.com/hironow/weaveback/pkg/weave/gen"
)

const (
	defaultServerURL = "https://trace.wandb.ai"
)

// testEnv holds the environment configuration for integration tests.
type testEnv struct {
	apiKey    string
	projectID string
	serverURL string
}

// loadTestEnv reads required environment variables and skips the test if not configured.
func loadTestEnv(t *testing.T) testEnv {
	t.Helper()

	apiKey := os.Getenv("WANDB_API_KEY")
	if apiKey == "" {
		t.Skip("WANDB_API_KEY not set; skipping integration test")
	}

	projectID := os.Getenv("WEAVE_TEST_PROJECT")
	if projectID == "" {
		t.Skip("WEAVE_TEST_PROJECT not set; skipping integration test (format: entity/project)")
	}

	serverURL := os.Getenv("WEAVE_SERVER_URL")
	if serverURL == "" {
		serverURL = defaultServerURL
	}

	return testEnv{
		apiKey:    apiKey,
		projectID: projectID,
		serverURL: serverURL,
	}
}

// newTestClient creates a Weave client for integration testing.
func newTestClient(t *testing.T, env testEnv) *weave.Client {
	t.Helper()

	client, err := weave.NewClient(env.serverURL, env.apiKey)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

// testWeaveRef generates a synthetic weave_ref for testing.
// Format: weave:///<entity>/<project>/object/<name>:<version>
func testWeaveRef(projectID string, uniqueID string) string {
	return fmt.Sprintf("weave:///%s/object/integration-test:%s", projectID, uniqueID)
}

// buildEqQuery constructs a Query with $eq expression using raw JSON.
func buildEqQuery(field string, value string) gen.Query {
	exprJSON := fmt.Sprintf(`{"$eq":[{"$getField":"%s"},{"$literal":"%s"}]}`, field, value)
	var q gen.Query
	raw := json.RawMessage(exprJSON)
	if err := q.Expr.UnmarshalJSON(raw); err != nil {
		panic(fmt.Sprintf("failed to build query: %v", err))
	}
	return q
}

func TestFeedbackLifecycle(t *testing.T) {
	env := loadTestEnv(t)
	client := newTestClient(t, env)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	feedbackType := "integration-test"
	uniqueID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	weaveRef := testWeaveRef(env.projectID, uniqueID)

	// --- Step 1: Create ---
	var createdID string
	t.Run("create", func(t *testing.T) {
		// given
		req := gen.FeedbackCreateReq{
			ProjectID:    env.projectID,
			FeedbackType: feedbackType,
			WeaveRef:     weaveRef,
			Payload: map[string]any{
				"test_id": uniqueID,
				"score":   0.85,
			},
		}

		// when
		resp, err := client.CreateFeedback(ctx, req)

		// then
		if err != nil {
			t.Fatalf("CreateFeedback failed: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if resp.ID == "" {
			t.Fatal("expected non-empty feedback ID")
		}
		createdID = resp.ID
		t.Logf("created feedback ID: %s", resp.ID)
	})

	// --- Step 2: Query ---
	var feedbackID string
	t.Run("query", func(t *testing.T) {
		if createdID == "" {
			t.Skip("skipping query: create step failed")
		}

		// given
		req := gen.FeedbackQueryReq{
			ProjectID: env.projectID,
		}

		// when
		resp, err := client.QueryFeedback(ctx, req)

		// then
		if err != nil {
			t.Fatalf("QueryFeedback failed: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if len(resp.Result) == 0 {
			t.Fatal("expected at least one feedback result")
		}

		// find our test feedback by ID
		for _, row := range resp.Result {
			if id, ok := row["id"].(string); ok && id == createdID {
				feedbackID = id
				break
			}
		}
		if feedbackID == "" {
			t.Fatalf("created feedback %s not found in query results", createdID)
		}
		t.Logf("found feedback ID: %s", feedbackID)
	})

	// --- Step 3: Replace ---
	t.Run("replace", func(t *testing.T) {
		if feedbackID == "" {
			t.Skip("skipping replace: no feedback ID from query step")
		}

		// given
		req := gen.FeedbackReplaceReq{
			ProjectID:    env.projectID,
			FeedbackID:   feedbackID,
			FeedbackType: feedbackType,
			WeaveRef:     weaveRef,
			Payload: map[string]any{
				"test_id":  uniqueID,
				"score":    0.95,
				"replaced": true,
			},
		}

		// when
		resp, err := client.ReplaceFeedback(ctx, req)

		// then
		if err != nil {
			t.Fatalf("ReplaceFeedback failed: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if resp.ID == "" {
			t.Fatal("expected non-empty feedback ID in replace response")
		}
		t.Logf("replaced feedback ID: %s", resp.ID)
	})

	// --- Step 4: Purge (cleanup) ---
	t.Run("purge", func(t *testing.T) {
		if feedbackID == "" {
			t.Skip("skipping purge: no feedback ID from query step")
		}

		// given
		query := buildEqQuery("id", feedbackID)
		req := gen.FeedbackPurgeReq{
			ProjectID: env.projectID,
			Query:     query,
		}

		// when
		resp, err := client.PurgeFeedback(ctx, req)

		// then
		if err != nil {
			t.Fatalf("PurgeFeedback failed: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response from purge")
		}
		t.Log("purge succeeded")
	})
}

func TestFeedbackQuery_NonExistentObject(t *testing.T) {
	env := loadTestEnv(t)
	client := newTestClient(t, env)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// given
	query := buildEqQuery("id", "non-existent-feedback-id-99999")
	req := gen.FeedbackQueryReq{
		ProjectID: env.projectID,
		Query:     &query,
	}

	// when
	resp, err := client.QueryFeedback(ctx, req)

	// then: either an error or empty results, depending on API behavior
	if err != nil {
		var clientErr *weave.ClientError
		if weave.AsClientError(err, &clientErr) {
			t.Logf("got expected error for non-existent object: status=%d", clientErr.StatusCode)
			return
		}
		t.Fatalf("unexpected error type: %v", err)
	}
	if resp != nil && len(resp.Result) == 0 {
		t.Log("query returned empty results for non-existent object (expected)")
		return
	}
	t.Logf("query returned %d results", len(resp.Result))
}

func TestFeedbackCreate_InvalidPayload(t *testing.T) {
	env := loadTestEnv(t)
	client := newTestClient(t, env)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// given: invalid project_id format (missing entity prefix)
	req := gen.FeedbackCreateReq{
		ProjectID:    "nonexistent-project-99999",
		FeedbackType: "test",
		WeaveRef:     "weave:///invalid/ref/object/test:v1",
		Payload:      map[string]any{"key": "value"},
	}

	// when
	_, err := client.CreateFeedback(ctx, req)

	// then
	if err == nil {
		t.Fatal("expected error for invalid payload, got nil")
	}
	var clientErr *weave.ClientError
	if weave.AsClientError(err, &clientErr) {
		t.Logf("got expected client error: status=%d", clientErr.StatusCode)
		if clientErr.StatusCode < 400 || clientErr.StatusCode >= 500 {
			t.Errorf("expected 4xx status code, got %d", clientErr.StatusCode)
		}
	} else {
		t.Logf("got non-ClientError (may indicate network or marshaling issue): %v", err)
	}
}

func TestFeedbackCreate_AuthenticationSkip(t *testing.T) {
	// given: empty apiKey

	// when
	_, err := weave.NewClient(defaultServerURL, "")

	// then
	if err == nil {
		t.Fatal("expected error when apiKey is empty")
	}
	t.Logf("graceful skip: %v", err)
}
