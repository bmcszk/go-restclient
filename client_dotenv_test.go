package restclient_test

import (
	"testing"
)

// TestExecuteFile_DotEnvProcessEnvPlaceholderExpands: .env values support {{$processEnv}}
// so secrets can come from the real environment (#49).
func TestExecuteFile_DotEnvProcessEnvPlaceholderExpands(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("ISS49_CID", "env-value").and().
		aDotEnvFile("CID={{$processEnv ISS49_CID}}").and().
		anEchoServer().and().
		aHttpFile("@client = {{CID}}\n\nPOST {{server}}/echo\nX-Client: {{client}}\n\nok").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedHeaderValue(0, "X-Client", "env-value")
}

// TestExecuteFile_DotEnvUnresolvablePlaceholderStaysRaw: missing process env leaves the
// {{$processEnv}} placeholder literal in the .env value (#49 pinned semantics).
func TestExecuteFile_DotEnvUnresolvablePlaceholderStaysRaw(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aDotEnvFile("CID={{$processEnv ISS49_MISSING}}").and().
		anEchoServer().and().
		aHttpFile("@client = {{CID}}\n\nPOST {{server}}/echo\nX-Client: {{client}}\n\nok").and().
		aClient()

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedHeaderValue(0, "X-Client", "{{$processEnv ISS49_MISSING}}")
}

// TestExecuteFile_EnvFlagLoadsNamedEnvFile: --env staging loads .env first, then .env.staging
// overrides only the keys it defines (#48).
func TestExecuteFile_EnvFlagLoadsNamedEnvFile(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aDotEnvFile("A=base").and().
		aDotEnvNamedFile("staging", "A=staging\nB=two").and().
		anEchoServer().and().
		aHttpFile("@client = {{A}}\n\nPOST {{server}}/echo\nX-A: {{A}}\nX-B: {{B}}\n\nok").and().
		aClientWithEnvName("staging")

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedHeaderValue(0, "X-A", "staging").and().
		serverReceivedHeaderValue(0, "X-B", "two")
}

// TestExecuteFile_NamedEnvFilePlaceholderExpands: .env.staging values support
// {{$processEnv}} expansion like .env (#49 + #48 combined).
func TestExecuteFile_NamedEnvFilePlaceholderExpands(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("ISS49_S", "expanded-staging").and().
		aDotEnvFileRemoved().and().
		aDotEnvNamedFile("staging", "CID={{$processEnv ISS49_S}}").and().
		anEchoServer().and().
		aHttpFile("@client = {{CID}}\n\nPOST {{server}}/echo\nX-Client: {{client}}\n\nok").and().
		aClientWithEnvName("staging")

	when.
		executeFile()

	then.
		responseCount(1).and().
		noError().and().
		serverReceivedHeaderValue(0, "X-Client", "expanded-staging")
}
