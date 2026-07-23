package restclient

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// reResponseRef matches {{name.response.body.X}}, {{name.response.headers.X}}, {{name.response.status}}.
var reResponseRef = regexp.MustCompile(
	`{{\s*([a-zA-Z_][a-zA-Z0-9_ -]*)\.response\.` +
		`(status|headers\.([a-zA-Z0-9_-]+)|body(?:\.([a-zA-Z0-9_\[\]\."'-]+))?)\s*}}`)

// resolveResponseReference resolves {{name.response.body.X}}, {{name.response.headers.X}}, {{name.response.status}}.
func resolveResponseReference(varName string, responseMap map[string]*Response) string {
	if responseMap == nil {
		return ""
	}

	matches := reResponseRef.FindStringSubmatch("{{" + varName + "}}")
	if matches == nil {
		return ""
	}

	requestName := matches[1]
	resp, ok := responseMap[requestName]
	if !ok {
		return ""
	}

	matchedPart := matches[2] // "status", "headers.X", or "body.field"
	headerName := matches[3]  // header name (if headers branch)
	bodyPath := matches[4]    // body path (if body branch)
	statusRef := ""
	if matchedPart == "status" {
		statusRef = "status"
	}

	switch {
	case strings.HasPrefix(matchedPart, "body"):
		return resolveResponseBody(bodyPath, resp)
	case headerName != "":
		return resolveResponseHeader(headerName, resp)
	case statusRef != "":
		return strconv.Itoa(resp.StatusCode)
	}

	return ""
}

// resolveResponseBody resolves {{name.response.body}} or {{name.response.body.path.to.field}}.
// bodyPath is the captured group after "body" — empty for whole body, or "field.nested" for sub-paths.
func resolveResponseBody(bodyPath string, resp *Response) string {
	if bodyPath == "" {
		return resp.BodyString
	}
	return resolveJSONPath(bodyPath, resp.Body)
}

// resolveResponseHeader resolves {{name.response.headers.HeaderName}}.
func resolveResponseHeader(headerName string, resp *Response) string {
	return resp.Headers.Get(headerName)
}

// resolveJSONPath resolves a simple dot-notation path against a JSON body.
// Supports: field, field.nested, array[0], field.array[0].nested.
func resolveJSONPath(path string, body []byte) string {
	if len(body) == 0 {
		return ""
	}

	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return ""
	}

	segments := parseJSONPathSegments(path)
	return walkJSONPath(raw, segments)
}

// reJSONPathSegment matches a segment: either a name or [index].
var reJSONPathSegment = regexp.MustCompile(`([^[.]+)|\[(\d+)\]`)

// parseJSONPathSegments splits a path like "data.items[0].name" into segments.
func parseJSONPathSegments(path string) []string {
	matches := reJSONPathSegment.FindAllStringSubmatch(path, -1)
	segments := make([]string, 0, len(matches))
	for _, m := range matches {
		if m[1] != "" {
			segments = append(segments, m[1])
		} else if m[2] != "" {
			segments = append(segments, m[2])
		}
	}
	return segments
}

// walkJSONPath traverses a parsed JSON value following the given segments.
func walkJSONPath(val any, segments []string) string {
	current := val
	for _, seg := range segments {
		current = walkJSONStep(current, seg)
		if current == nil {
			return ""
		}
	}
	return jsonStringify(current)
}

// walkJSONStep advances one segment into a JSON value.
func walkJSONStep(val any, seg string) any {
	switch v := val.(type) {
	case map[string]any:
		next, ok := v[seg]
		if !ok {
			return nil
		}
		return next
	case []any:
		return walkJSONArray(v, seg)
	default:
		return nil
	}
}

// walkJSONArray indexes into a JSON array by segment.
func walkJSONArray(arr []any, seg string) any {
	idx, err := strconv.Atoi(seg)
	if err != nil || idx < 0 || idx >= len(arr) {
		return nil
	}
	return arr[idx]
}

// jsonStringify converts a JSON value to its string representation.
func jsonStringify(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return ""
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}
