package restclient

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)


// multipartPart represents a parsed multipart form part
type multipartPart struct {
	Name            string
	Filename        string
	ContentType     string
	Content         string
	IsFileReference bool
}

// isMultipartFormWithFileReferences checks if the request is a multipart form containing file references
func (*Client) isMultipartFormWithFileReferences(restClientReq *Request) bool {
	contentType := restClientReq.Headers.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "multipart/form-data") {
		return false
	}
	
	// Check if the body contains file reference syntax (< filename)
	return strings.Contains(restClientReq.RawBody, "< ")
}

// processMultipartFormWithFiles processes multipart form data and replaces file references with actual file content
func (c *Client) processMultipartFormWithFiles(
	restClientReq *Request,
	parsedFile *ParsedFile,
	requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool),
) (string, error) {
	// First apply variable substitution to the raw body
	resolvedBody := resolveVariablesInText(
		restClientReq.RawBody,
		c.programmaticVars,
		restClientReq.ActiveVariables,
		parsedFile.EnvironmentVariables,
		parsedFile.GlobalVariables,
		requestScopedSystemVars,
		osEnvGetter,
		c.currentDotEnvVars,
	)
	
	processedBody := substituteDynamicSystemVariables(
		resolvedBody,
		c.currentDotEnvVars,
		c.programmaticVars,
	)
	
	// Parse and reconstruct the multipart form with file substitution
	result, err := c.reconstructMultipartFormWithFiles(processedBody, restClientReq)
	if err != nil {
		return "", err
	}
	return result, nil
}

// reconstructMultipartFormWithFiles parses multipart form data and reconstructs it with file content substitution
func (c *Client) reconstructMultipartFormWithFiles(body string, restClientReq *Request) (string, error) {
	boundary, formParts, err := c.parseMultipartFormData(body, restClientReq)
	if err != nil {
		return "", err
	}
	
	return c.buildMultipartForm(boundary, formParts, restClientReq.FilePath)
}

// parseMultipartFormData extracts boundary and parses form parts
func (c *Client) parseMultipartFormData(body string, restClientReq *Request) (string, []multipartPart, error) {
	contentType := restClientReq.Headers.Get("Content-Type")
	boundary := c.extractBoundaryFromContentType(contentType)
	if boundary == "" {
		return "", nil, fmt.Errorf("no boundary found in Content-Type header: %s", contentType)
	}
	
	formParts, err := c.parseMultipartBody(body, boundary)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse multipart body: %w", err)
	}
	
	return boundary, formParts, nil
}

// buildMultipartForm creates a new multipart form with file substitution
func (c *Client) buildMultipartForm(boundary string, formParts []multipartPart, filePath string) (string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	
	if err := writer.SetBoundary(boundary); err != nil {
		return "", fmt.Errorf("failed to set multipart boundary: %w", err)
	}
	
	for _, part := range formParts {
		if err := c.writePartToMultipart(writer, part, filePath); err != nil {
			return "", err
		}
	}
	
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}
	
	return buf.String(), nil
}

// writePartToMultipart writes a single part to the multipart writer
func (c *Client) writePartToMultipart(writer *multipart.Writer, part multipartPart, filePath string) error {
	if part.IsFileReference {
		if err := c.writeFilePartToMultipart(writer, part, filePath); err != nil {
			return fmt.Errorf("failed to write file part: %w", err)
		}
	} else {
		if err := c.writeFieldPartToMultipart(writer, part); err != nil {
			return fmt.Errorf("failed to write field part: %w", err)
		}
	}
	return nil
}

