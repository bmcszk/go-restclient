package restclient_test

import (
	"testing"
	"time"
)

// `# @sleep 250` parses and stores 250ms duration.
func TestParseFile_SleepDirectiveStoresDuration(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### sleeping
# @sleep 250
GET https://example.com/api`).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestSleep(0, 250*time.Millisecond)
}

// `// @sleep 250` also parses (mirror of disabled).
func TestParseFile_SleepDirectiveSlashSlashStyle(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### sleeping
// @sleep 250
GET https://example.com/api`).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestSleep(0, 250*time.Millisecond)
}

// @sleep with non-integer/negative/missing arg must fail to parse; check parse failure + directive named in error.
func TestParseFile_SleepInvalidArgErrors(t *testing.T) {
	t.Run("non_numeric", func(t *testing.T) {
		given, when, then := newParts(t)
		given.
			aHttpFile(`### sleeping
# @sleep abc
GET https://example.com/api`).and().
			aClient()
		when.parsingFile()
		then.parseErrorContains("@sleep")
	})

	t.Run("negative", func(t *testing.T) {
		given, when, then := newParts(t)
		given.
			aHttpFile(`### sleeping
# @sleep -5
GET https://example.com/api`).and().
			aClient()
		when.parsingFile()
		then.parseErrorContains("@sleep")
	})

	t.Run("no_arg", func(t *testing.T) {
		given, when, then := newParts(t)
		given.
			aHttpFile(`### sleeping
# @sleep
GET https://example.com/api`).and().
			aClient()
		when.parsingFile()
		then.parseErrorContains("@sleep")
	})
}

// single request with `# @sleep 200` must take >= ~200ms before sending; measure elapsed with loose lower bound.
func TestExecuteFile_SleepWaitsBeforeSend(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### sleeping
# @sleep 200
GET {{server}}/slow`).and().
		aClient()

	start := time.Now()
	when.
		executeFile()
	elapsed := time.Since(start)

	then.
		noError().and().
		capturedRequestCount(1)
	// Loose lower bound (180ms) — accounts for timer slop and go-routine scheduling.
	then.assert.GreaterOrEqual(elapsed, 180*time.Millisecond,
		"@sleep 200 should delay send; elapsed=%s", elapsed)
}

// `@sleep 0` is a valid no-op — runs immediately, no error, 1 hit.
func TestExecuteFile_SleepZeroIsNoop(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### instant
# @sleep 0
GET {{server}}/instant`).and().
		aClient()

	start := time.Now()
	when.
		executeFile()
	elapsed := time.Since(start)

	then.
		noError().and().
		capturedRequestCount(1)
	// Loose upper bound (1s) — generous, just guards against accidentally introducing a delay.
	then.assert.Less(elapsed, time.Second,
		"@sleep 0 should be a no-op; elapsed=%s", elapsed)
}

// two requests, only second has `# @sleep 200`; inter-request gap between server-side timestamps >= ~200ms.
func TestExecuteFile_SleepAppliesPerRequest(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### first
GET {{server}}/first

### second
# @sleep 200
GET {{server}}/second`).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(2).and().
		gapBetweenCapturedRequestTimesAtLeast(0, 1, 180*time.Millisecond)
}
