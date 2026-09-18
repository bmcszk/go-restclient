// Code in this file is migrated from test/client_execute_system_vars.go (RunExecuteFile_WithGuidSystemVariable,
// RunExecuteFile_WithIsoTimestampSystemVariable, RunExecuteFile_WithDatetimeSystemVariables,
// RunExecuteFile_WithTimestampSystemVariable, RunExecuteFile_WithRandomIntSystemVariable,
// RunExecuteFile_WithFakerPersonData, RunExecuteFile_WithContactAndInternetFakerData).
package restclient_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestExecuteFile_WithGuidSystemVariable: System Variables {{$guid}} and {{$uuid}}.
func TestExecuteFile_WithGuidSystemVariable(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aHttpFileFromTemplate("system_var_guid.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		tracking("guid").and().
		capturedURLSegment(0).and().
		capturedHeaderField(0, "X-Request-ID").and().
		capturedJSONField(0, "transactionId").and().
		capturedJSONField(0, "correlationId").and().
		capturedJSONField(0, "randomUuidAlias").and().
		allTrackedValuesAreValidUUIDs().and().
		allTrackedValuesEqual()
}

// TestExecuteFile_WithIsoTimestampSystemVariable: System Variables {{$isoTimestamp}}.
func TestExecuteFile_WithIsoTimestampSystemVariable(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).and().
		aHttpFileFromTemplate("system_var_iso_timestamp.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		tracking("iso").and().
		capturedHeaderField(0, "X-Timestamp-Header").and().
		capturedJSONField(0, "requestTime").and().
		allTrackedValuesEqual().and().
		allTrackedValuesAreValidRFC3339Timestamps()
}

// TestExecuteFile_WithDatetimeSystemVariables: System Variables {{$datetime "format"}}.
//
// Datetime values use 6 tracked buckets split by layout + zone, with
// allTrackedDatetimeValuesAreWithin / allTrackedDatetimeValuesHaveUTCZone /
// allTrackedDatetimeValuesHaveLocalZone for the per-bucket assertions. The
// "now" reference time is captured as a local var (parity-preserving).
func TestExecuteFile_WithDatetimeSystemVariables(t *testing.T) {
	given, when, then := newParts(t)
	now := time.Now()

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aHttpFileFromTemplate("system_var_datetime.http").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		tracking("rfc1123-utc").and().
		capturedHeaderField(0, "X-Datetime-RFC1123").and().
		capturedJSONField(0, "utc_rfc1123").and().
		allTrackedDatetimeValuesAreWithin(5*time.Second, time.RFC1123).and().
		allTrackedDatetimeValuesHaveUTCZone().and().
		tracking("iso8601-utc").and().
		capturedHeaderField(0, "X-Datetime-ISO8601").and().
		capturedHeaderField(0, "X-Datetime-Default").and().
		capturedJSONField(0, "utc_iso8601").and().
		capturedJSONField(0, "utc_default_iso").and().
		allTrackedDatetimeValuesAreWithin(5*time.Second, time.RFC3339).and().
		allTrackedDatetimeValuesHaveUTCZone().and().
		tracking("timestamp-utc").and().
		capturedHeaderField(0, "X-Datetime-Timestamp").and().
		capturedJSONField(0, "utc_timestamp").and().
		allTrackedDatetimeValuesAreWithin(5*time.Second, "timestamp").and().
		allTrackedDatetimeValuesHaveUTCZone().and().
		tracking("rfc1123-local").and().
		capturedHeaderField(0, "X-LocalDatetime-RFC1123").and().
		capturedJSONField(0, "local_rfc1123").and().
		allTrackedDatetimeValuesAreWithin(5*time.Second, time.RFC1123).and().
		allTrackedDatetimeValuesHaveLocalZone().and().
		tracking("iso8601-local").and().
		capturedHeaderField(0, "X-LocalDatetime-ISO8601").and().
		capturedHeaderField(0, "X-LocalDatetime-Default").and().
		capturedJSONField(0, "local_iso8601").and().
		capturedJSONField(0, "local_default_iso").and().
		allTrackedDatetimeValuesAreWithin(5*time.Second, time.RFC3339).and().
		allTrackedDatetimeValuesHaveLocalZone().and().
		tracking("timestamp-local").and().
		capturedHeaderField(0, "X-LocalDatetime-Timestamp").and().
		capturedJSONField(0, "local_timestamp").and().
		allTrackedDatetimeValuesAreWithin(5*time.Second, "timestamp").and().
		allTrackedDatetimeValuesHaveLocalZone().and().
		serverReceivedHeaderValue(0, "X-Datetime-Invalid", `{{$datetime "invalidFormat"}}`).and().
		capturedJSONFieldIs(0, `{{$datetime "invalidFormat"}}`, "invalid_format_test")

	// "now" was captured before ExecuteFile; suppress unused-var warning.
	_ = now
}

// TestExecuteFile_WithTimestampSystemVariable: System Variables {{$timestamp}}.
func TestExecuteFile_WithTimestampSystemVariable(t *testing.T) {
	runTimestampConsistencyAssertions(t)
}

// TestExecuteFile_WithRandomIntSystemVariable: System Variables {{$randomInt [MIN MAX]}}.
//
// Four subtests with exact original names. Fresh parts per subtest keep the
// captured state isolated.
func TestExecuteFile_WithRandomIntSystemVariable(t *testing.T) {
	t.Run("valid min max args", func(t *testing.T) {
		runRandomIntNumericAssertions(t, "system_var_randomint_valid_args.http", 10, 20, 1, 5, 100, 105)
	})

	t.Run("no args", func(t *testing.T) {
		runRandomIntNumericAssertions(t, "system_var_randomint_no_args.http", 0, 1000, 0, 1000, 0, 1000)
	})

	t.Run("swapped min max args", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "ok")
			}).and().
			aHttpFileFromTemplate("system_var_randomint_swapped_args.http").and().
			aClient()

		when.
			executeFile()

		then.
			responseCount(1).and().
			noError().and().
			responseAt(0).and().
			responseHasNoError().and().
			responseCode(http.StatusOK).and().
			requestRawURLContains("/rint/{{$randomInt 30 25}}/{{$randomInt 30 25}}").and().
			serverReceivedHeaderValue(0, "X-Random-ID", "{{$randomInt 30 25}}").and().
			capturedJSONStringMapIs(0, map[string]string{
				"value": "{{$randomInt 30 25}}",
			})
	})

	t.Run("malformed args", func(t *testing.T) {
		given, when, then := newParts(t)

		given.
			aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "ok")
			}).and().
			aHttpFileFromTemplate("system_var_randomint_malformed_args.http").and().
			aClient()

		when.
			executeFile()

		then.
			responseCount(1).and().
			noError().and().
			responseAt(0).and().
			responseHasNoError().and().
			responseCode(http.StatusOK).and().
			requestRawURLContains("{{$randomInt abc def}}").and().
			serverReceivedHeaderValue(0, "X-Random-ID", "{{$randomInt 1 xyz}}").and().
			capturedJSONStringMapIs(0, map[string]string{
				"value": "{{$randomInt foo bar}}",
			})
	})
}

