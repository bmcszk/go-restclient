package restclient

import (
	"encoding/base64"
	"log/slog"
	"regexp"
)

// reBase64Encode matches {{$base64encode value}} placeholders.
var reBase64Encode = regexp.MustCompile(`{{\s*\$base64encode\s+(.+?)\s*}}`)

// substituteBase64Encode handles {{$base64encode value}} system variable.
func substituteBase64Encode(text string) string {
	return reBase64Encode.ReplaceAllStringFunc(text, func(match string) string {
		parts := reBase64Encode.FindStringSubmatch(match)
		if len(parts) == 2 {
			return base64.StdEncoding.EncodeToString([]byte(parts[1]))
		}
		return match
	})
}

// rePlaceholderWarn matches surviving {{placeholder}} tokens for the unresolved warning.
var rePlaceholderWarn = regexp.MustCompile(`{{\s*[^}]+\s*}}`)

// warnUnresolvedPlaceholders logs a warning for each {{placeholder}} that survived substitution.
func warnUnresolvedPlaceholders(text string) {
	matches := rePlaceholderWarn.FindAllString(text, -1)
	for _, m := range matches {
		slog.Warn("unresolved placeholder", "placeholder", m)
	}
}
