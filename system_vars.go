package restclient

import (
	"encoding/base64"
	"log/slog"
	"regexp"
)

// substituteBase64Encode handles {{$base64encode value}} system variable.
func substituteBase64Encode(text string) string {
	re := regexp.MustCompile(`{{\s*\$base64encode\s+(.+?)\s*}}`)
	return re.ReplaceAllStringFunc(text, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) == 2 {
			return base64.StdEncoding.EncodeToString([]byte(parts[1]))
		}
		return match
	})
}

// warnUnresolvedPlaceholders logs a warning for each {{placeholder}} that survived substitution.
func warnUnresolvedPlaceholders(text string) {
	re := regexp.MustCompile(`{{\s*[^}]+\s*}}`)
	matches := re.FindAllString(text, -1)
	for _, m := range matches {
		slog.Warn("unresolved placeholder", "placeholder", m)
	}
}
