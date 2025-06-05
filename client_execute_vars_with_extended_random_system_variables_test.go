package restclient

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteFile_WithExtendedRandomSystemVariables(t *testing.T) {
	// Given
	var interceptedRequest struct {
		Body string
	}
	server := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		interceptedRequest.Body = string(bodyBytes)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client, _ := NewClient()
	requestFilePath := createTestFileFromTemplate(t, "testdata/http_request_files/system_var_extended_random.http", struct{ ServerURL string }{ServerURL: server.URL})

	// When
	responses, err := client.ExecuteFile(context.Background(), requestFilePath)

	// Then
	require.NoError(t, err, "ExecuteFile should not return an error for extended random variable processing")
	require.Len(t, responses, 1, "Expected 1 response")

	resp := responses[0]
	assert.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var bodyJSON map[string]string
	err = json.Unmarshal([]byte(interceptedRequest.Body), &bodyJSON)
	require.NoError(t, err, "Failed to unmarshal response body JSON: %s", interceptedRequest.Body)

	// $random.integer
	randIntValue, err := strconv.Atoi(bodyJSON["randInt"])
	assert.NoError(t, err, "randInt should be an integer")
	assert.GreaterOrEqual(t, randIntValue, 10, "randInt should be >= 10")
	assert.LessOrEqual(t, randIntValue, 20, "randInt should be <= 20")

	randIntNegativeValue, err := strconv.Atoi(bodyJSON["randIntNegative"])
	assert.NoError(t, err, "randIntNegative should be an integer")
	assert.GreaterOrEqual(t, randIntNegativeValue, -5, "randIntNegative should be >= -5")
	assert.LessOrEqual(t, randIntNegativeValue, 5, "randIntNegative should be <= 5")

	assert.Equal(t, "{{$random.integer 10 1}}", bodyJSON["randIntInvalidRange"], "randIntInvalidRange should remain unsubstituted")
	assert.Equal(t, "{{$random.integer 10 abc}}", bodyJSON["randIntInvalidArgs"], "randIntInvalidArgs should remain unsubstituted")

	// $random.float
	randFloatValue, err := strconv.ParseFloat(bodyJSON["randFloat"], 64)
	assert.NoError(t, err, "randFloat should be a float")
	assert.GreaterOrEqual(t, randFloatValue, 1.0, "randFloat should be >= 1.0")
	assert.LessOrEqual(t, randFloatValue, 2.5, "randFloat should be <= 2.5")

	randFloatNegativeValue, err := strconv.ParseFloat(bodyJSON["randFloatNegative"], 64)
	assert.NoError(t, err, "randFloatNegative should be a float")
	assert.GreaterOrEqual(t, randFloatNegativeValue, -1.5, "randFloatNegative should be >= -1.5")
	assert.LessOrEqual(t, randFloatNegativeValue, 0.5, "randFloatNegative should be <= 0.5")

	assert.Equal(t, "{{$random.float 5.0 1.0}}", bodyJSON["randFloatInvalidRange"], "randFloatInvalidRange should remain unsubstituted")

	// $random.alphabetic
	randAlphabeticValue := bodyJSON["randAlphabetic"]
	assert.Len(t, randAlphabeticValue, 10, "randAlphabetic length mismatch")
	for _, r := range randAlphabeticValue {
		assert.True(t, (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'), "randAlphabetic char not in alphabet: %c", r)
	}
	assert.Equal(t, "", bodyJSON["randAlphabeticZero"], "randAlphabeticZero should be empty")
	assert.Equal(t, "{{$random.alphabetic abc}}", bodyJSON["randAlphabeticInvalid"], "randAlphabeticInvalid should remain unsubstituted")

	// $random.alphanumeric
	randAlphanumericValue := bodyJSON["randAlphanumeric"]
	assert.Len(t, randAlphanumericValue, 15, "randAlphanumeric length mismatch")
	for _, r := range randAlphanumericValue {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isNumber := r >= '0' && r <= '9'
		isUnderscore := r == '_'
		assert.True(t, isLetter || isNumber || isUnderscore, "randAlphanumeric char not alphanumeric: %c", r)
	}

	// $random.hexadecimal
	randHexValue := bodyJSON["randHex"]
	assert.Len(t, randHexValue, 8, "randHex length mismatch")
	_, err = hex.DecodeString(randHexValue)
	assert.NoError(t, err, "randHex should be valid hexadecimal: %s", randHexValue)

	// $random.email
	randEmailValue := bodyJSON["randEmail"]
	parts := strings.Split(randEmailValue, "@")
	require.Len(t, parts, 2, "randEmail should have one @ symbol")
	domainParts := strings.Split(parts[1], ".")
	require.GreaterOrEqual(t, len(domainParts), 2, "randEmail domain should have at least one .")
	assert.Regexp(t, `^[a-zA-Z0-9_]+@[a-zA-Z]+\.[a-zA-Z]{2,3}$`, randEmailValue, "randEmail format is incorrect")
}
