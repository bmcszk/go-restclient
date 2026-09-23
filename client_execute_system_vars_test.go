// RunExecuteFile_WithIsoTimestampSystemVariable, RunExecuteFile_WithDatetimeSystemVariables,
// RunExecuteFile_WithTimestampSystemVariable, RunExecuteFile_WithRandomIntSystemVariable,
// RunExecuteFile_WithFakerPersonData, RunExecuteFile_WithContactAndInternetFakerData).
package restclient_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	rc "github.com/bmcszk/go-restclient"
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
func TestExecuteFile_WithDatetimeSystemVariables(t *testing.T) {
	given, when, then := newParts(t)

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
}

// TestExecuteFile_WithTimestampSystemVariable: System Variables {{$timestamp}}.
func TestExecuteFile_WithTimestampSystemVariable(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aHttpFileFromTemplate("system_var_timestamp.http").and().
		aClient()

	beforeSec := time.Now().UTC().Unix()
	when.
		executeFile()
	afterSec := time.Now().UTC().Unix()

	then.
		responseCount(1).and().
		noError().and().
		responseAt(0).and().
		responseHasNoError().and().
		responseCode(http.StatusOK).and().
		tracking("timestamp").and().
		capturedURLSegment(0).and().
		capturedHeaderField(0, "X-Request-Time").and().
		capturedJSONField(0, "event_time").and().
		capturedJSONField(0, "processed_at").and().
		allTrackedValuesEqual().and().
		firstTrackedValueIsIntegerInRange(beforeSec, afterSec).and().
		allTrackedValuesArePositiveIntegers()
}

// TestExecuteFile_WithRandomIntSystemVariable_ValidMinMaxArgs: System Variables {{$randomInt [MIN MAX]}}.
func TestExecuteFile_WithRandomIntSystemVariable_ValidMinMaxArgs(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aHttpFileFromTemplate("system_var_randomint_valid_args.http").and().
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
		firstTrackedValueIsIntegerInRange(10, 20).and().
		tracking("header").and().
		capturedHeaderField(0, "X-Random-ID").and().
		firstTrackedValueIsIntegerInRange(1, 5).and().
		tracking("body").and().
		capturedJSONNumberField(0, "value").and().
		firstTrackedValueIsIntegerInRange(100, 105)
}

// TestExecuteFile_WithRandomIntSystemVariable_NoArgs: System Variables {{$randomInt}} with no args.
func TestExecuteFile_WithRandomIntSystemVariable_NoArgs(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok")
		}).and().
		aHttpFileFromTemplate("system_var_randomint_no_args.http").and().
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
		firstTrackedValueIsIntegerInRange(0, 1000).and().
		tracking("header").and().
		capturedHeaderField(0, "X-Random-ID").and().
		firstTrackedValueIsIntegerInRange(0, 1000).and().
		tracking("body").and().
		capturedJSONNumberField(0, "value").and().
		firstTrackedValueIsIntegerInRange(0, 1000)
}

// TestExecuteFile_WithRandomIntSystemVariable_SwappedMinMaxArgs: System Variables {{$randomInt [MIN MAX]}}.
// URL / header / body. Successful 200 because the runtime does not reject the syntax.
func TestExecuteFile_WithRandomIntSystemVariable_SwappedMinMaxArgs(t *testing.T) {
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
}

// TestExecuteFile_WithRandomIntSystemVariable_MalformedArgs: System Variables {{$randomInt [MIN MAX]}}.
func TestExecuteFile_WithRandomIntSystemVariable_MalformedArgs(t *testing.T) {
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
}

