package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hironow/rest/pkg/weave"
	"github.com/hironow/rest/pkg/weave/gen"
)

const (
	defaultServerURL = "https://trace.wandb.ai"
	envWandbAPIKey   = "WANDB_API_KEY"
)

func runFeedback(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		printFeedbackUsage(stderr)
		return ExitUserError
	}

	switch args[0] {
	case "create":
		return runFeedbackCreate(args[1:], stdin, stdout, stderr)
	case "query":
		return runFeedbackQuery(args[1:], stdout, stderr)
	case "purge":
		return runFeedbackPurge(args[1:], stdout, stderr)
	case "replace":
		return runFeedbackReplace(args[1:], stdout, stderr)
	case "batch-create":
		return runFeedbackBatchCreate(args[1:], stdout, stderr)
	default:
		writeError(stderr, fmt.Sprintf("unknown feedback subcommand: %s", args[0]), ExitUserError)
		printFeedbackUsage(stderr)
		return ExitUserError
	}
}

func printFeedbackUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: weaveback feedback <subcommand> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  create        Create a single feedback entry")
	fmt.Fprintln(w, "  query         Query feedback entries")
	fmt.Fprintln(w, "  purge         Purge feedback entries")
	fmt.Fprintln(w, "  replace       Replace a feedback entry")
	fmt.Fprintln(w, "  batch-create  Create multiple feedback entries from JSON Lines file")
}

// resolveToken resolves the API token with priority: --token flag > WANDB_API_KEY env.
func resolveToken(tokenFlag string) (string, error) {
	if tokenFlag != "" {
		return tokenFlag, nil
	}
	if v := os.Getenv(envWandbAPIKey); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("authentication required: set --token flag or %s environment variable", envWandbAPIKey)
}

