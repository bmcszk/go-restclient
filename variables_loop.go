package restclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

// resolveLoopItemReference resolves a loop item ref like `item`, `item.name`, or `alias.sub[0].field`.
func resolveLoopItemReference(varName string, aliases map[string]any) string {
	if len(aliases) == 0 {
		return ""
	}
	root, rest, hasPath := strings.Cut(varName, ".")
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	val, ok := aliases[root]
	if !ok {
		return ""
	}
	if !hasPath {
		return stringifyLoopItemValue(val)
	}
	segments := parseJSONPathSegments(rest)
	return walkJSONPath(val, segments)
}

// stringifyLoopItemValue formats a single (non-path) loop item value as a string for substitution.
func stringifyLoopItemValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return ""
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", x)
		}
		return string(b)
	}
}
