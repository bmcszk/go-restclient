package restclient

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// evaluateDisabledExpr returns the truthiness of a substituted @disabled !<expr>.
func (c *Client) evaluateDisabledExpr(
	expr string,
	parsedFile *ParsedFile,
	restClientReq *Request,
	osEnvGetter func(string) (string, bool),
) (bool, error) {
	rctx := c.directiveResolveContext(parsedFile, restClientReq, osEnvGetter, nil, nil, -1)

	if missing := findFirstUndefinedDisabledExprVar(expr, rctx); missing != "" {
		return false, fmt.Errorf("undefined variable %q in @disabled expr", missing)
	}

	resolved := resolveVariablesInText(expr, rctx)
	resolved = substituteDynamicSystemVariables(resolved, c.currentDotEnvVars, c.programmaticVars)

	return computeDisabledTruthiness(resolved), nil
}

// findFirstUndefinedDisabledExprVar returns the first unresolvable {{name}} in expr, or "".
func findFirstUndefinedDisabledExprVar(expr string, rctx resolveContext) string {
	re := regexp.MustCompile(`{{\s*([^}]+?)\s*}}`)
	for _, match := range re.FindAllStringSubmatch(expr, -1) {
		directive := strings.TrimSpace(match[1])
		varName := directive
		if i := strings.Index(directive, "|"); i >= 0 {
			varName = strings.TrimSpace(directive[:i])
		}
		if strings.HasPrefix(varName, "$") {
			continue
		}
		if _, ok := lookupVar(varName, rctx); !ok {
			return varName
		}
	}
	return ""
}

// lookupVar resolves name through the standard variable sources, nil-safe.
func lookupVar(name string, rctx resolveContext) (any, bool) {
	if v, ok := rctx.programmaticVars[name]; ok {
		return v, true
	}
	if v, ok := rctx.fileScopedVars["@"+name]; ok {
		return v, true
	}
	if v, ok := rctx.environmentVars[name]; ok {
		return v, true
	}
	if v, ok := rctx.globalVars[name]; ok {
		return v, true
	}
	if v, ok := rctx.dotEnvVars[name]; ok {
		return v, true
	}
	if rctx.osEnvGetter == nil {
		return nil, false
	}
	return rctx.osEnvGetter(name)
}

// computeDisabledTruthiness applies @disabled rules: true/non-zero/non-empty are truthy; false/zero/empty are falsy.
func computeDisabledTruthiness(s string) bool {
	switch strings.ToLower(s) {
	case "true":
		return true
	case "false":
		return false
	}
	if i, err := strconv.Atoi(s); err == nil {
		return i != 0
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f != 0
	}
	return s != ""
}
