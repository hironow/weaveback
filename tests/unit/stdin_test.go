package unit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/hironow/rest/cmd/weaveback/cli"
)

func TestProcessStdinLines_EmptyInput_ExitCode0(t *testing.T) {
	// given
	stdin := strings.NewReader("")
	var stdout, stderr bytes.Buffer
	processor := func(line string) (json.RawMessage, error) {
		t.Fatal("processor should not be called for empty input")
		return nil, nil
	}

	// when
	code := cli.ProcessStdinLines(stdin, processor, &stdout, &stderr)

	// then
	if code != 0 {
		t.Errorf("expected exit code 0 for empty stdin, got %d", code)
	}
	if stdout.String() != "" {
		t.Errorf("expected empty stdout, got: %s", stdout.String())
	}
}

func TestProcessStdinLines_ValidLines_ProcessedInOrder(t *testing.T) {
	// given
	input := `{"msg":"first"}
{"msg":"second"}
{"msg":"third"}
`
	stdin := strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	callOrder := []string{}
	processor := func(line string) (json.RawMessage, error) {
		callOrder = append(callOrder, line)
		return json.RawMessage(`{"ok":true,"input":` + line + `}`), nil
	}

	// when
	code := cli.ProcessStdinLines(stdin, processor, &stdout, &stderr)

	// then
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if len(callOrder) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(callOrder))
	}
	if callOrder[0] != `{"msg":"first"}` {
		t.Errorf("expected first call with first line, got: %s", callOrder[0])
	}

	// verify stdout has 3 lines of JSON output
	outLines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(outLines) != 3 {
		t.Errorf("expected 3 output lines, got %d: %s", len(outLines), stdout.String())
	}
}

func TestProcessStdinLines_InvalidJSON_ReportedOnStderr(t *testing.T) {
	// given
	input := `{"msg":"valid"}
not-json
{"msg":"also-valid"}
`
	stdin := strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	processor := func(line string) (json.RawMessage, error) {
		return json.RawMessage(`{"ok":true}`), nil
	}

	// when
	code := cli.ProcessStdinLines(stdin, processor, &stdout, &stderr)

	// then
	if code != cli.ExitPartialError {
		t.Errorf("expected exit code %d (partial), got %d", cli.ExitPartialError, code)
	}

	// stderr should contain error about line 2
	errOut := stderr.String()
	if !strings.Contains(errOut, `"line":2`) {
		t.Errorf("expected stderr to mention line 2, got: %s", errOut)
	}
	if !strings.Contains(errOut, "invalid JSON") {
		t.Errorf("expected stderr to mention 'invalid JSON', got: %s", errOut)
	}

	// stdout should have 2 lines (only valid ones)
	outLines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(outLines) != 2 {
		t.Errorf("expected 2 output lines on stdout, got %d: %s", len(outLines), stdout.String())
	}
}

func TestProcessStdinLines_AllInvalid_ExitCode1(t *testing.T) {
	// given
	input := `not-json-1
not-json-2
`
	stdin := strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	processor := func(line string) (json.RawMessage, error) {
		t.Fatal("processor should not be called for invalid JSON")
		return nil, nil
	}

	// when
	code := cli.ProcessStdinLines(stdin, processor, &stdout, &stderr)

	// then
	if code != cli.ExitUserError {
		t.Errorf("expected exit code %d (all invalid), got %d", cli.ExitUserError, code)
	}
	if stdout.String() != "" {
		t.Errorf("expected empty stdout when all lines invalid, got: %s", stdout.String())
	}
}

func TestProcessStdinLines_BlankLinesSkipped(t *testing.T) {
	// given
	input := `
{"msg":"valid"}

{"msg":"also-valid"}
`
	stdin := strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	callCount := 0
	processor := func(line string) (json.RawMessage, error) {
		callCount++
		return json.RawMessage(`{"ok":true}`), nil
	}

	// when
	code := cli.ProcessStdinLines(stdin, processor, &stdout, &stderr)

	// then
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if callCount != 2 {
		t.Errorf("expected 2 processor calls (blank lines skipped), got %d", callCount)
	}
}

func TestProcessStdinLines_APIError_ReportedOnStderr(t *testing.T) {
	// given
	input := `{"msg":"will-fail"}
{"msg":"will-succeed"}
`
	stdin := strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	callCount := 0
	processor := func(line string) (json.RawMessage, error) {
		callCount++
		if callCount == 1 {
			return nil, fmt.Errorf("API error: 500")
		}
		return json.RawMessage(`{"ok":true}`), nil
	}

	// when
	code := cli.ProcessStdinLines(stdin, processor, &stdout, &stderr)

	// then
	if code != cli.ExitPartialError {
		t.Errorf("expected exit code %d (partial), got %d", cli.ExitPartialError, code)
	}

	// stdout should have 1 line (only the successful one)
	outLines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(outLines) != 1 {
		t.Errorf("expected 1 output line on stdout, got %d: %s", len(outLines), stdout.String())
	}
}

func TestFeedbackCreate_StdinFlag_NoToken_ExitCode2(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer
	t.Setenv("WANDB_API_KEY", "")
	stdin := strings.NewReader(`{"project_id":"p","feedback_type":"note","payload":{"x":1}}`)

	// when
	code := cli.Run([]string{
		"weaveback", "feedback", "create", "--stdin",
	}, stdin, &stdout, &stderr)

	// then
	if code != cli.ExitAuthError {
		t.Errorf("expected exit code %d (auth error), got %d", cli.ExitAuthError, code)
	}
}

func TestFeedbackCreate_StdinAndPayload_MutuallyExclusive(t *testing.T) {
	// given
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	// when
	code := cli.Run([]string{
		"weaveback", "feedback", "create",
		"--stdin",
		"--project-id", "test",
		"--feedback-type", "note",
		"--payload", `{"x":1}`,
	}, stdin, &stdout, &stderr)

	// then
	if code != cli.ExitUserError {
		t.Errorf("expected exit code %d (user error for mutual exclusion), got %d", cli.ExitUserError, code)
	}
}