// TestExecuteFile_WithFakerPersonData: Faker Library Support - Person/Identity Data.
//
// regex (NotEmpty) + NotContains "{{" + (for first/last/full) field-count.
func TestExecuteFile_WithFakerPersonData(t *testing.T) {
	given, when, then := newParts(t)

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
		serverReceivedFakerHeaders(0, []fakerHeaderRule{
			{key: "X-Random-First-Name", pattern: `^\S+$`, fieldCount: 1},
			{key: "X-Random-Last-Name", pattern: `^\S+$`, fieldCount: 1},
			{key: "X-Random-Full-Name", pattern: `^\S+ \S+$`, fieldCount: 2},
			{key: "X-Random-Job-Title", pattern: `^.+$`, fieldCount: -1},
		}).and().
		serverReceivedFakerHeaders(1, []fakerHeaderRule{
			{key: "X-Random-First-Name-Dot", pattern: `^\S+$`, fieldCount: 1},
			{key: "X-Random-Last-Name-Dot", pattern: `^\S+$`, fieldCount: 1},
			{key: "X-Random-Full-Name-Dot", pattern: `^\S+ \S+$`, fieldCount: 2},
			{key: "X-Random-Job-Title-Dot", pattern: `^.+$`, fieldCount: -1},
		})
}

// TestExecuteFile_WithContactAndInternetFakerData: Faker Library - Contact and Internet Data.
func TestExecuteFile_WithContactAndInternetFakerData(t *testing.T) {
	given, when, then := newParts(t)

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
		serverReceivedFakerHeaders(0, []fakerHeaderRule{
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
		}).and().
		serverReceivedFakerHeaders(1, []fakerHeaderRule{
			{key: "X-Phone-Dot", pattern: `^\(\d{3}\) \d{3}-\d{4}$`, fieldCount: -1},
			{key: "X-Address-Dot", pattern: `^.+$`, fieldCount: -1},
			{key: "X-City-Dot", pattern: `^.+$`, fieldCount: -1},
			{key: "X-Url-Dot", pattern: `^https?://`, fieldCount: -1},
			{key: "X-Mac-Dot", pattern: `^([0-9a-f]{2}:){5}[0-9a-f]{2}$`, fieldCount: -1},
		}).and().
		capturedJSONFieldNotContains(0, "{{", "contact", "phone").and().
		capturedJSONFieldNotContains(0, "{{", "contact", "address", "street").and().
		capturedJSONFieldNotContains(0, "{{", "technical", "website").and().
		capturedJSONFieldMatchesRegexp(0, `^https?://`, "technical", "website").and().
		capturedJSONFieldNotContains(1, "{{", "contact", "phone").and().
		capturedJSONFieldNotContains(1, "{{", "contact", "address", "street").and().
		capturedJSONFieldNotContains(1, "{{", "technical", "website").and().
		capturedJSONFieldMatchesRegexp(1, `^https?://`, "technical", "website")
}

// {{$randomPassword N}} substitutes an N-char password from the default charset.
func TestExecuteFile_WithRandomPasswordSystemVariable_ValidLength(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### pw
POST {{server}}/pw

{{$randomPassword 8}}`).and().
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(1).and().
		serverReceivedBodyLengthIs(0, 8)
}

// programmatic "password.charset" overrides the random-password charset.
func TestExecuteFile_WithRandomPasswordSystemVariable_CharsetOverride(t *testing.T) {
	given, when, then := newParts(t)

	given.
		anEchoServer().and().
		aHttpFile(`### pw
POST {{server}}/pw

{{$randomPassword 6}}`).and().
		aClient(rc.WithVars(map[string]any{
			"password": map[string]string{"charset": "xyz"},
		}))

	when.
		executeFile()

	then.
		noError().and().
		capturedRequestCount(1).and().
		serverReceivedBodyMatches(0, `^[xyz]{6}$`)
}

// A malformed random-password length leaves the placeholder unresolved.
func TestParseFile_WithRandomPasswordSystemVariable_MalformedLength(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aHttpFile(`### pw
POST https://example.com/pw

{{$randomPassword -3}}`).and().
		aClient()

	when.
		parsingFile()

	then.
		parseSucceeded().and().
		parsedRequestBodyIs(0, "{{$randomPassword -3}}")
}
