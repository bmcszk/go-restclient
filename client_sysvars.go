package restclient

// Request- and file-scoped system variable generation and placeholder resolution.

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// substituteDynamicSystemVariables handles system variables that require argument
// parsing or dynamic evaluation at substitution time.

// generateRequestScopedSystemVariables creates a map of system variables that are generated once per request.
// This ensures that if, for example, {{$uuid}} is used multiple times within the same request
// (e.g., in the URL and a header), it resolves to the same value for that specific request.
func (*Client) generateRequestScopedSystemVariables() map[string]string {
	vars := make(map[string]string)
	vars["$uuid"] = uuid.NewString()
	vars["$guid"] = vars["$uuid"]        // Alias $guid to $uuid
	vars["$random.uuid"] = vars["$uuid"] // Add $random.uuid as alias
	vars["$timestamp"] = strconv.FormatInt(time.Now().UTC().Unix(), 10)
	vars["$isoTimestamp"] = time.Now().UTC().Format(time.RFC3339) // Add $isoTimestamp
	vars["$randomInt"] = strconv.Itoa(rand.Intn(1001))            // 0-1000 inclusive as per PRD
	// Add other simple, no-argument system variables here if any

	return vars
}

// resolveFileScopedSystemVariables resolves file-scoped variables that contain system variable placeholders.
// This ensures that system variables in file-scoped variable definitions (like @scenarioId = {{$uuid}})
// are resolved once per file, not once per request, maintaining consistency across all requests in the file.
func (c *Client) resolveFileScopedSystemVariables(parsedFile *ParsedFile) {
	if parsedFile == nil || parsedFile.FileVariables == nil {
		return
	}

	// Generate file-scoped system variables once for the entire file
	fileScopedSystemVars := c.generateRequestScopedSystemVariables()

	// Resolve file-scoped variables and track resolved ones
	resolvedVariables := c.resolveFileVariables(parsedFile, fileScopedSystemVars)

	// Update all requests' ActiveVariables to reflect the resolved values
	c.updateRequestActiveVariables(parsedFile.Requests, resolvedVariables)
}

// resolveFileVariables processes each file-scoped variable that contains system variable placeholders
func (c *Client) resolveFileVariables(
	parsedFile *ParsedFile,
	fileScopedSystemVars map[string]string,
) map[string]string {
	resolvedVariables := make(map[string]string)

	for varName, varValue := range parsedFile.FileVariables {
		if isSystemVariablePlaceholder(varValue) {
			resolvedValue := resolveSystemVariablePlaceholder(
				varValue, fileScopedSystemVars, c.currentDotEnvVars, c.programmaticVars)
			parsedFile.FileVariables[varName] = resolvedValue
			resolvedVariables[varName] = resolvedValue
		}
	}

	return resolvedVariables
}

// updateRequestActiveVariables updates all requests' ActiveVariables with resolved values
func (c *Client) updateRequestActiveVariables(requests []*Request, resolvedVariables map[string]string) {
	for _, request := range requests {
		c.updateSingleRequestActiveVariables(request, resolvedVariables)
	}
}

// updateSingleRequestActiveVariables updates a single request's ActiveVariables with resolved values
func (*Client) updateSingleRequestActiveVariables(request *Request, resolvedVariables map[string]string) {
	if request.ActiveVariables == nil {
		return
	}

	for varName, resolvedValue := range resolvedVariables {
		if _, exists := request.ActiveVariables[varName]; exists {
			request.ActiveVariables[varName] = resolvedValue
		}
	}
}

// isSystemVariablePlaceholder checks if a string contains a system variable placeholder
func isSystemVariablePlaceholder(value string) bool {
	if !strings.HasPrefix(value, "{{") || !strings.HasSuffix(value, "}}") {
		return false
	}

	innerDirective := strings.TrimSpace(value[2 : len(value)-2])
	return strings.HasPrefix(innerDirective, "$")
}

// resolveSystemVariablePlaceholder resolves a system variable placeholder to its value
func resolveSystemVariablePlaceholder(
	placeholder string,
	systemVars map[string]string,
	dotEnvVars map[string]string,
	programmaticVars map[string]any,
) string {
	innerDirective := strings.TrimSpace(placeholder[2 : len(placeholder)-2])

	// Check if it's a simple system variable that we have pre-generated
	if val, ok := systemVars[innerDirective]; ok {
		return val
	}

	// For dynamic system variables, use the existing substitution logic
	return substituteDynamicSystemVariables(placeholder, dotEnvVars, programmaticVars)
}
