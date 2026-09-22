package restclient

// @ref/@forceRef dependency execution: ref state, cycle detection, and response caching.

import (
	"context"
	"fmt"
)

// refExecutionState tracks per-ExecuteFile execution of @ref/@forceRef dependencies.
type refExecutionState struct {
	executed map[string]*Response // name -> most recent response
	onStack  map[string]bool      // names currently being resolved (cycle detection)
}

func newRefExecutionState() *refExecutionState {
	return &refExecutionState{
		executed: make(map[string]*Response),
		onStack:  make(map[string]bool),
	}
}

func (s *refExecutionState) alreadyExecuted(name string) bool {
	return name != "" && s.executed[name] != nil
}

func (s *refExecutionState) recordExecuted(name string, resp *Response) {
	if name != "" && resp != nil {
		s.executed[name] = resp
	}
}

func (s *refExecutionState) pushStack(name string) func() {
	s.onStack[name] = true
	return func() { delete(s.onStack, name) }
}

// resolveRequestRefs resolves refs depth-first; @ref caches, @forceRef re-runs.
func (c *Client) resolveRequestRefs(
	ctx context.Context,
	req *Request,
	parsedFile *ParsedFile,
	state *refExecutionState,
	osEnvGetter func(string) (string, bool),
) error {
	for _, ref := range req.Refs {
		if refCacheReuseable(ref, state) {
			continue
		}
		if err := c.executeReferencedRequest(ctx, ref, parsedFile, state, osEnvGetter); err != nil {
			return err
		}
	}
	return nil
}

func refCacheReuseable(ref RequestRef, state *refExecutionState) bool {
	return !ref.Force && state.executed[ref.Name] != nil
}

func (c *Client) executeReferencedRequest(
	ctx context.Context,
	ref RequestRef,
	parsedFile *ParsedFile,
	state *refExecutionState,
	osEnvGetter func(string) (string, bool),
) error {
	if state.onStack[ref.Name] {
		return fmt.Errorf("cycle detected in @ref graph involving request %q", ref.Name)
	}
	target := findRequestByName(parsedFile, ref.Name)
	if target == nil {
		return fmt.Errorf("unknown referenced request %q", ref.Name)
	}

	popStack := state.pushStack(ref.Name)
	defer popStack()

	if err := c.resolveRequestRefs(ctx, target, parsedFile, state, osEnvGetter); err != nil {
		return err
	}
	if !ref.Force && state.executed[ref.Name] != nil {
		return nil
	}
	c.runAndCacheReferenced(ctx, target, ref.Name, parsedFile, state, osEnvGetter)
	return nil
}

func findRequestByName(parsedFile *ParsedFile, name string) *Request {
	for _, r := range parsedFile.Requests {
		if r.Name == name {
			return r
		}
	}
	return nil
}

func (c *Client) runAndCacheReferenced(
	ctx context.Context,
	target *Request,
	name string,
	parsedFile *ParsedFile,
	state *refExecutionState,
	osEnvGetter func(string) (string, bool),
) {
	response, _ := c.executeRequestWithVariables(ctx, target, parsedFile, osEnvGetter, requestIndex(parsedFile, target))
	state.recordExecuted(name, response)
	if response != nil {
		storeResponse(parsedFile, target, response)
	}
}

func requestIndex(parsedFile *ParsedFile, target *Request) int {
	for i, r := range parsedFile.Requests {
		if r == target {
			return i
		}
	}
	return -1
}

// storeResponse saves a response in the response map keyed by request name.
func storeResponse(parsedFile *ParsedFile, req *Request, resp *Response) {
	if req.Name != "" {
		parsedFile.ResponseMap[req.Name] = resp
	}
}