// newClient creates a Weave API client with the resolved token.
func newClient(tokenFlag string, stderr io.Writer) (*weave.Client, int) {
	token, err := resolveToken(tokenFlag)
	if err != nil {
		writeError(stderr, err.Error(), ExitAuthError)
		return nil, ExitAuthError
	}
	// Set env for the client constructor which reads WANDB_API_KEY
	os.Setenv(envWandbAPIKey, token)
	client, err := weave.NewClient(defaultServerURL)
	if err != nil {
		writeError(stderr, fmt.Sprintf("failed to create client: %v", err), ExitAPIError)
		return nil, ExitAPIError
	}
	return client, ExitSuccess
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func runFeedbackCreate(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("feedback create", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectID := fs.String("project-id", "", "Project ID (required)")
	weaveRef := fs.String("weave-ref", "", "Weave reference URI (required)")
	feedbackType := fs.String("feedback-type", "", "Feedback type (required)")
	payload := fs.String("payload", "", "Payload as JSON string (required)")
	callRef := fs.String("call-ref", "", "Call reference (optional)")
	token := fs.String("token", "", "API token (overrides WANDB_API_KEY)")
	useStdin := fs.Bool("stdin", false, "Read JSON Lines from stdin (mutually exclusive with --payload)")

	if err := fs.Parse(args); err != nil {
		return ExitUserError
	}

	if *useStdin {
		if *payload != "" {
			writeError(stderr, "--stdin and --payload are mutually exclusive", ExitUserError)
			return ExitUserError
		}
		return runFeedbackCreateStdin(*token, stdin, stdout, stderr)
	}

	if *projectID == "" || *weaveRef == "" || *feedbackType == "" || *payload == "" {
		writeError(stderr, "required flags: --project-id, --weave-ref, --feedback-type, --payload", ExitUserError)
		fs.Usage()
		return ExitUserError
	}

	var payloadMap map[string]any
	if err := json.Unmarshal([]byte(*payload), &payloadMap); err != nil {
		writeError(stderr, fmt.Sprintf("invalid --payload JSON: %v", err), ExitUserError)
		return ExitUserError
	}

	client, code := newClient(*token, stderr)
	if code != ExitSuccess {
		return code
	}

	req := gen.FeedbackCreateReq{
		ProjectID:    *projectID,
		WeaveRef:     *weaveRef,
		FeedbackType: *feedbackType,
		Payload:      payloadMap,
	}
	if *callRef != "" {
		req.CallRef = callRef
	}

	res, err := client.CreateFeedback(context.Background(), req)
	if err != nil {
		return handleAPIError(stderr, err)
	}

	writeJSON(stdout, res)
	return ExitSuccess
}

func runFeedbackCreateStdin(tokenFlag string, stdin io.Reader, stdout, stderr io.Writer) int {
	client, code := newClient(tokenFlag, stderr)
	if code != ExitSuccess {
		return code
	}

	processor := func(line string) (json.RawMessage, error) {
		var req gen.FeedbackCreateReq
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			return nil, fmt.Errorf("invalid request: %w", err)
		}
		res, err := client.CreateFeedback(context.Background(), req)
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(res)
		if err != nil {
			return nil, fmt.Errorf("marshal response: %w", err)
		}
		return b, nil
	}

	return ProcessStdinLines(stdin, processor, stdout, stderr)
}

func runFeedbackQuery(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("feedback query", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectID := fs.String("project-id", "", "Project ID (required)")
	limit := fs.Int("limit", 0, "Max results to return")
	offset := fs.Int("offset", 0, "Offset for pagination")
	token := fs.String("token", "", "API token (overrides WANDB_API_KEY)")

	if err := fs.Parse(args); err != nil {
		return ExitUserError
	}

	if *projectID == "" {
		writeError(stderr, "required flag: --project-id", ExitUserError)
		fs.Usage()
		return ExitUserError
	}

	client, code := newClient(*token, stderr)
	if code != ExitSuccess {
		return code
	}

	req := gen.FeedbackQueryReq{
		ProjectID: *projectID,
	}
	if *limit > 0 {
		req.Limit = limit
	}
	if *offset > 0 {
		req.Offset = offset
	}

	res, err := client.QueryFeedback(context.Background(), req)
	if err != nil {
		return handleAPIError(stderr, err)
	}

	writeJSON(stdout, res)
	return ExitSuccess
}

func runFeedbackPurge(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("feedback purge", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectID := fs.String("project-id", "", "Project ID (required)")
	query := fs.String("query", "", "Query expression as JSON (required)")
	token := fs.String("token", "", "API token (overrides WANDB_API_KEY)")

	if err := fs.Parse(args); err != nil {
		return ExitUserError
	}

	if *projectID == "" || *query == "" {
		writeError(stderr, "required flags: --project-id, --query", ExitUserError)
		fs.Usage()
		return ExitUserError
	}

	var queryObj gen.Query
	if err := json.Unmarshal([]byte(*query), &queryObj); err != nil {
		writeError(stderr, fmt.Sprintf("invalid --query JSON: %v", err), ExitUserError)
		return ExitUserError
	}

	client, code := newClient(*token, stderr)
	if code != ExitSuccess {
		return code
	}

	req := gen.FeedbackPurgeReq{
		ProjectID: *projectID,
		Query:     queryObj,
	}

	res, err := client.PurgeFeedback(context.Background(), req)
	if err != nil {
		return handleAPIError(stderr, err)
	}

	writeJSON(stdout, res)
	return ExitSuccess
}

func runFeedbackReplace(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("feedback replace", flag.ContinueOnError)
	fs.SetOutput(stderr)

	feedbackID := fs.String("feedback-id", "", "Feedback ID to replace (required)")
	projectID := fs.String("project-id", "", "Project ID (required)")
	weaveRef := fs.String("weave-ref", "", "Weave reference URI (required)")
	feedbackType := fs.String("feedback-type", "", "Feedback type (required)")
	payload := fs.String("payload", "", "Payload as JSON string (required)")
	token := fs.String("token", "", "API token (overrides WANDB_API_KEY)")

	if err := fs.Parse(args); err != nil {
		return ExitUserError
	}

	if *feedbackID == "" || *projectID == "" || *weaveRef == "" || *feedbackType == "" || *payload == "" {
		writeError(stderr, "required flags: --feedback-id, --project-id, --weave-ref, --feedback-type, --payload", ExitUserError)
		fs.Usage()
		return ExitUserError
	}

	var payloadMap map[string]any
	if err := json.Unmarshal([]byte(*payload), &payloadMap); err != nil {
		writeError(stderr, fmt.Sprintf("invalid --payload JSON: %v", err), ExitUserError)
		return ExitUserError
	}

	client, code := newClient(*token, stderr)
	if code != ExitSuccess {
		return code
	}

	req := gen.FeedbackReplaceReq{
		FeedbackID:   *feedbackID,
		ProjectID:    *projectID,
		WeaveRef:     *weaveRef,
		FeedbackType: *feedbackType,
		Payload:      payloadMap,
	}

	res, err := client.ReplaceFeedback(context.Background(), req)
	if err != nil {
		return handleAPIError(stderr, err)
	}

	writeJSON(stdout, res)
	return ExitSuccess
}

func runFeedbackBatchCreate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("feedback batch-create", flag.ContinueOnError)
	fs.SetOutput(stderr)

	filePath := fs.String("file", "", "Path to JSON Lines file (required)")
	token := fs.String("token", "", "API token (overrides WANDB_API_KEY)")

	if err := fs.Parse(args); err != nil {
		return ExitUserError
	}

	if *filePath == "" {
		writeError(stderr, "required flag: --file", ExitUserError)
		fs.Usage()
		return ExitUserError
	}

	client, code := newClient(*token, stderr)
	if code != ExitSuccess {
		return code
	}

	f, err := os.Open(*filePath)
	if err != nil {
		writeError(stderr, fmt.Sprintf("failed to open file: %v", err), ExitUserError)
		return ExitUserError
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	failCount := 0
	var results []json.RawMessage

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req gen.FeedbackCreateReq
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			failCount++
			writeError(stderr, fmt.Sprintf("line %d: invalid JSON: %v", lineNum, err), ExitPartialError)
			continue
		}

		res, err := client.CreateFeedback(context.Background(), req)
		if err != nil {
			failCount++
			writeError(stderr, fmt.Sprintf("line %d: API error: %v", lineNum, err), ExitPartialError)
			continue
		}

		b, _ := json.Marshal(res)
		results = append(results, b)
	}

	if err := scanner.Err(); err != nil {
		writeError(stderr, fmt.Sprintf("reading file: %v", err), ExitAPIError)
		return ExitAPIError
	}

	// Write successful results to stdout
	for _, r := range results {
		fmt.Fprintln(stdout, string(r))
	}

	if failCount > 0 {
		writeError(stderr, fmt.Sprintf("%d of %d lines failed", failCount, lineNum), ExitPartialError)
		return ExitPartialError
	}

	return ExitSuccess
}

// handleAPIError maps weave.ClientError to appropriate exit codes.
func handleAPIError(stderr io.Writer, err error) int {
	var ce *weave.ClientError
	if weave.AsClientError(err, &ce) {
		writeError(stderr, fmt.Sprintf("API error (HTTP %d): %s", ce.StatusCode, string(ce.Body)), ExitAPIError)
		return ExitAPIError
	}
	writeError(stderr, fmt.Sprintf("API error: %v", err), ExitAPIError)
	return ExitAPIError
}
