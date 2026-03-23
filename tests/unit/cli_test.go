package unit

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hironow/weaveback/cmd/weaveback/cli"
)

func TestNoArgs_ShowsUsage(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer

	// when
	code := cli.Run([]string{"weaveback"}, nil, &stdout, &stderr)

	// then
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Errorf("expected usage help on stderr, got: %s", stderr.String())
	}
}

func TestUnknownSubcommand_ExitCode1(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer

	// when
	code := cli.Run([]string{"weaveback", "unknown"}, nil, &stdout, &stderr)

	// then
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown subcommand") {
		t.Errorf("expected 'unknown subcommand' on stderr, got: %s", stderr.String())
	}
}

func TestFeedbackNoSubcommand_ShowsUsage(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer

	// when
	code := cli.Run([]string{"weaveback", "feedback"}, nil, &stdout, &stderr)

	// then
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Errorf("expected usage help on stderr, got: %s", stderr.String())
	}
}

func TestFeedbackCreate_MissingRequiredFlags(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer

	// when
	code := cli.Run([]string{"weaveback", "feedback", "create"}, nil, &stdout, &stderr)

	// then
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "error") {
		t.Errorf("expected error message on stderr, got: %s", errOut)
	}
}

func TestFeedbackQuery_MissingProjectID(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer

	// when
	code := cli.Run([]string{"weaveback", "feedback", "query"}, nil, &stdout, &stderr)

	// then
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "project-id") {
		t.Errorf("expected error about project-id on stderr, got: %s", errOut)
	}
}

func TestFeedbackPurge_MissingProjectID(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer

	// when
	code := cli.Run([]string{"weaveback", "feedback", "purge"}, nil, &stdout, &stderr)

	// then
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestFeedbackReplace_MissingRequiredFlags(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer

	// when
	code := cli.Run([]string{"weaveback", "feedback", "replace"}, nil, &stdout, &stderr)

	// then
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestFeedbackBatchCreate_MissingFile(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer

	// when
	code := cli.Run([]string{"weaveback", "feedback", "batch-create"}, nil, &stdout, &stderr)

	// then
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestErrorOutput_IsJSON(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer

	// when
	cli.Run([]string{"weaveback", "unknown"}, nil, &stdout, &stderr)

	// then
	errOut := stderr.String()
	lines := strings.Split(strings.TrimSpace(errOut), "\n")
	if len(lines) == 0 {
		t.Fatal("expected at least one line on stderr")
	}
	var errJSON struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &errJSON); err != nil {
		t.Errorf("expected JSON error on stderr, got parse error: %v for line: %s", err, lines[0])
	}
	if errJSON.Code != 1 {
		t.Errorf("expected code 1 in JSON error, got %d", errJSON.Code)
	}
}

func TestFeedbackBatchCreate_NonexistentFile(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer
	t.Setenv("WANDB_API_KEY", "test-key")

	// when
	code := cli.Run([]string{
		"weaveback", "feedback", "batch-create",
		"--file", "/nonexistent/path/data.jsonl",
	}, nil, &stdout, &stderr)

	// then
	if code != 1 {
		t.Errorf("expected exit code 1 for nonexistent file, got %d", code)
	}
}

func TestFeedbackCreate_InvalidPayloadJSON(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer

	// when
	code := cli.Run([]string{
		"weaveback", "feedback", "create",
		"--project-id", "test",
		"--feedback-type", "note",
		"--payload", "not-json",
	}, nil, &stdout, &stderr)

	// then
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid JSON, got %d", code)
	}
}

func TestFeedbackCreate_MissingWeaveRef_ExitCode1(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer
	t.Setenv("WANDB_API_KEY", "")

	// when — all other required flags present, but --weave-ref is missing
	code := cli.Run([]string{
		"weaveback", "feedback", "create",
		"--project-id", "test-project",
		"--feedback-type", "note",
		"--payload", `{"note":"hello"}`,
	}, nil, &stdout, &stderr)

	// then
	if code != 1 {
		t.Errorf("expected exit code 1 (missing --weave-ref), got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "weave-ref") {
		t.Errorf("expected error mentioning weave-ref, got: %s", errOut)
	}
}

func TestFeedbackCreate_WithWeaveRef_NoToken_ExitCode2(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer
	t.Setenv("WANDB_API_KEY", "")

	// when — all required flags including --weave-ref, but no token
	code := cli.Run([]string{
		"weaveback", "feedback", "create",
		"--project-id", "test-project",
		"--feedback-type", "note",
		"--payload", `{"note":"hello"}`,
		"--weave-ref", "weave:///entity/project/object/name:version",
	}, nil, &stdout, &stderr)

	// then
	if code != 2 {
		t.Errorf("expected exit code 2 (auth error), got %d", code)
	}
}

func TestFeedbackReplace_MissingWeaveRef_ExitCode1(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer
	t.Setenv("WANDB_API_KEY", "")

	// when — all other required flags present, but --weave-ref is missing
	code := cli.Run([]string{
		"weaveback", "feedback", "replace",
		"--feedback-id", "test-id",
		"--project-id", "test-project",
		"--feedback-type", "note",
		"--payload", `{"note":"hello"}`,
	}, nil, &stdout, &stderr)

	// then
	if code != 1 {
		t.Errorf("expected exit code 1 (missing --weave-ref), got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "weave-ref") {
		t.Errorf("expected error mentioning weave-ref, got: %s", errOut)
	}
}