// extractBoundaryFromContentType extracts the boundary from a Content-Type header
func (*Client) extractBoundaryFromContentType(contentType string) string {
	// Look for boundary= in Content-Type
	re := regexp.MustCompile(`boundary=([^;]+)`)
	matches := re.FindStringSubmatch(contentType)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// parseMultipartBody parses a multipart body into individual parts
func (c *Client) parseMultipartBody(body, boundary string) ([]multipartPart, error) {
	var parts []multipartPart
	
	// Split by boundary
	boundaryDelimiter := "--" + boundary
	sections := strings.Split(body, boundaryDelimiter)
	
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" || section == "--" {
			continue
		}
		
		part, err := c.parseMultipartSection(section)
		if err != nil {
			continue // Skip malformed sections
		}
		
		parts = append(parts, part)
	}
	
	if len(parts) == 0 {
		return nil, errors.New("no valid multipart sections found in body")
	}
	
	return parts, nil
}

// parseMultipartSection parses a single multipart section
func (c *Client) parseMultipartSection(section string) (multipartPart, error) {
	var part multipartPart
	
	// Trim the section to remove leading/trailing whitespace
	section = strings.TrimSpace(section)
	
	headerLines, contentLines := c.splitSectionIntoHeadersAndContent(section)
	c.parseMultipartHeaders(&part, headerLines)
	c.parseMultipartContent(&part, contentLines)
	
	if part.Name == "" {
		return part, errors.New("no name found in multipart section")
	}
	return part, nil
}

// splitSectionIntoHeadersAndContent splits a multipart section into headers and content
func (c *Client) splitSectionIntoHeadersAndContent(section string) (headerLines []string, contentLines []string) {
	lines := strings.Split(section, "\n")
	
	contentStartIndex := c.findContentStartIndex(lines)
	return c.splitLinesAtIndex(lines, contentStartIndex)
}

// findContentStartIndex finds where content starts in multipart section lines
func (*Client) findContentStartIndex(lines []string) int {
	// Look for the first empty line (proper separator)
	if emptyLineIndex := findEmptyLineIndex(lines); emptyLineIndex != -1 {
		return emptyLineIndex + 1
	}
	
	// Use heuristic to separate headers from content
	return findContentStartByHeuristic(lines)
}

// splitLinesAtIndex splits lines into headers and content at the given index
func (*Client) splitLinesAtIndex(lines []string, contentStartIndex int) (headerLines []string, contentLines []string) {
	if contentStartIndex == -1 {
		return lines, nil // All lines are headers
	}
	
	headerLines = lines[:contentStartIndex]
	if contentStartIndex < len(lines) {
		contentLines = lines[contentStartIndex:]
	}
	return headerLines, contentLines
}

// findEmptyLineIndex finds the first empty line in the slice
func findEmptyLineIndex(lines []string) int {
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			return i
		}
	}
	return -1
}

// findContentStartByHeuristic finds content start using header detection heuristic
func findContentStartByHeuristic(lines []string) int {
	for i, line := range lines {
		if !isMultipartHeaderLine(line) {
			return i
		}
	}
	return -1
}

// isMultipartHeaderLine checks if a line looks like a multipart header
func isMultipartHeaderLine(line string) bool {
	if !strings.Contains(line, ":") {
		return false
	}
	
	trimmedLine := strings.TrimSpace(line)
	return strings.HasPrefix(trimmedLine, "Content-Disposition:") ||
		strings.HasPrefix(trimmedLine, "Content-Type:") ||
		strings.HasPrefix(trimmedLine, "Content-Length:") ||
		strings.HasPrefix(trimmedLine, "Content-Encoding:")
}

// parseMultipartHeaders parses the headers of a multipart section
func (c *Client) parseMultipartHeaders(part *multipartPart, headerLines []string) {
	for _, headerLine := range headerLines {
		if strings.Contains(headerLine, "Content-Disposition:") {
			part.Name = c.extractFormFieldName(headerLine)
			part.Filename = c.extractFilename(headerLine)
		} else if strings.Contains(headerLine, "Content-Type:") {
			part.ContentType = c.extractContentType(headerLine)
		}
	}
}

