// Command restclient executes HTTP requests defined in a .http / .rest file.
//
// Install: go install github.com/bmcszk/go-restclient/cmd/restclient@latest
// Usage:   restclient -f requests.http [-n name | -i index] [-e expected.hresp]
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"math"
	"os"
	"slices"
	"strings"

	"github.com/bmcszk/go-restclient"
)

func main() {
	file := flag.String("f", "", "path to the .http / .rest request file (required)")
	name := flag.String("n", "", "run only the request with this name")
	index := flag.Int("i", math.MinInt, "run only the request at this 0-based index")
	expected := flag.String("e", "", "path to .hresp file for response assertion")
	flag.Parse()
	os.Exit(run(*file, *name, *index, *expected))
}

func run(filePath, name string, index int, expected string) int {
	if filePath == "" {
		flushError("-f <file> is required")
		return 2
	}
	if name != "" && index != math.MinInt {
		flushError("-n and -i are mutually exclusive")
		return 2
	}

	client, err := restclient.NewClient()
	if err != nil {
		flushError(err.Error())
		return 1
	}

	parsedFile, err := client.ParseFile(filePath)
	if err != nil {
		flushError(err.Error())
		return 1
	}

	responses, execErr := executeRequests(client, parsedFile, filePath, name, index)
	if execErr != nil {
		flushError(execErr.Error())
		return 1
	}

	if assertErr := validateIfRequested(client, expected, responses); assertErr != nil {
		flushError(assertErr.Error())
		return 1
	}

	return emit(responses)
}

// validateIfRequested runs ValidateResponses when an expected file is provided.
func validateIfRequested(
	client *restclient.Client,
	expected string,
	responses []*restclient.Response,
) error {
	if expected == "" {
		return nil
	}
	return client.ValidateResponses(expected, responses...)
}

// executeRequests runs one or all requests and returns the responses.
func executeRequests(
	client *restclient.Client,
	parsedFile *restclient.ParsedFile,
	filePath, name string,
	index int,
) ([]*restclient.Response, error) {
	if name != "" || index != math.MinInt {
		return executeSingle(client, parsedFile, name, index)
	}
	return client.ExecuteFile(context.Background(), filePath)
}

// executeSingle runs one request from a parsed file by name or index.
func executeSingle(
	client *restclient.Client,
	parsedFile *restclient.ParsedFile,
	name string,
	index int,
) ([]*restclient.Response, error) {
	reqIndex, findErr := findRequestIndex(parsedFile.Requests, name, index)
	if findErr != nil {
		return nil, findErr
	}
	resp, err := client.ExecuteRequest(context.Background(), parsedFile, reqIndex)
	if err != nil {
		return nil, err
	}
	return []*restclient.Response{resp}, nil
}

// findRequestIndex resolves a request name or index to its position in the list.
func findRequestIndex(requests []*restclient.Request, name string, index int) (int, error) {
	if name != "" {
		idx, found := findByName(requests, name)
		if !found {
			return -1, fmt.Errorf("request name %q not found", name)
		}
		return idx, nil
	}
	if index < 0 || index >= len(requests) {
		return -1, fmt.Errorf(
			"request index %d out of range (file has %d requests)", index, len(requests))
	}
	return index, nil
}

// findByName returns the index of the first request with a matching name (case-insensitive).
func findByName(requests []*restclient.Request, name string) (int, bool) {
	for i, r := range requests {
		if strings.EqualFold(r.Name, name) {
			return i, true
		}
	}
	return -1, false
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

func flushOutput(s string) error {
	_, err := os.Stdout.WriteString(s)
	return err
}

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

