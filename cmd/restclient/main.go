// Command restclient executes HTTP requests defined in a .http / .rest file.
//
// Install: go install github.com/bmcszk/go-restclient/cmd/restclient@latest
// Usage:   restclient -f requests.http [-n name | -i index] [-e expected.hresp]
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/bmcszk/go-restclient"
)

type cli struct {
	File          string   `short:"f" long:"file" required:"true" help:"Request file path" group:"required"`
	Name          string   `short:"n" long:"name" help:"Run request by name" group:"selection"`
	Index         *int     `short:"i" long:"index" help:"Run request by index" group:"selection"`
	All           bool     `long:"all" help:"Run all requests in file" group:"selection"`
	Expected      string   `short:"e" long:"expected" help:"Expected response file" group:"validation"`
	ExpectedName  string   `long:"e-name" help:"Expected response name" group:"validation"`
	ExpectedIndex int      `long:"e-index" help:"Expected response index" default:"-1" group:"validation"`
	List          bool     `short:"l" long:"list" help:"List requests" group:"output"`
	FailOnError   bool     `short:"E" long:"fail-on-error" help:"Fail on 4xx/5xx" group:"output"`
	Output        string   `short:"o" long:"output" help:"Output format: body, jsonpath, env" group:"output"`
	After         string   `short:"A" long:"after" help:"Prerequisite request" group:"output"`
	Define        []string `short:"D" long:"define" help:"Define variable key=value" group:"variables"`
}

var c cli

func main() {
	_ = kong.Parse(&c,
		kong.Name("restclient"),
		kong.Description("Execute HTTP requests defined in a .http / .rest file"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true, Summary: true}),
	)

	if c.List {
		os.Exit(runList(c.File))
	}
	os.Exit(run(c))
}

func run(c cli) int {
	client, parsedFile, err := setupClient(c)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	if err := runPrerequisite(client, parsedFile, c.File, c.After); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	responses, execErr := dispatchExecution(client, parsedFile, c)
	if execErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", execErr)
		return 1
	}

	if c.Expected != "" {
		if err := validateAssertions(client, c, responses); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
	}

	return emit(responses, c.FailOnError, c.Output)
}

func dispatchExecution(
	client *restclient.Client,
	parsedFile *restclient.ParsedFile,
	c cli,
) ([]*restclient.Response, error) {
	if c.All {
		return client.ExecuteFile(context.Background(), c.File)
	}
	if c.Name != "" || c.Index != nil {
		idx := math.MinInt
		if c.Index != nil {
			idx = *c.Index
		}
		return executeSingle(client, parsedFile, c.Name, idx)
	}
	return nil, errors.New("specify -n NAME, -i INDEX, or --all to run requests")
}

func setupClient(c cli) (*restclient.Client, *restclient.ParsedFile, error) {
	if err := validateConfig(c); err != nil {
		return nil, nil, err
	}
	client, err := newClient()
	if err != nil {
		return nil, nil, err
	}
	if err := applyDefines(client, c.Define); err != nil {
		return nil, nil, err
	}
	parsedFile, err := client.ParseFile(c.File)
	if err != nil {
		return nil, nil, err
	}
	return client, parsedFile, nil
}

func validateConfig(c cli) error {
	if c.Name != "" && c.Index != nil {
		return errors.New("-n and -i are mutually exclusive")
	}
	return nil
}

func newClient() (*restclient.Client, error) {
	return restclient.NewClient()
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

func runPrerequisite(client *restclient.Client, parsedFile *restclient.ParsedFile, _, after string) error {
	if after == "" {
		return nil
	}
	_, err := executeSingle(client, parsedFile, after, math.MinInt)
	if err != nil {
		return fmt.Errorf("prerequisite request %q failed: %v", after, err)
	}
	return nil
}

func validateAssertions(client *restclient.Client, c cli, responses []*restclient.Response) error {
	return client.ValidateResponsesWithOptions(c.Expected, restclient.ValidateOptions{
		ExpectedName:  c.ExpectedName,
		ExpectedIndex: c.ExpectedIndex,
	}, responses...)
}

func runList(filePath string) int {
	if filePath == "" {
		_, _ = fmt.Fprint(os.Stderr, "error: -f <file> is required\n")
		return 2
	}
	client, err := restclient.NewClient()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	parsedFile, err := client.ParseFile(filePath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	printRequestList(parsedFile)
	return 0
}


func executeSingle(
	client *restclient.Client,
	parsedFile *restclient.ParsedFile,
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

func findRequestIndex(
	requests []*restclient.Request,
	name string,
	index int,
) (int, error) {
	if name != "" {
		return findByName(requests, name)
	}
	return findByIndex(requests, index)
}

func findByName(requests []*restclient.Request, name string) (int, error) {
	for i, r := range requests {
		if r.Name == name {
			return i, nil
		}
	}
	return -1, fmt.Errorf("request name %q not found", name)
}

func findByIndex(requests []*restclient.Request, index int) (int, error) {
	if index == math.MinInt {
		return -1, errors.New("either -n or -i must be specified")
	}
	if index < 0 || index >= len(requests) {
		return -1, fmt.Errorf("request index %d out of range (file has %d requests)", index, len(requests))
	}
	return index, nil
}

func shouldFail(resp *restclient.Response, failOnError bool) bool {
	return resp.Error != nil || (failOnError && resp.StatusCode >= 400)
}

func emit(responses []*restclient.Response, failOnError bool, output string) int {
	var failed bool
	var out []string
	for _, resp := range responses {
		out = append(out, formatOutput(resp, output))
		failed = failed || shouldFail(resp, failOnError)
	}
	if err := flushOutput(strings.Join(out, "")); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	if failed {
		return 1
	}
	return 0
}

func formatOutput(resp *restclient.Response, output string) string {
	if output == "" {
		return formatResponse(resp)
	}
	if resp.Error != nil {
		return ""
	}
	switch {
	case output == "body":
		return string(resp.Body) + "\n"
	case strings.HasPrefix(output, "jsonpath"):
		expr := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(output, "jsonpath"), ": "))
		return jsonPathExtract(resp.Body, expr) + "\n"
	case strings.HasPrefix(output, "env"):
		key := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(output, "env"), ": "))
		return fmt.Sprintf("%s=%s\n", key, jsonPathExtract(resp.Body, key))
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
	before, after, _ := strings.Cut(part, "[")
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
	if len(segments) == 0 {
		return val
	}
	seg := segments[0]
	switch v := val.(type) {
	case map[string]any:
		return walkOutputPath(v[seg], segments[1:])
	case []any:
		idx, err := strconv.Atoi(seg)
		if err != nil || idx < 0 || idx >= len(v) {
			return nil
		}
		return walkOutputPath(v[idx], segments[1:])
	default:
		return nil
	}
}

func formatResponse(resp *restclient.Response) string {
	if resp.Error != nil {
		return requestLine(resp) + "\n"
	}
	return requestLine(resp) + "\n" +
		fmt.Sprintf("  %s %s (%s)\n", resp.Proto, resp.Status, resp.Duration) +
		formatHeaders(resp.Headers) +
		formatBody(resp.Body) + "\n"
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
	var lines []string
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

func printRequestList(parsedFile *restclient.ParsedFile) {
	for i, req := range parsedFile.Requests {
		name := req.Name
		if name == "" {
			name = "(unnamed)"
		}
		_, _ = fmt.Fprintf(os.Stdout, "%d  %s\n", i, name)
	}
}