// runRandomIntNumericAssertions is the shared body for the numeric subtests
// ("valid min max args" / "no args") of TestExecuteFile_WithRandomIntSystemVariable.
func runRandomIntNumericAssertions(t *testing.T, fixture string,
	urlLow, urlHigh, headerLow, headerHigh, bodyLow, bodyHigh int64) {
	t.Helper()
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aHttpFileFromTemplate(fixture).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		tracking("url").and().
		capturedURLSegmentAt(0, 2).and().
		firstTrackedValueIsIntegerInRange(urlLow, urlHigh).and().
		tracking("header").and().
		capturedHeaderField(0, "X-Random-ID").and().
		firstTrackedValueIsIntegerInRange(headerLow, headerHigh).and().
		tracking("body").and().
		capturedJSONNumberField(0, "value").and().
		firstTrackedValueIsIntegerInRange(bodyLow, bodyHigh)
}

// fakerHeaderRule describes the per-request header validation to perform for the faker
// tests. fieldCount == -1 means "do not check field count". containsCheck is non-empty when
// the header value must contain that fragment (instead of matching a regex).
type fakerHeaderRule struct {
	key           string
	pattern       string
	fieldCount    int
	containsCheck string
}

// assertFakerHeaders performs the standard set of header checks (regex + NotContains "{{" +
// optional field-count or contains check) for every rule in the slice, attached to the
// current then chain.
func (p *parts) assertFakerHeaders(reqIndex int, rules []fakerHeaderRule) *parts {
	for _, rule := range rules {
		if rule.containsCheck != "" {
			p = p.serverReceivedHeaderContains(reqIndex, rule.key, rule.containsCheck).and()
			p = p.serverReceivedHeaderNotContains(reqIndex, rule.key, "{{").and()
			continue
		}

		p = p.serverReceivedHeaderMatchesRegexp(reqIndex, rule.key, rule.pattern).and()
		p = p.serverReceivedHeaderNotContains(reqIndex, rule.key, "{{").and()
		if rule.fieldCount > 0 {
			p = p.serverReceivedHeaderFieldCountIs(reqIndex, rule.key, rule.fieldCount).and()
		}
	}

	return p
}

