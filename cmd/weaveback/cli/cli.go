package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

// Exit codes per MY-397 specification.
const (
	ExitSuccess      = 0
	ExitUserError    = 1
	ExitAuthError    = 2
	ExitAPIError     = 3
	ExitPartialError = 4
)

// Run executes the CLI with the given arguments and I/O streams.
// It returns an exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		printUsage(stderr)
		return ExitUserError
	}

	switch args[1] {
	case "feedback":
		return runFeedback(args[2:], stdin, stdout, stderr)
	default:
		writeError(stderr, fmt.Sprintf("unknown subcommand: %s", args[1]), ExitUserError)
		return ExitUserError
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: weaveback <command> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  feedback    Manage Weave feedback entries")
}

// writeError writes a JSON error message to stderr.
func writeError(w io.Writer, msg string, code int) {
	e := struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	}{
		Error: msg,
		Code:  code,
	}
	b, _ := json.Marshal(e)
	fmt.Fprintln(w, string(b))
}