// parseMultipartContent parses the content of a multipart section
func (*Client) parseMultipartContent(part *multipartPart, contentLines []string) {
	content := strings.Join(contentLines, "\n")
	content = strings.TrimSpace(content)
	
	if strings.HasPrefix(content, "< ") {
		part.IsFileReference = true
		part.Content = strings.TrimSpace(content[2:]) // Remove "< "
	} else {
		part.IsFileReference = false
		part.Content = content
	}
}

// extractFormFieldName extracts the name from Content-Disposition header
func (*Client) extractFormFieldName(header string) string {
	re := regexp.MustCompile(`name="([^"]+)"`)
	matches := re.FindStringSubmatch(header)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// extractFilename extracts the filename from Content-Disposition header
func (*Client) extractFilename(header string) string {
	re := regexp.MustCompile(`filename="([^"]+)"`)
	matches := re.FindStringSubmatch(header)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// extractContentType extracts the content type from Content-Type header
func (*Client) extractContentType(header string) string {
	parts := strings.SplitN(header, ":", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

// writeFilePartToMultipart writes a file part to the multipart writer
func (c *Client) writeFilePartToMultipart(writer *multipart.Writer, part multipartPart, requestFilePath string) error {
	filePath := c.resolveFilePath(part.Content, requestFilePath)
	
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	
	formWriter, err := c.createMultipartFormWriter(writer, part)
	if err != nil {
		return fmt.Errorf("failed to create form field: %w", err)
	}
	
	if _, err = formWriter.Write(fileContent); err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}
	
	return nil
}

// resolveFilePath resolves a file path relative to request file or working directory
func (*Client) resolveFilePath(contentPath, requestFilePath string) string {
	if filepath.IsAbs(contentPath) {
		return contentPath
	}
	
	requestDir := filepath.Dir(requestFilePath)
	
	// If the request file is in a temporary directory, try resolving relative to cwd first
	if strings.Contains(requestDir, os.TempDir()) {
		if resolvedPath := tryResolveFromCwd(contentPath); resolvedPath != "" {
			return resolvedPath
		}
	}
	
	// Fallback to request directory
	return filepath.Join(requestDir, contentPath)
}

// tryResolveFromCwd attempts to resolve a path relative to current working directory
func tryResolveFromCwd(contentPath string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	
	cwdPath := filepath.Join(cwd, contentPath)
	if _, err := os.Stat(cwdPath); err == nil {
		return cwdPath
	}
	
	return ""
}

// createMultipartFormWriter creates the appropriate form writer based on part characteristics
func (*Client) createMultipartFormWriter(writer *multipart.Writer, part multipartPart) (io.Writer, error) {
	if part.Filename != "" {
		return createFilePartWithFilename(writer, part)
	}
	
	if part.ContentType != "" {
		return createFilePartWithoutFilename(writer, part)
	}
	
	return writer.CreateFormField(part.Name)
}

// createFilePartWithFilename creates a multipart file part with explicit filename
func createFilePartWithFilename(writer *multipart.Writer, part multipartPart) (io.Writer, error) {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, part.Name, part.Filename))
	if part.ContentType != "" {
		header.Set("Content-Type", part.ContentType)
	}
	return writer.CreatePart(header)
}

// createFilePartWithoutFilename creates a multipart file part with inferred filename
func createFilePartWithoutFilename(writer *multipart.Writer, part multipartPart) (io.Writer, error) {
	filename := filepath.Base(part.Content)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, part.Name, filename))
	header.Set("Content-Type", part.ContentType)
	return writer.CreatePart(header)
}

// writeFieldPartToMultipart writes a regular form field to the multipart writer
func (*Client) writeFieldPartToMultipart(writer *multipart.Writer, part multipartPart) error {
	formWriter, err := writer.CreateFormField(part.Name)
	if err != nil {
		return fmt.Errorf("failed to create form field: %w", err)
	}
	
	_, err = formWriter.Write([]byte(part.Content))
	if err != nil {
		return fmt.Errorf("failed to write field content: %w", err)
	}
	
	return nil
}