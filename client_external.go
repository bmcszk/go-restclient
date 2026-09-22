package restclient

// External file references and character-encoding handling for the client.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

// processExternalFile reads and processes external file references with optional variable substitution and encoding
func (c *Client) processExternalFile(
	restClientReq *Request,
	parsedFile *ParsedFile,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
) (string, error) {
	// Resolve the file path relative to the request's file directory
	requestDir := filepath.Dir(restClientReq.FilePath)
	fullPath := restClientReq.ExternalFilePath
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(requestDir, restClientReq.ExternalFilePath)
	}

	// Read the file with appropriate encoding
	content, err := c.readFileWithEncoding(fullPath, restClientReq.ExternalFileEncoding)
	if err != nil {
		return "", fmt.Errorf("failed to read external file %s: %w", restClientReq.ExternalFilePath, err)
	}

	// Apply variable substitution if requested
	if restClientReq.ExternalFileWithVariables {
		loopAliases, loopIndex := currentLoopBindings(restClientReq)
		rctx := c.directiveResolveContext(parsedFile, restClientReq, osEnvGetter,
			requestScopedSystemVars, loopAliases, loopIndex)
		resolvedContent := resolveVariablesInText(content, rctx)
		content = substituteDynamicSystemVariables(
			resolvedContent,
			c.currentDotEnvVars,
			c.programmaticVars,
		)
	}

	return content, nil
}

// readFileWithEncoding reads a file with the specified encoding, defaulting to UTF-8
func (c *Client) readFileWithEncoding(filePath, encodingName string) (string, error) {
	// Read the file as bytes
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// If no encoding specified or UTF-8, return as-is
	if encodingName == "" || strings.ToLower(encodingName) == "utf-8" || strings.ToLower(encodingName) == "utf8" {
		return string(data), nil
	}

	// Get the decoder for the specified encoding
	decoder, err := c.getEncodingDecoder(encodingName)
	if err != nil {
		return "", fmt.Errorf("unsupported encoding %s: %w", encodingName, err)
	}

	// Decode the content
	decodedContent, err := decoder.Bytes(data)
	if err != nil {
		return "", fmt.Errorf("failed to decode content with encoding %s: %w", encodingName, err)
	}

	return string(decodedContent), nil
}

// getEncodingDecoder returns the appropriate decoder for the given encoding name
func (*Client) getEncodingDecoder(encodingName string) (*encoding.Decoder, error) {
	encodingName = strings.ToLower(encodingName)

	switch encodingName {
	case "latin1", "iso-8859-1":
		return charmap.ISO8859_1.NewDecoder(), nil
	case "cp1252", "windows-1252":
		return charmap.Windows1252.NewDecoder(), nil
	case "ascii":
		// ASCII is a subset of UTF-8, so we can use UTF-8 decoder
		return unicode.UTF8.NewDecoder(), nil
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", encodingName)
	}
}
