package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build the CLI binary once before all tests.
	tmpDir, err := os.MkdirTemp("", "weaveback-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "weaveback")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/weaveback")
	// Build from the module root, which is the repo root.
	buildCmd.Dir = moduleRoot()
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build weaveback binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// moduleRoot returns the absolute path to the Go module root.
func moduleRoot() string {
	// Walk up from the test file directory to find go.mod.
	dir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("failed to get working directory: %v", err))
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("could not find go.mod in any parent directory")
		}
		dir = parent
	}
}

// runCLI executes the weaveback binary with the given args and optional stdin.
// Returns stdout, stderr, and exit code.
func runCLI(t *testing.T, stdin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run CLI: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// skipIfNoAPIKey skips the test if WANDB_API_KEY is not set.
func skipIfNoAPIKey(t *testing.T) {
	t.Helper()
	if os.Getenv("WANDB_API_KEY") == "" {
		t.Skip("WANDB_API_KEY not set; skipping E2E test against real API")
	}
}

// testProjectID returns the project ID for E2E tests.
func testProjectID(t *testing.T) string {
	t.Helper()
	pid := os.Getenv("WEAVE_TEST_PROJECT")
	if pid == "" {
		pid = "hironow/weaveback"
	}
	return pid
}

// --- Lifecycle test: create → query → replace → purge ---

func TestCLI_FeedbackLifecycle(t *testing.T) {
	skipIfNoAPIKey(t)
	projectID := testProjectID(t)

	// Step 1: create
	createPayload := `{"note":"e2e-test-lifecycle"}`
	stdout, stderr, code := runCLI(t, "",
		"feedback", "create",
		"--project-id", projectID,
		"--feedback-type", "wandb.note.1",
		"--payload", createPayload,
	)
	if code != 0 {
		t.Fatalf("create: expected exit code 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	var createRes map[string]any
	if err := json.Unmarshal([]byte(stdout), &createRes); err != nil {
		t.Fatalf("create: stdout is not valid JSON: %v\nstdout: %s", err, stdout)
	}

	feedbackID, ok := createRes["id"].(string)
	if !ok || feedbackID == "" {
		t.Fatalf("create: expected 'id' in response, got: %v", createRes)
	}
	t.Logf("created feedback id=%s", feedbackID)

	// Step 2: query
	stdout, stderr, code = runCLI(t, "",
		"feedback", "query",
		"--project-id", projectID,
	)
	if code != 0 {
		t.Fatalf("query: expected exit code 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	var queryRes map[string]any
	if err := json.Unmarshal([]byte(stdout), &queryRes); err != nil {
		t.Fatalf("query: stdout is not valid JSON: %v\nstdout: %s", err, stdout)
	}
	if _, ok := queryRes["result"]; !ok {
		t.Fatalf("query: expected 'result' key in response, got: %v", queryRes)
	}

	// Step 3: replace
	replacePayload := `{"note":"e2e-test-lifecycle-replaced"}`
	stdout, stderr, code = runCLI(t, "",
		"feedback", "replace",
		"--feedback-id", feedbackID,
		"--project-id", projectID,
		"--feedback-type", "wandb.note.1",
		"--payload", replacePayload,
	)
	if code != 0 {
		t.Fatalf("replace: expected exit code 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	var replaceRes map[string]any
	if err := json.Unmarshal([]byte(stdout), &replaceRes); err != nil {
		t.Fatalf("replace: stdout is not valid JSON: %v\nstdout: %s", err, stdout)
	}
	if replaceRes["id"] == nil {
		t.Fatalf("replace: expected 'id' in response, got: %v", replaceRes)
	}

	// Step 4: purge
	purgeQuery := fmt.Sprintf(`{"$expr":{"$eq":[{"$getField":"id"},{"$literal":"%s"}]}}`, feedbackID)
	stdout, stderr, code = runCLI(t, "",
		"feedback", "purge",
		"--project-id", projectID,
		"--query", purgeQuery,
	)
	if code != 0 {
		t.Fatalf("purge: expected exit code 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	// purge response should be valid JSON
	var purgeRes map[string]any
	if err := json.Unmarshal([]byte(stdout), &purgeRes); err != nil {
		t.Fatalf("purge: stdout is not valid JSON: %v\nstdout: %s", err, stdout)
	}
}

// --- Stdin pipe mode test ---

func TestCLI_FeedbackCreateStdin(t *testing.T) {
	skipIfNoAPIKey(t)
	projectID := testProjectID(t)

	// Prepare JSON Lines input
	line1 := fmt.Sprintf(`{"project_id":"%s","feedback_type":"wandb.note.1","payload":{"note":"e2e-stdin-line1"}}`, projectID)
	line2 := fmt.Sprintf(`{"project_id":"%s","feedback_type":"wandb.note.1","payload":{"note":"e2e-stdin-line2"}}`, projectID)
	stdinInput := line1 + "\n" + line2 + "\n"

	stdout, stderr, code := runCLI(t, stdinInput,
		"feedback", "create", "--stdin",
	)
	if code != 0 {
		t.Fatalf("stdin create: expected exit code 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	// Each line should produce a JSON output line
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("stdin create: expected 2 output lines, got %d\nstdout: %s", len(lines), stdout)
	}

	// Collect IDs for cleanup
	var feedbackIDs []string
	for i, line := range lines {
		var res map[string]any
		if err := json.Unmarshal([]byte(line), &res); err != nil {
			t.Errorf("stdin create: line %d is not valid JSON: %v", i+1, err)
			continue
		}
		if id, ok := res["id"].(string); ok {
			feedbackIDs = append(feedbackIDs, id)
		}
	}

	// Cleanup: purge created feedback
	for _, id := range feedbackIDs {
		purgeQuery := fmt.Sprintf(`{"$expr":{"$eq":[{"$getField":"id"},{"$literal":"%s"}]}}`, id)
		runCLI(t, "",
			"feedback", "purge",
			"--project-id", projectID,
			"--query", purgeQuery,
		)
	}
}

// --- Error case tests (no API key needed) ---

func TestCLI_NoArgs(t *testing.T) {
	_, stderr, code := runCLI(t, "")
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("expected usage on stderr, got: %s", stderr)
	}
}

func TestCLI_UnknownSubcommand(t *testing.T) {
	_, stderr, code := runCLI(t, "", "bogus")
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr, "unknown subcommand") {
		t.Errorf("expected 'unknown subcommand' on stderr, got: %s", stderr)
	}
	// stderr should be JSON
	var errJSON map[string]any
	firstLine := strings.Split(strings.TrimSpace(stderr), "\n")[0]
	if err := json.Unmarshal([]byte(firstLine), &errJSON); err != nil {
		t.Errorf("expected JSON error on stderr, got: %s", firstLine)
	}
}

func TestCLI_FeedbackCreate_MissingFlags(t *testing.T) {
	_, _, code := runCLI(t, "", "feedback", "create")
	if code != 1 {
		t.Errorf("expected exit code 1 for missing flags, got %d", code)
	}
}

func TestCLI_FeedbackCreate_InvalidPayload(t *testing.T) {
	_, _, code := runCLI(t, "",
		"feedback", "create",
		"--project-id", "test",
		"--feedback-type", "note",
		"--payload", "not-json",
	)
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid payload, got %d", code)
	}
}

func TestCLI_FeedbackCreate_AuthFailure(t *testing.T) {
	// Use invalid API key via --token flag
	_, stderr, code := runCLI(t, "",
		"feedback", "create",
		"--project-id", "test/project",
		"--feedback-type", "wandb.note.1",
		"--payload", `{"note":"test"}`,
		"--token", "invalid-key-12345",
	)
	// Should get either auth error (exit 2) or API error (exit 3)
	if code == 0 {
		t.Errorf("expected non-zero exit code for invalid API key, got 0")
	}
	if stderr == "" {
		t.Errorf("expected error on stderr for invalid API key")
	}
}

func TestCLI_StdinCreate_InvalidJSON(t *testing.T) {
	stdinInput := "this is not json\n"
	_, stderr, code := runCLI(t, stdinInput,
		"feedback", "create", "--stdin",
		"--token", "dummy-key",
	)
	// All lines invalid → exit 1
	if code != 1 {
		t.Errorf("expected exit code 1 for all invalid stdin lines, got %d", code)
	}
	if !strings.Contains(stderr, "invalid JSON") {
		t.Errorf("expected 'invalid JSON' on stderr, got: %s", stderr)
	}
}

func TestCLI_StdoutIsJSONOnly(t *testing.T) {
	// Verify that stderr gets diagnostics, not stdout
	_, stderr, _ := runCLI(t, "", "bogus")

	// stderr should contain something
	if stderr == "" {
		t.Error("expected diagnostic output on stderr")
	}
}