// fakerPersonDataHeaders lists the per-fixture header validation rules for the two
// faker_person_data.http requests.
func fakerPersonDataHeaders() (vsCode, jetBrains []fakerHeaderRule) {
	vsCode = []fakerHeaderRule{
		{key: "X-Random-First-Name", pattern: `^\S+$`, fieldCount: 1},
		{key: "X-Random-Last-Name", pattern: `^\S+$`, fieldCount: 1},
		{key: "X-Random-Full-Name", pattern: `^\S+ \S+$`, fieldCount: 2},
		{key: "X-Random-Job-Title", pattern: `^.+$`, fieldCount: -1},
	}
	jetBrains = []fakerHeaderRule{
		{key: "X-Random-First-Name-Dot", pattern: `^\S+$`, fieldCount: 1},
		{key: "X-Random-Last-Name-Dot", pattern: `^\S+$`, fieldCount: 1},
		{key: "X-Random-Full-Name-Dot", pattern: `^\S+ \S+$`, fieldCount: 2},
		{key: "X-Random-Job-Title-Dot", pattern: `^.+$`, fieldCount: -1},
	}
	return vsCode, jetBrains
}

// fakerContactInternetHeaders lists the per-fixture header validation rules for the two
// faker_contact_internet_data.http requests.
func fakerContactInternetHeaders() (vsCode, jetBrains []fakerHeaderRule) {
	vsCode = []fakerHeaderRule{
		{key: "X-Phone", pattern: `^\(\d{3}\) \d{3}-\d{4}$`, fieldCount: -1},
		{key: "X-Address", pattern: `^\d+ .+`, fieldCount: -1},
		{key: "X-City", pattern: `^.+$`, fieldCount: -1},
		{key: "X-State", pattern: `^.+$`, fieldCount: -1},
		{key: "X-Zip", pattern: `^\d{5}$`, fieldCount: -1},
		{key: "X-Country", pattern: `^.+$`, fieldCount: -1},
		{key: "X-Url", pattern: `^https?://`, fieldCount: -1},
		{key: "X-Domain", pattern: `^.+\..+$`, fieldCount: -1},
		{key: "X-User-Agent", containsCheck: "Mozilla"},
		{key: "X-Mac", pattern: `^([0-9a-f]{2}:){5}[0-9a-f]{2}$`, fieldCount: -1},
	}
	jetBrains = []fakerHeaderRule{
		{key: "X-Phone-Dot", pattern: `^\(\d{3}\) \d{3}-\d{4}$`, fieldCount: -1},
		{key: "X-Address-Dot", pattern: `^.+$`, fieldCount: -1},
		{key: "X-City-Dot", pattern: `^.+$`, fieldCount: -1},
		{key: "X-Url-Dot", pattern: `^https?://`, fieldCount: -1},
		{key: "X-Mac-Dot", pattern: `^([0-9a-f]{2}:){5}[0-9a-f]{2}$`, fieldCount: -1},
	}
	return vsCode, jetBrains
}

// TestExecuteFile_WithFakerPersonData: Faker Library Support - Person/Identity Data.
//
// Two requests in the fixture. Each request's headers are validated against
// regex (NotEmpty) + NotContains "{{" + (for first/last/full) field-count.
func TestExecuteFile_WithFakerPersonData(t *testing.T) {
	given, when, then := newParts(t)
	vsCode, jetBrains := fakerPersonDataHeaders()

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aTemplateFixture("system_variables", "faker_person_data.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		capturedRequestCount(2).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responseAt(1).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		assertFakerHeaders(0, vsCode).and().
		assertFakerHeaders(1, jetBrains)
}

// TestExecuteFile_WithContactAndInternetFakerData: Faker Library - Contact and Internet Data.
func TestExecuteFile_WithContactAndInternetFakerData(t *testing.T) {
	given, when, then := newParts(t)
	vsCode, jetBrains := fakerContactInternetHeaders()

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aTemplateFixture("system_variables", "faker_contact_internet_data.http",
			struct{ ServerURL string }{ServerURL: given.serverURL}).and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(2).and().
		capturedRequestCount(2).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		responseAt(1).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		assertFakerHeaders(0, vsCode).and().
		assertFakerHeaders(1, jetBrains).and().
		// JSON bodies for both requests: contact.phone/address.street/technical.website
		capturedJSONFieldNotContains(0, "{{", "contact", "phone").and().
		capturedJSONFieldNotContains(0, "{{", "contact", "address", "street").and().
		capturedJSONFieldNotContains(0, "{{", "technical", "website").and().
		capturedJSONFieldMatchesRegexp(0, `^https?://`, "technical", "website").and().
		capturedJSONFieldNotContains(1, "{{", "contact", "phone").and().
		capturedJSONFieldNotContains(1, "{{", "contact", "address", "street").and().
		capturedJSONFieldNotContains(1, "{{", "technical", "website").and().
		capturedJSONFieldMatchesRegexp(1, `^https?://`, "technical", "website")
}
