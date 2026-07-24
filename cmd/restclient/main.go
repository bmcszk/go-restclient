// Command restclient executes HTTP requests defined in a .http / .rest file.
//
// Install: go install github.com/bmcszk/go-restclient/cmd/restclient@latest
// Usage:   restclient -f requests.http [-n name | -i index] [-e expected.hresp]
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/bmcszk/go-restclient"
)

type runConfig struct {
	filePath      string
	name          string
	index         int
	expected      string
	expectedName  string
	expectedIndex int
	failOnError   bool
	output        string
	after         string
	defines       []string
}

type defineArray []string

func (d *defineArray) String() string {
	return strings.Join(*d, ",")
}

func (d *defineArray) Set(value string) error {
	*d = append(*d, value)
	return nil
}

func main() {
	file := flag.String("f", "", "path to the .http / .rest request file (required)")
	flag.StringVar(file, "file", "", "path to the .http / .rest request file (required)")
	name := flag.String("n", "", "run only the request with this name")
	flag.StringVar(name, "name", "", "run only the request with this name")
	index := flag.Int("i", math.MinInt, "run only the request at this 0-based index")
	flag.IntVar(index, "index", math.MinInt, "run only the request at this 0-based index")
	expected := flag.String("e", "", "path to .hresp file for response assertion")
	flag.StringVar(expected, "expected", "", "path to .hresp file for response assertion")
	expectedName := flag.String("e-name", "", "expected response name in .hresp file to validate (for use with -n/-i)")
	e := flag.Int("e-index", -1, "expected response 0-based index in .hresp file to validate (for use with -n/-i)")
	expectedIndex := e
	listFlag := flag.Bool("list", false, "list named requests and indices without executing")
	flag.BoolVar(listFlag, "l", false, "list named requests and indices without executing (shorthand)")
	failOnError := flag.Bool("fail-on-error", false, "exit with code 1 on HTTP 4xx/5xx responses")
	flag.BoolVar(failOnError, "E", false, "exit with code 1 on HTTP 4xx/5xx responses (shorthand)")
	output := flag.String("o", "", "output format: body, jsonpath <expr>, env <key>")
	flag.StringVar(output, "output", "", "output format: body, jsonpath <expr>, env <key>")
	after := flag.String("after", "", "run prerequisite request by name before target (for response references)")
	flag.StringVar(after, "A", "", "run prerequisite request by name before target (shorthand)")
	var defines defineArray
	flag.Var(&defines, "D", "define variable key=value (repeatable)")
	flag.Var(&defines, "define", "define variable key=value (repeatable)")
	flag.Parse()

	if *listFlag {
		os.Exit(runList(*file))
	}
	config := runConfig{
		filePath:      *file,
		name:          *name,
		index:         *index,
		expected:      *expected,
		expectedName:  *expectedName,
		expectedIndex: *expectedIndex,
		failOnError:   *failOnError,
		output:        *output,
		after:         *after,
		defines:       defines,
	}
	os.Exit(run(config))
}

