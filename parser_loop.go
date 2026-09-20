package restclient

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// handleLoopDirective parses `# @loop for <rest>` into LoopFor/LoopExpr/LoopCollection/LoopItemName.
// Exactly one of LoopFor, LoopExpr, LoopCollection is set; LoopItemName is set only for `name of coll`.
func (p *requestParserState) handleLoopDirective(commentContent string) (bool, error) {
	const prefix = "@loop"
	if !strings.HasPrefix(commentContent, prefix) {
		return false, nil
	}
	arg, err := extractLoopArg(commentContent[len(prefix):])
	if err != nil {
		return true, err
	}
	return p.parseLoopArg(arg)
}

// extractLoopArg strips the `for ` clause and the surrounding whitespace; returns an error when
// the directive is missing the required `for` clause or argument.
func extractLoopArg(rest string) (string, error) {
	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, "for ") {
		return "", errors.New("@loop directive requires a `for` clause (got: " + rest + ")")
	}
	arg := strings.TrimSpace(rest[len("for "):])
	if arg == "" {
		return "", errors.New("@loop directive requires an argument after `for`")
	}
	return arg, nil
}

// parseLoopArg dispatches the @loop argument to one of the three supported forms.
func (p *requestParserState) parseLoopArg(arg string) (bool, error) {
	if isLoopPlaceholder(arg) {
		return p.setLoopExpr(arg)
	}
	if idx := strings.Index(arg, " of "); idx > 0 {
		return p.setLoopCollection(arg, idx)
	}
	return p.setLoopCount(arg)
}

// isLoopPlaceholder reports whether arg is a `{{name}}` style placeholder.
func isLoopPlaceholder(arg string) bool {
	return strings.HasPrefix(arg, "{{") && strings.HasSuffix(arg, "}}")
}

// setLoopExpr stores the `{{var}}` form on the current request.
func (p *requestParserState) setLoopExpr(arg string) (bool, error) {
	inner := strings.TrimSpace(arg[2 : len(arg)-2])
	if inner == "" {
		return true, errors.New("@loop expression placeholder is empty")
	}
	p.currentRequest.LoopExpr = arg
	p.currentRequest.LoopDeclared = true
	return true, nil
}

// setLoopCollection stores the `<item> of <coll>` form on the current request.
func (p *requestParserState) setLoopCollection(arg string, idx int) (bool, error) {
	itemName := strings.TrimSpace(arg[:idx])
	collName := strings.TrimSpace(arg[idx+len(" of "):])
	if itemName == "" || collName == "" {
		return true, errors.New("@loop `of` form requires both item name and collection variable")
	}
	p.currentRequest.LoopItemName = itemName
	p.currentRequest.LoopCollection = collName
	p.currentRequest.LoopDeclared = true
	return true, nil
}

// setLoopCount stores the integer literal form on the current request.
func (p *requestParserState) setLoopCount(arg string) (bool, error) {
	n, err := strconv.Atoi(arg)
	if err != nil {
		return true, errors.New("@loop directive expects integer, {{var}} or `name of collection` (got: " + arg + ")")
	}
	p.currentRequest.LoopFor = n
	p.currentRequest.LoopDeclared = true
	return true, nil
}

// snapshotLoopState snapshots pre-substitution body/headers so loop iterations can be
// re-substituted from originals instead of last iteration's output.
func (*requestParserState) snapshotLoopState(req *Request) {
	if req.LoopDeclared {
		req.loopOriginalRawBody = req.RawBody
		snapshot := make(http.Header, len(req.Headers))
		for k, vs := range req.Headers {
			snapshot[k] = append([]string(nil), vs...)
		}
		req.loopOriginalHeaders = snapshot
	}
}
