package restclient

import (
	"fmt"
	"github.com/google/uuid"
	"math/rand"
	"strconv"
	"strings"
)

// substituteRandomVariables handles the substitution of $random.* variables.
func substituteRandomVariables(text string, programmaticVars map[string]any) string {
	// Integer types
	text = reRandomInt.ReplaceAllStringFunc(text,
		_substituteRandomIntFunc(reRandomInt, defaultRandomMinInt, defaultRandomMaxInt))
	text = reRandomDotInteger.ReplaceAllStringFunc(text,
		_substituteRandomIntFunc(reRandomDotInteger, defaultRandomMinInt, defaultRandomMaxInt))

	// Float types
	text = reRandomFloat.ReplaceAllStringFunc(text,
		_substituteRandomFloatFunc(reRandomFloat, defaultRandomMinFloat, defaultRandomMaxFloat))
	text = reRandomDotFloat.ReplaceAllStringFunc(text,
		_substituteRandomFloatFunc(reRandomDotFloat, defaultRandomMinFloat, defaultRandomMaxFloat))

	// Boolean
	text = strings.ReplaceAll(text, "{{$randomBoolean}}", strconv.FormatBool(rand.Intn(2) == 0))

	// Hexadecimal
	text = reRandomHex.ReplaceAllStringFunc(text, _substituteRandomHexHelper(reRandomHex, defaultRandomHexLength))
	text = reRandomDotHexadecimal.ReplaceAllStringFunc(text,
		_substituteRandomHexHelper(reRandomDotHexadecimal, defaultRandomHexLength))

	// Alphabetic / Alphanumeric
	text = reRandomDotAlphabetic.ReplaceAllStringFunc(text,
		_substituteRandomLengthCharsetFunc(reRandomDotAlphabetic, charsetAlphabetic))
	// Uses underscore
	text = reRandomAlphaNumeric.ReplaceAllStringFunc(text,
		_substituteRandomLengthCharsetFunc(reRandomAlphaNumeric, charsetAlphaNumericWithExtra))
	// No underscore
	text = reRandomDotAlphanumeric.ReplaceAllStringFunc(text,
		_substituteRandomLengthCharsetFunc(reRandomDotAlphanumeric, charsetAlphaNumeric))

	// General Random String
	text = reRandomString.ReplaceAllStringFunc(text, _substituteRandomLengthCharsetFunc(reRandomString, charsetFull))

	// Email
	emailGenerator := func() string {
		return fmt.Sprintf("%s@%s.com",
			randomStringFromCharset(10, charsetAlphaNumeric),
			randomStringFromCharset(7, charsetAlphabetic))
	}
	text = strings.ReplaceAll(text, "{{$randomEmail}}", emailGenerator())
	text = strings.ReplaceAll(text, "{{$random.email}}", emailGenerator())

	// Domain
	text = strings.ReplaceAll(text, "{{$randomDomain}}",
		fmt.Sprintf("%s.com", randomStringFromCharset(10, charsetAlphabetic)))

	// IP Addresses
	text = strings.ReplaceAll(text, "{{$randomIPv4}}",
		fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256)))

	text = strings.ReplaceAll(text, "{{$randomIPv6}}", func() string {
		segments := make([]string, 8)
		for i := 0; i < 8; i++ {
			segments[i] = fmt.Sprintf("%x", rand.Intn(0x10000))
		}
		return strings.Join(segments, ":")
	}())

	// UUID
	text = strings.ReplaceAll(text, "{{$randomUUID}}", uuid.New().String())

	// Password (uses programmaticVars, so it calls the existing _substituteRandomPasswordFunc with modification)
	text = reRandomPassword.ReplaceAllStringFunc(text, func(match string) string {
		return _substituteRandomPasswordFunc(match, programmaticVars)
	})

	// Color
	text = strings.ReplaceAll(text, "{{$randomColor}}",
		fmt.Sprintf("#%02x%02x%02x", rand.Intn(256), rand.Intn(256), rand.Intn(256)))

	// Word
	if len(randomWords) > 0 { // Prevent panic on empty slice
		text = strings.ReplaceAll(text, "{{$randomWord}}", randomWords[rand.Intn(len(randomWords))])
	}

	// Person/Identity data (faker variables)
	text = substituteFakerVariables(text)

	return text
}

// _substituteRandomPasswordFunc handles the substitution of $randomPassword.* variables.
// It now accepts programmaticVars to allow charset overrides.
func _substituteRandomPasswordFunc(match string, programmaticVars map[string]any) string {
	length := parsePasswordLength(match)
	if length < 0 {
		return match // Malformed length
	}
	if length == 0 {
		return ""
	}

	charset := getPasswordCharset(programmaticVars)
	return randomStringFromCharset(length, charset)
}

// parsePasswordLength extracts and validates the length parameter from a password match
func parsePasswordLength(match string) int {
	parts := reRandomPassword.FindStringSubmatch(match)
	length := defaultRandomPasswordLength
	if len(parts) >= 2 && parts[1] != "" {
		parsedLen, err := strconv.Atoi(parts[1])
		if err != nil || parsedLen < 0 {
			return -1 // Invalid length
		}
		length = parsedLen
	}
	return length
}

// getPasswordCharset determines the charset to use for password generation
func getPasswordCharset(programmaticVars map[string]any) string {
	if psVars, ok := programmaticVars["password"]; ok {
		if psMap, ok := psVars.(map[string]string); ok {
			if charset, ok := psMap["charset"]; ok && charset != "" {
				return charset
			}
		}
	}
	return charsetFull
}
