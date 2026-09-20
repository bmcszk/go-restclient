package restclient

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// handleSleepDirective parses `@sleep <ms>`; non-negative integer ms stored in SleepDuration.
func (p *requestParserState) handleSleepDirective(commentContent string) (bool, error) {
	if !strings.HasPrefix(commentContent, "@sleep") {
		return false, nil
	}
	ms, err := strconv.Atoi(strings.TrimSpace(commentContent[len("@sleep"):]))
	if err != nil || ms < 0 {
		return true, errors.New("@sleep directive requires a non-negative integer milliseconds argument")
	}
	p.currentRequest.SleepDuration = time.Duration(ms) * time.Millisecond
	return true, nil
}
