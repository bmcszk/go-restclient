// Command restclient executes HTTP requests defined in a .http / .rest file.
//
// Install: go install github.com/bmcszk/go-restclient/cmd/restclient@latest
// Usage:   restclient -f requests.http
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/bmcszk/go-restclient"
)

func main() {
	file := flag.String("f", "", "path to the .http / .rest request file (required)")
	flag.Parse()
	os.Exit(run(*file))
}

func run(filePath string) int {
	if filePath == "" {
		flushError("-f <file> is required")
		return 2
	}
	client, err := restclient.NewClient()
	if err != nil {
		flushError(err.Error())
		return 1
	}
	responses, err := client.ExecuteFile(context.Background(), filePath)
	if err != nil {
		flushError(err.Error())
		return 1
	}
	return emit(responses)
}

// emit builds and writes the output for all responses. Returns the exit code:
// 1 if any response carried an error, 0 otherwise.
func emit(responses []*restclient.Response) int {
	var failed bool
	out := make([]string, 0, len(responses)+1)
	for _, resp := range responses {
		out = append(out, formatResponse(resp))
		if resp.Error != nil {
			failed = true
		}
	}
	if err := flushOutput(strings.Join(out, "")); err != nil {
		flushError(err.Error())
		return 1
	}
	if failed {
		return 1
	}
	return 0
}

// flushOutput writes accumulated output to stdout, returning any write error.
func flushOutput(s string) error {
	_, err := os.Stdout.WriteString(s)
	return err
}

// flushError writes a single error message to stderr. A broken stderr is the
// one failure mode this program cannot report, so the write result is
// discarded after the attempt.
func flushError(msg string) {
	_, _ = fmt.Fprintf(os.Stderr, "error: %s\n", msg)
}

func formatResponse(resp *restclient.Response) string {
	if resp.Error != nil {
		flushError(resp.Error.Error())
		return requestLine(resp) + "\n"
	}
	return requestLine(resp) + "\n" +
		fmt.Sprintf("  %s %s (%s)\n", resp.Proto, resp.Status, resp.Duration) +
		formatHeaders(resp.Headers) +
		formatBody(resp.Body) +
		"\n"
}

func requestLine(resp *restclient.Response) string {
	method := "-"
	url := "-"
	if resp.Request != nil {
		if resp.Request.Method != "" {
			method = resp.Request.Method
		}
		url = requestURL(resp.Request)
	}
	return method + " " + url
}

func requestURL(req *restclient.Request) string {
	if req.URL != nil {
		return req.URL.String()
	}
	if req.RawURLString != "" {
		return req.RawURLString
	}
	return "-"
}

func formatHeaders(h http.Header) string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		for _, v := range h[k] {
			lines = append(lines, fmt.Sprintf("  %s: %s\n", k, v))
		}
	}
	return strings.Join(lines, "")
}

func formatBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	s := "\n" + string(body)
	if body[len(body)-1] != '\n' {
		s += "\n"
	}
	return s
}
