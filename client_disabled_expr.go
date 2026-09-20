package restclient

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// evaluateDisabledExpr decides whether a conditional @disabled !<expr> should skip
// the request. expr is run through the standard {{var}} substitution used for URLs
// and bodies; the resolved string is evaluated as truthy when it equals the
// case-insensitive literal "true", a non-zero number, or any other non-empty string
// that is not the case-insensitive literal "false". A reference to an undefined
// variable surfaces as an error instead of being silently treated as a non-match.
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

// findFirstUndefinedDisabledExprVar returns the first {{name}} reference in expr that
// cannot be resolved through the standard variable sources, or "" when all references
// are resolvable. System variable references ($prefix) are skipped; their handling
// is delegated to the substitution engine.
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

// disabledExprVariableExists checks whether varName can be resolved through any of
// the standard variable sources (programmatic, file-scoped, environment, global,
// OS env, .env).
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

// fileScopedVarExists preserves the nil-map guard of the original lookup so the
// rewrite has identical semantics in every reachable path.
func fileScopedVarExists(m map[string]string, v string) bool {
	if m == nil {
		return false
	}
	_, ok := m["@"+v]
	return ok
}

// computeDisabledTruthiness applies the @disabled truthiness rules to a substituted
// string: case-insensitive "true" is truthy, case-insensitive "false" is falsy, a
// numeric value is truthy when non-zero, any other non-empty string is truthy, and
// the empty string is falsy.
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
