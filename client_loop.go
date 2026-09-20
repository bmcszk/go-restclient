package restclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"net/http"
)

// loopIteration carries the per-iteration state for `@loop` request expansion.
type loopIteration struct {
	index int // 0-based iteration index, exposed via {{$index}}
	item  any // current iteration value (nil for `for N` loops)
}

// runRequestWithLoops handles the loop expansion of a single request during ExecuteFile.
func (c *Client) runRequestWithLoops(
	ctx context.Context,
	restClientReq *Request,
	index int,
	parsedFile *ParsedFile,
	refState *refExecutionState,
	osEnvGetter func(string) (string, bool),
	responses *[]*Response,
	multiErr **multierror.Error,
) (handled bool) {
	iterations, err := c.resolveLoopIterations(restClientReq, parsedFile, osEnvGetter)
	if err != nil {
		*multiErr = multierror.Append(*multiErr, err)
		return true
	}
	isLooped := c.isLoopedRequest(restClientReq)
	if isLooped && len(iterations) == 0 {
		// No-op loop (N<=0 / empty collection): no refs, no response entries, no error.
		return true
	}
	if err := c.resolveRequestRefs(ctx, restClientReq, parsedFile, refState, osEnvGetter); err != nil {
		*multiErr = multierror.Append(*multiErr, err)
		return true
	}
	if !isLooped {
		return false
	}
	c.runLoopIterations(ctx, restClientReq, iterations, parsedFile, osEnvGetter, index, responses, multiErr)
	// Plain name addressing resolves to the first iteration's response (zero-based name0).
	if resp := parsedFile.ResponseMap[loopResponseName(restClientReq.Name, 0)]; resp != nil {
		refState.recordExecuted(restClientReq.Name, resp)
	}
	return true
}

// executeLoopAndStore handles the loop expansion of a single request during ExecuteRequest.
func (c *Client) executeLoopAndStore(
	ctx context.Context,
	restClientReq *Request,
	parsedFile *ParsedFile,
	osEnvGetter func(string) (string, bool),
	index int,
) (*Response, error) {
	iterations, err := c.resolveLoopIterations(restClientReq, parsedFile, osEnvGetter)
	if err != nil {
		return &Response{Request: restClientReq, Error: err}, err
	}
	if len(iterations) == 0 {
		// No-op loop (N<=0 / empty collection): no response entries, no error.
		return nil, nil
	}
	first := c.runLoopIterationsForStore(ctx, restClientReq, iterations, parsedFile, osEnvGetter, index)
	restClientReq.loopIterationIndex = 0
	restClientReq.loopIterationValue = nil
	return first, nil
}

// runLoopIterationsForStore runs each loop iteration, stores each under `nameN`, returns the first response.
func (c *Client) runLoopIterationsForStore(
	ctx context.Context,
	restClientReq *Request,
	iterations []loopIteration,
	parsedFile *ParsedFile,
	osEnvGetter func(string) (string, bool),
	index int,
) *Response {
	var first *Response
	for i, iter := range iterations {
		restClientReq.loopIterationIndex = iter.index
		restClientReq.loopIterationValue = iter.item
		resetLoopIterationState(restClientReq)
		sleepFirstIteration(restClientReq, i)
		resp := c.runOneLoopIteration(ctx, restClientReq, parsedFile, osEnvGetter, index)
		storeLoopResponse(parsedFile, restClientReq, i, resp)
		if i == 0 {
			first = resp
		}
	}
	return first
}

// sleepFirstIteration applies the @sleep directive before the first iteration only.
func sleepFirstIteration(req *Request, i int) {
	if i == 0 && req.SleepDuration > 0 {
		time.Sleep(req.SleepDuration)
	}
}

// runOneLoopIteration executes a single loop iteration, ensuring a non-nil *Response on exec error.
func (c *Client) runOneLoopIteration(
	ctx context.Context,
	restClientReq *Request,
	parsedFile *ParsedFile,
	osEnvGetter func(string) (string, bool),
	index int,
) *Response {
	resp, execErr := c.executeRequestWithVariables(ctx, restClientReq, parsedFile, osEnvGetter, index)
	if execErr != nil && resp == nil {
		resp = &Response{Request: restClientReq, Error: execErr}
	}
	return resp
}