// runList handles the --list flag: parse and print request names.
func runList(filePath string) int {
	if filePath == "" {
		flushError("-f <file> is required")
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
	printRequestList(parsedFile)
	return 0
}

func run(config runConfig) int {
	if err := validateRunConfig(config); err != nil {
		flushError(err.Error())
		return 2
	}

	client, err := newClient()
	if err != nil {
		flushError(err.Error())
		return 1
	}

	if err := applyDefines(client, config.defines); err != nil {
		flushError(err.Error())
		return 1
	}

	parsedFile, err := client.ParseFile(config.filePath)
	if err != nil {
		flushError(err.Error())
		return 1
	}

	if err := runPrerequisite(client, parsedFile, config.filePath, config.after); err != nil {
		return 1
	}

	responses, execErr := executeRequests(client, parsedFile, config.filePath, config.name, config.index)
	if execErr != nil {
		flushError(execErr.Error())
		return 1
	}

	if err := validateAssertions(client, config, responses); err != nil {
		flushError(err.Error())
		return 1
	}

	return emit(responses, config.failOnError, config.output)
}

func applyDefines(client *restclient.Client, defines []string) error {
	if len(defines) == 0 {
		return nil
	}
	vars := make(map[string]any)
	for _, d := range defines {
		key, value, found := strings.Cut(d, "=")
		if !found {
			return fmt.Errorf("invalid -D/--define format: %q (expected key=value)", d)
		}
		vars[key] = value
	}
	client.SetProgrammaticVars(vars)
	return nil
}

func validateRunConfig(config runConfig) error {
	if config.filePath == "" {
		return errors.New("-f <file> is required")
	}
	if config.name != "" && config.index != math.MinInt {
		return errors.New("-n and -i are mutually exclusive")
	}
	return nil
}

func newClient() (*restclient.Client, error) {
	return restclient.NewClient()
}

func validateAssertions(
	client *restclient.Client,
	config runConfig,
	responses []*restclient.Response,
) error {
	assertErr := validateIfRequested(
		client,
		config.expected,
		config.expectedName,
		config.expectedIndex,
		responses,
	)
	return assertErr
}

// validateIfRequested runs ValidateResponses when an expected file is provided.
func validateIfRequested(
	client *restclient.Client,
	expected string,
	expectedName string,
	expectedIndex int,
	responses []*restclient.Response,
) error {
	if expected == "" {
		return nil
	}
	return client.ValidateResponsesWithOptions(expected, restclient.ValidateOptions{
		ExpectedName:  expectedName,
		ExpectedIndex: expectedIndex,
	}, responses...)
}

// executeRequests runs one or all requests and returns the responses.
func executeRequests(
	client *restclient.Client,
	parsedFile *restclient.ParsedFile,
	filePath, name string,
	index int,
) ([]*restclient.Response, error) {
	if name != "" || index != math.MinInt {
		return executeSingle(client, parsedFile, filePath, name, index)
	}
	return executeAll(client, parsedFile, filePath)
}

func executeSingle(
	client *restclient.Client,
	parsedFile *restclient.ParsedFile,
	_ string, // filePath unused
	name string,
	index int,
) ([]*restclient.Response, error) {
	reqIdx, err := findRequestIndex(parsedFile.Requests, name, index)
	if err != nil {
		return nil, err
	}

	resp, err := client.ExecuteRequest(context.Background(), parsedFile, reqIdx)
	if err != nil {
		return nil, err
	}
	return []*restclient.Response{resp}, nil
}

func executeAll(
	client *restclient.Client,
	_ *restclient.ParsedFile, // parsedFile unused
	filePath string,
) ([]*restclient.Response, error) {
	responses, err := client.ExecuteFile(context.Background(), filePath)
	if err != nil {
		return nil, err
	}
	return responses, nil
}

// findRequestIndex resolves a request name or index to its position in the list.
func runPrerequisite(client *restclient.Client, parsedFile *restclient.ParsedFile, filePath, after string) error {
	if after == "" {
		return nil
	}
	_, err := executeSingle(client, parsedFile, filePath, after, math.MinInt)
	if err != nil {
		flushError(fmt.Sprintf("prerequisite request %q failed: %v", after, err))
		return err
	}
	return nil
}

func findRequestIndex(requests []*restclient.Request, name string, index int) (int, error) {
	if name != "" {
		idx, found := findByName(requests, name)
		if !found {
			return -1, fmt.Errorf("request name %q not found", name)
		}
		return idx, nil
	}
	if index != math.MinInt {
		if index < 0 || index >= len(requests) {
			return -1, fmt.Errorf("request index %d out of range (file has %d requests)", index, len(requests))
		}
		return index, nil
	}
	return -1, errors.New("either -n or -i must be specified")
}

func findByName(requests []*restclient.Request, name string) (int, bool) {
	for i, r := range requests {
		if r.Name == name {
			return i, true
		}
	}
	return -1, false
}

// shouldFail returns true if the response indicates failure.
func shouldFail(resp *restclient.Response, failOnError bool) bool {
	if resp.Error != nil {
		return true
	}
	return failOnError && resp.StatusCode >= 400
}

func emit(responses []*restclient.Response, failOnError bool, output string) int {
	var failed bool
	out := make([]string, 0, len(responses)+1)
	for _, resp := range responses {
		out = append(out, formatOutput(resp, output))
		failed = failed || shouldFail(resp, failOnError)
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

// formatOutput formats a single response according to output mode.
func formatOutput(resp *restclient.Response, output string) string {
	if output == "" || output == "default" {
		return formatResponse(resp)
	}

	if resp.Error != nil {
		return ""
	}

	switch {
	case output == "body":
		return string(resp.Body) + "\n"
	case strings.HasPrefix(output, "jsonpath ") || strings.HasPrefix(output, "jsonpath:"):
		expr := strings.TrimSpace(strings.TrimPrefix(output, "jsonpath "))
		expr = strings.TrimSpace(strings.TrimPrefix(expr, "jsonpath:"))
		return jsonPathExtract(resp.Body, expr) + "\n"
	case strings.HasPrefix(output, "env ") || strings.HasPrefix(output, "env:"):
		key := strings.TrimSpace(strings.TrimPrefix(output, "env "))
		key = strings.TrimSpace(strings.TrimPrefix(key, "env:"))
		val := jsonPathExtract(resp.Body, key)
		return fmt.Sprintf("%s=%s\n", key, val)
	default:
		return formatResponse(resp)
	}
}

func jsonPathExtract(body []byte, expr string) string {
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}

	segments := parseOutputPath(expr)
	val := walkOutputPath(data, segments)
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

func parseOutputPath(path string) []string {
	if path == "" {
		return nil
	}
	var segments []string
	for _, part := range strings.Split(path, ".") {
		segments = append(segments, splitBracketPart(part)...)
	}
	return segments
}

func splitBracketPart(part string) []string {
	if !strings.Contains(part, "[") {
		return []string{part}
	}
	var segs []string
	before, after, found := strings.Cut(part, "[")
	_ = found
	if before != "" {
		segs = append(segs, before)
	}
	for after != "" {
		idxStr, rest, _ := strings.Cut(after, "]")
		segs = append(segs, idxStr)
		if rest == "" {
			break
		}
		after = strings.TrimPrefix(rest, ".")
	}
	return segs
}

func walkOutputPath(val any, segments []string) any {
	return walkPathRecursive(val, segments)
}

func walkPathRecursive(val any, segments []string) any {
	if len(segments) == 0 {
		return val
	}
	seg := segments[0]
	switch v := val.(type) {
	case map[string]any:
		return walkPathRecursive(v[seg], segments[1:])
	case []any:
		idx, err := strconv.Atoi(seg)
		if err != nil || idx < 0 || idx >= len(v) {
			return nil
		}
		return walkPathRecursive(v[idx], segments[1:])
	default:
		return nil
	}
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

func flushOutput(s string) error {
	_, err := os.Stdout.WriteString(s)
	return err
}

func flushError(msg string) {
	_, _ = fmt.Fprintf(os.Stderr, "error: %s\n", msg)
}

// printRequestList prints numbered list of requests with names.
func printRequestList(parsedFile *restclient.ParsedFile) {
	for i, req := range parsedFile.Requests {
		name := req.Name
		if name == "" {
			name = "(unnamed)"
		}
		_, _ = fmt.Fprintf(os.Stdout, "%d  %s\n", i, name)
	}
}