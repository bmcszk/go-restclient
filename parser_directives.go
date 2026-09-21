package restclient

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// processCommentDirectives processes various comment directives
func (p *requestParserState) processCommentDirectives(commentContent string) error {
	boolDirectives := []func(string) bool{
		p.handleNameDirective,
		p.handleNoRedirectDirective,
		p.handleNoCookieJarDirective,
		p.handleTimeoutDirective,
	}
	for _, handle := range boolDirectives {
		if handle(commentContent) {
			return nil
		}
	}
	errDirectives := []func(string) (bool, error){
		p.handleDisabledDirective,
		p.handleSleepDirective,
		p.handleLoopDirective,
		p.handleImportDirective,
	}
	for _, handle := range errDirectives {
		if handled, err := handle(commentContent); handled {
			return err
		}
	}
	return p.handleRefDirective(commentContent) // Other comment content - no special handling needed
}

// handleNameDirective processes @name directives
func (p *requestParserState) handleNameDirective(commentContent string) bool {
	parsedName, isNameDirective := parseNameFromAtNameDirective(commentContent)
	if isNameDirective && parsedName != "" {
		p.currentRequest.Name = parsedName
	}
	return isNameDirective
}

// handleNoRedirectDirective processes @no-redirect directives
func (p *requestParserState) handleNoRedirectDirective(commentContent string) bool {
	if strings.HasPrefix(commentContent, "@no-redirect") {
		p.currentRequest.NoRedirect = true
		return true
	}
	return false
}

// handleNoCookieJarDirective processes @no-cookie-jar directives
func (p *requestParserState) handleNoCookieJarDirective(commentContent string) bool {
	if strings.HasPrefix(commentContent, "@no-cookie-jar") {
		p.currentRequest.NoCookieJar = true
		return true
	}
	return false
}

// handleDisabledDirective parses `@disabled`; `@disabled !<expr>` stores a conditional expression.
func (p *requestParserState) handleDisabledDirective(commentContent string) (bool, error) {
	const prefix = "@disabled"
	if !strings.HasPrefix(commentContent, prefix) {
		return false, nil
	}
	rest := strings.TrimSpace(commentContent[len(prefix):])
	if rest == "" {
		p.currentRequest.Disabled = true
		return true, nil
	}
	if !strings.HasPrefix(rest, "!") {
		return true, fmt.Errorf("@disabled directive takes no arguments (got %q)", rest)
	}
	p.currentRequest.DisabledExpr = strings.TrimSpace(rest[1:])

	return true, nil
}

// handleTimeoutDirective processes @timeout directives
func (p *requestParserState) handleTimeoutDirective(commentContent string) bool {
	if strings.HasPrefix(commentContent, "@timeout ") {
		p.processTimeoutDirective(commentContent)
		return true
	}
	return false
}

// handleRefDirective processes @ref and @forceRef directives.
func (p *requestParserState) handleRefDirective(commentContent string) error {
	if strings.HasPrefix(commentContent, "@forceRef ") {
		return p.appendRef("@forceRef", commentContent[len("@forceRef "):])
	}
	if strings.HasPrefix(commentContent, "@ref ") {
		return p.appendRef("@ref", commentContent[len("@ref "):])
	}
	return nil
}

// handleImportDirective parses `# @import <path>`, merges its vars/requests.
func (p *requestParserState) handleImportDirective(commentContent string) (bool, error) {
	const prefix = "@import"
	if !strings.HasPrefix(commentContent, prefix) {
		return false, nil
	}
	raw := strings.TrimSpace(commentContent[len(prefix):])
	if raw == "" {
		return true, errors.New("@import directive requires a path")
	}
	importedPath := filepath.Join(filepath.Dir(p.filePath), raw)
	imported, err := parseRequestFile(importedPath, p.client, p.importStack)
	if err != nil {
		return true, fmt.Errorf("@import %s: %w", raw, err)
	}
	for k, v := range imported.FileVariables {
		if _, already := p.currentFileVariables[k]; already {
			continue
		}
		p.currentFileVariables[k] = v
	}
	p.importedParsedFiles = append(p.importedParsedFiles, imported)
	return true, nil
}

func (p *requestParserState) appendRef(directive, raw string) error {
	name := strings.TrimSpace(raw)
	if name == "" {
		return fmt.Errorf("missing reference name in %s directive", directive)
	}
	p.currentRequest.Refs = append(p.currentRequest.Refs, RequestRef{Name: name, Force: directive == "@forceRef"})
	return nil
}

// processTimeoutDirective handles the @timeout directive with milliseconds value
func (p *requestParserState) processTimeoutDirective(commentContent string) {
	p.ensureCurrentRequest()
	timeoutStr := strings.TrimSpace(commentContent[len("@timeout "):])
	if timeoutStr == "" {
		return
	}

	timeoutMs, err := strconv.Atoi(timeoutStr)
	if err != nil || timeoutMs <= 0 {
		slog.Warn("Invalid timeout value in @timeout directive",
			"value", timeoutStr,
			"lineNumber", p.lineNumber,
			"filePath", p.filePath)
		return
	}

	p.currentRequest.Timeout = time.Duration(timeoutMs) * time.Millisecond
}
