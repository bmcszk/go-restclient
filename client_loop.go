package restclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// loopIteration carries the per-iteration state for `@loop` request expansion.
type loopIteration struct {
	index int // 0-based iteration index, exposed via {{$index}}
	item  any // current iteration value (nil for `for N` loops)
}

// runRequestWithLoops runs the request as a @loop when declared; returns handled, appended responses, and the error.
func (c *Client) runRequestWithLoops(
	ctx context.Context,
	restClientReq *Request,
	index int,
	parsedFile *ParsedFile,
	refState *refExecutionState,
	osEnvGetter func(string) (string, bool),
) (handled bool, responses []*Response, err error) {
	iterations, iterErr := c.resolveLoopIterations(restClientReq, parsedFile, osEnvGetter)
	if iterErr != nil {
		return true, nil, iterErr
	}
	isLooped := c.isLoopedRequest(restClientReq)
	if isLooped && len(iterations) == 0 {
		// No-op loop (N<=0 / empty collection): no refs, no response entries, no error.
		return true, nil, nil
	}
	if refErr := c.resolveRequestRefs(ctx, restClientReq, parsedFile, refState, osEnvGetter); refErr != nil {
		return true, nil, refErr
	}
	if !isLooped {
		return false, nil, nil
	}
	iterResponses, loopErr := c.runLoopIterations(ctx, restClientReq, iterations, parsedFile, osEnvGetter, index)
	if loopErr != nil {
		return true, nil, loopErr
	}
	// Plain name addressing resolves to the first iteration's response (zero-based name0).
	if resp := parsedFile.ResponseMap[loopResponseName(restClientReq.Name, 0)]; resp != nil {
		refState.recordExecuted(restClientReq.Name, resp)
	}
	return true, iterResponses, nil
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
	iterResponses, runErr := c.runLoopIterations(ctx, restClientReq, iterations, parsedFile, osEnvGetter, index)
	restClientReq.loopIterationIndex = 0
	restClientReq.loopIterationValue = nil
	if runErr != nil {
		return &Response{Request: restClientReq, Error: runErr}, runErr
	}
	return iterResponses[0], nil
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
	rctx := c.directiveResolveContext(parsedFile, restClientReq, osEnvGetter, nil, nil, -1)
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
	rctx := c.directiveResolveContext(parsedFile, restClientReq, osEnvGetter, nil, nil, -1)
	val, ok := lookupVar(collName, rctx)
	if !ok {
		return nil, fmt.Errorf("undefined variable %q in @loop collection", collName)
	}
	return decodeCollectionValue(val, collName)
}

// directiveResolveContext builds the client-scoped resolveContext used by all
// variable substitution sites; loopItemAliases/loopIndex are nil/-1 outside loop iterations.
func (c *Client) directiveResolveContext(
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
	systemVars map[string]string,
	loopItemAliases map[string]any,
	loopIndex int,
) resolveContext {
	var environmentVars, globalVars map[string]string
	var responseMap map[string]*Response
	if parsedFile != nil {
		environmentVars = parsedFile.EnvironmentVariables
		globalVars = parsedFile.GlobalVariables
		responseMap = parsedFile.ResponseMap
	}
	return resolveContext{
		programmaticVars: c.programmaticVars,
		fileScopedVars:   restClientReq.ActiveVariables,
		environmentVars:  environmentVars,
		globalVars:       globalVars,
		systemVars:       systemVars,
		osEnvGetter:      osEnvGetter,
		dotEnvVars:       c.currentDotEnvVars,
		responseMap:      responseMap,
		loopItemAliases:  loopItemAliases,
		loopIndex:        loopIndex,
	}
}

// newDotEnvResolveContext builds the minimal dotenv-scoped resolveContext for
// expanding placeholders inside .env values (OS env + dotenv lookups only).
func newDotEnvResolveContext(dotEnvVars map[string]string) resolveContext {
	return resolveContext{
		osEnvGetter: os.LookupEnv,
		dotEnvVars:  dotEnvVars,
	}
}
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
	if err := json.Unmarshal([]byte(trimmed), &arr); err != nil {
		return nil, fmt.Errorf("@loop collection variable %q is not a JSON array: %w", collName, err)
	}
	return arr, nil
}

// runLoopIterations executes each loop iteration, stores each under nameN; aborts on first iteration error.
func (c *Client) runLoopIterations(
	ctx context.Context,
	req *Request,
	iterations []loopIteration,
	parsedFile *ParsedFile,
	osEnvGetter func(string) (string, bool),
	index int,
) ([]*Response, error) {
	var responses []*Response
	for i, iter := range iterations {
		resp, err := c.runLoopIteration(ctx, req, parsedFile, osEnvGetter, index, i, iter)
		if err != nil {
			return responses, err
		}
		if resp != nil {
			responses = append(responses, resp)
		}
	}
	return responses, nil
}

// runLoopIteration executes a single loop iteration and stores its response under nameN.
func (c *Client) runLoopIteration(
	ctx context.Context,
	req *Request,
	parsedFile *ParsedFile,
	osEnvGetter func(string) (string, bool),
	index int,
	i int,
	iter loopIteration,
) (*Response, error) {
	req.loopIterationIndex = iter.index
	req.loopIterationValue = iter.item
	resetLoopIterationState(req)
	if i == 0 && req.SleepDuration > 0 {
		time.Sleep(req.SleepDuration)
	}
	resp, execErr := c.executeRequestWithVariables(ctx, req, parsedFile, osEnvGetter, index)
	if execErr != nil {
		if resp == nil {
			resp = &Response{Request: req, Error: execErr}
		}
		storeLoopResponse(parsedFile, req, i, resp)
		return nil, fmt.Errorf("request %q: %w", req.Name, execErr)
	}
	if resp != nil {
		storeLoopResponse(parsedFile, req, i, resp)
	}
	return resp, nil
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