// resolveLoopIterations computes the iteration values for a looped request at execution time.
func (c *Client) resolveLoopIterations(
	req *Request,
	parsedFile *ParsedFile,
	osEnvGetter func(string) (string, bool),
) ([]loopIteration, error) {
	switch {
	case req.LoopFor > 0:
		return iterateByCount(req.LoopFor), nil
	case req.LoopExpr != "":
		return c.resolveLoopExprIterations(req.LoopExpr, parsedFile, req, osEnvGetter)
	case req.LoopCollection != "":
		return c.resolveLoopCollectionIterations(req.LoopCollection, parsedFile, req, osEnvGetter)
	}
	return nil, nil
}

// iterateByCount builds n iterations where item == index == i (0-based).
func iterateByCount(n int) []loopIteration {
	out := make([]loopIteration, n)
	for i := range out {
		out[i] = loopIteration{index: i, item: i}
	}
	return out
}

// resolveLoopExprIterations resolves @loop `{{var}}` to an iteration count and builds iterations.
func (c *Client) resolveLoopExprIterations(
	expr string,
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
) ([]loopIteration, error) {
	n, err := c.resolveLoopExprCount(expr, parsedFile, restClientReq, osEnvGetter)
	if err != nil {
		return nil, err
	}
	if n <= 0 {
		return nil, nil
	}
	return iterateByCount(n), nil
}

// resolveLoopCollectionIterations resolves the @loop collection variable and builds iterations for each element.
func (c *Client) resolveLoopCollectionIterations(
	collName string,
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
) ([]loopIteration, error) {
	items, err := c.resolveLoopCollection(collName, parsedFile, restClientReq, osEnvGetter)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	out := make([]loopIteration, len(items))
	for i, it := range items {
		out[i] = loopIteration{index: i, item: it}
	}
	return out, nil
}

// resolveLoopExprCount resolves a `{{var}}` expression to a non-negative integer iteration count.
func (c *Client) resolveLoopExprCount(
	expr string,
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
) (int, error) {
	requestScopedSystemVars := c.generateRequestScopedSystemVariables()
	rctx := resolveContext{
		programmaticVars: c.programmaticVars,
		fileScopedVars:   restClientReq.ActiveVariables,
		environmentVars:  parsedFile.EnvironmentVariables,
		globalVars:       parsedFile.GlobalVariables,
		systemVars:       requestScopedSystemVars,
		osEnvGetter:      osEnvGetter,
		dotEnvVars:       c.currentDotEnvVars,
		responseMap:      parsedFile.ResponseMap,
	}
	if missing := findFirstUndefinedDisabledExprVar(expr, rctx); missing != "" {
		return 0, fmt.Errorf("undefined variable %q in @loop expression", missing)
	}
	resolved := resolveVariablesInText(expr, rctx)
	resolved = substituteDynamicSystemVariables(resolved, c.currentDotEnvVars, c.programmaticVars)
	resolved = strings.TrimSpace(resolved)
	if resolved == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(resolved)
	if err != nil {
		return 0, fmt.Errorf("@loop expression did not resolve to a non-negative integer: %q", resolved)
	}
	return n, nil
}

// resolveLoopCollection looks up a collection variable and decodes it into a slice of any.
func (c *Client) resolveLoopCollection(
	collName string,
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
) ([]any, error) {
	rctx := c.loopResolveContext(parsedFile, restClientReq, osEnvGetter)
	val, ok := lookupCollectionVar(collName, rctx)
	if !ok {
		return nil, fmt.Errorf("undefined variable %q in @loop collection", collName)
	}
	return decodeCollectionValue(val, collName)
}

// loopResolveContext builds the resolveContext used for @loop variable lookups.
func (c *Client) loopResolveContext(
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
) resolveContext {
	return resolveContext{
		programmaticVars: c.programmaticVars,
		fileScopedVars:   restClientReq.ActiveVariables,
		environmentVars:  parsedFile.EnvironmentVariables,
		globalVars:       parsedFile.GlobalVariables,
		systemVars:       c.generateRequestScopedSystemVariables(),
		osEnvGetter:      osEnvGetter,
		dotEnvVars:       c.currentDotEnvVars,
		responseMap:      parsedFile.ResponseMap,
	}
}

// decodeCollectionValue decodes a resolved collection variable into []any; supports []any, []string, JSON strings.
func decodeCollectionValue(val any, collName string) ([]any, error) {
	switch x := val.(type) {
	case []any:
		return x, nil
	case []string:
		return liftStrings(x), nil
	case string:
		return decodeStringCollection(x, collName)
	}
	return nil, fmt.Errorf("@loop collection variable %q is not an array", collName)
}

