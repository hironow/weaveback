package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// LineProcessor processes a single JSON line and returns the result as JSON.
// It is called for each valid JSON line read from stdin.
type LineProcessor func(line string) (json.RawMessage, error)

// ProcessStdinLines reads JSON Lines from r, validates each line as JSON,
// and calls processor for each valid line. Results are streamed to stdout
// immediately (no buffering). Invalid JSON lines and processor errors are
// reported to stderr. Blank lines are skipped.
//
// Exit codes:
//   - 0: all lines processed successfully (or empty input)
//   - 1: all non-blank lines were invalid
//   - 4: some lines failed (partial success)
func ProcessStdinLines(r io.Reader, processor LineProcessor, stdout, stderr io.Writer) int {
	scanner := bufio.NewScanner(r)
	lineNum := 0
	successCount := 0
	failCount := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if !json.Valid([]byte(line)) {
			failCount++
			writeStdinError(stderr, "invalid JSON", lineNum, line)
			continue
		}

		result, err := processor(line)
		if err != nil {
			failCount++
			writeStdinError(stderr, fmt.Sprintf("API error: %v", err), lineNum, line)
			continue
		}

		fmt.Fprintln(stdout, string(result))
		successCount++
	}

	if err := scanner.Err(); err != nil {
		writeError(stderr, fmt.Sprintf("reading stdin: %v", err), ExitAPIError)
		return ExitAPIError
	}

	if failCount > 0 && successCount == 0 {
		return ExitUserError
	}
	if failCount > 0 {
		return ExitPartialError
	}
	return ExitSuccess
}

// writeStdinError writes a JSON error to stderr with line number and raw content.
func writeStdinError(w io.Writer, msg string, lineNum int, raw string) {
	e := struct {
		Error string `json:"error"`
		Line  int    `json:"line"`
		Raw   string `json:"raw"`
	}{
		Error: msg,
		Line:  lineNum,
		Raw:   raw,
	}
	b, _ := json.Marshal(e)
	fmt.Fprintln(w, string(b))
}
