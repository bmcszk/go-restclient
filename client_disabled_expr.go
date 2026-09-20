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
		if !disabledExprVariableExists(varName, rctx) {
			return varName
		}
	}
	return ""
}

// disabledExprVariableExists reports whether varName resolves through any standard variable source.
func disabledExprVariableExists(varName string, rctx resolveContext) bool {
	lookups := []func(string) bool{
		func(v string) bool { _, ok := rctx.programmaticVars[v]; return ok },
		func(v string) bool { return fileScopedVarExists(rctx.fileScopedVars, v) },
		func(v string) bool { _, ok := rctx.environmentVars[v]; return ok },
		func(v string) bool { _, ok := rctx.globalVars[v]; return ok },
		func(v string) bool { _, ok := rctx.dotEnvVars[v]; return ok },
	}
	for _, has := range lookups {
		if has(varName) {
			return true
		}
	}
	if rctx.osEnvGetter != nil {
		if _, ok := rctx.osEnvGetter(varName); ok {
			return true
		}
	}
	return false
}

// fileScopedVarExists reports whether m (nil-safe) contains @v.
func fileScopedVarExists(m map[string]string, v string) bool {
	if m == nil {
		return false
	}
	_, ok := m["@"+v]
	return ok
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