// liftStrings copies []string into a freshly-allocated []any.
func liftStrings(x []string) []any {
	out := make([]any, len(x))
	for i, s := range x {
		out[i] = s
	}
	return out
}

// decodeStringCollection parses a JSON-encoded string collection, returning an empty slice for empty input.
func decodeStringCollection(s string, collName string) ([]any, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil, nil
	}
	if trimmed[0] != '[' {
		return nil, fmt.Errorf("@loop collection variable %q is not an array", collName)
	}
	var arr []any
	if err := decodeJSONInto(trimmed, &arr); err != nil {
		return nil, fmt.Errorf("@loop collection variable %q is not a JSON array: %w", collName, err)
	}
	return arr, nil
}

// lookupCollectionVar searches the same variable sources as the @disabled expr resolver.
func lookupCollectionVar(name string, rctx resolveContext) (any, bool) {
	if v, ok := rctx.programmaticVars[name]; ok {
		return v, true
	}
	if v, ok := rctx.fileScopedVars["@"+name]; ok {
		return v, true
	}
	if v, ok := stringMapLookup(rctx.environmentVars, name); ok {
		return v, true
	}
	if v, ok := stringMapLookup(rctx.globalVars, name); ok {
		return v, true
	}
	if v, ok := stringMapLookup(rctx.dotEnvVars, name); ok {
		return v, true
	}
	return lookupFromOSEnv(rctx, name)
}

// stringMapLookup returns the value for name from a string map; small wrapper for the lookup chain.
func stringMapLookup(m map[string]string, name string) (string, bool) {
	v, ok := m[name]
	return v, ok
}

// lookupFromOSEnv probes the OS environment getter when configured.
func lookupFromOSEnv(rctx resolveContext, name string) (any, bool) {
	if rctx.osEnvGetter == nil {
		return nil, false
	}
	return rctx.osEnvGetter(name)
}

// decodeJSONInto wraps encoding/json's Unmarshal for clarity in this file's callers.
func decodeJSONInto(data string, into any) error {
	return json.Unmarshal([]byte(data), into)
}

// runLoopIterations executes each iteration of a looped request and stores the responses.
func (c *Client) runLoopIterations(
	ctx context.Context,
	req *Request,
	iterations []loopIteration,
	parsedFile *ParsedFile,
	osEnvGetter func(string) (string, bool),
	index int,
	responses *[]*Response,
	multiErr **multierror.Error,
) {
	for i, iter := range iterations {
		req.loopIterationIndex = iter.index
		req.loopIterationValue = iter.item
		resetLoopIterationState(req)
		if i == 0 && req.SleepDuration > 0 {
			time.Sleep(req.SleepDuration)
		}
		response, err := c.executeRequestWithVariables(ctx, req, parsedFile, osEnvGetter, index)
		response, shouldSkip := c.handleRequestExecutionError(response, err, req, index, multiErr)
		if shouldSkip || response == nil {
			continue
		}
		*responses = append(*responses, response)
		storeLoopResponse(parsedFile, req, i, response)
	}
	// Clear transient loop state so the request looks un-looped after iteration ends.
	req.loopIterationIndex = 0
	req.loopIterationValue = nil
}

// resetLoopIterationState restores RawBody / Headers to the parse-time snapshot for re-substitution.
func resetLoopIterationState(req *Request) {
	if req.loopOriginalRawBody != "" || req.ExternalFilePath == "" {
		req.RawBody = req.loopOriginalRawBody
	}
	if req.loopOriginalHeaders != nil {
		restored := make(http.Header, len(req.loopOriginalHeaders))
		for k, vs := range req.loopOriginalHeaders {
			restored[k] = append([]string(nil), vs...)
		}
		req.Headers = restored
	}
}

// storeLoopResponse stores a loop iteration's response under `nameN` for every iteration.
func storeLoopResponse(parsedFile *ParsedFile, req *Request, iterIndex int, resp *Response) {
	if req.Name == "" || resp == nil {
		return
	}
	parsedFile.ResponseMap[loopResponseName(req.Name, iterIndex)] = resp
}

// loopResponseName builds the `nameN` key for a given iteration index.
func loopResponseName(name string, iterIndex int) string {
	return fmt.Sprintf("%s%d", name, iterIndex)
}
