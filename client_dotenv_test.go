package restclient_test

import (
	"net/http"
	"testing"
)

// TestExecuteFile_DotEnvProcessEnvPlaceholderExpands verifies .env values with
// {{$processEnv X}} placeholders resolve before use (#49).
func TestExecuteFile_DotEnvProcessEnvPlaceholderExpands(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("ISS49_CID", "env-value").and().
		aDotEnvFile("CID={{$processEnv ISS49_CID}}\n").and().
		anEchoServer().and().
		aHttpFile("### dotenv echo\nGET {{server}}/{{client}}\n").
		aClient()

	when.
		executeFile()

	then.
		noError().and().
		serverReceivedMethodAndPath(0, http.MethodGet, "/env-value")
}

// TestExecuteFile_EnvFlagLoadsNamedEnvFile verifies WithDotEnvName loads
// .env.<name> after .env without dropping base keys (#48).
func TestExecuteFile_EnvFlagLoadsNamedEnvFile(t *testing.T) {
	given, when, then := newParts(t)

	given.
		aDotEnvFile("A=base\n").and().
		aDotEnvNamedFile("staging", "A=staging\nB=two\n").and().
		anEchoServer().and().
		aHttpFile("### two requests\nGET {{server}}/{{A}}\n\n###\nGET {{server}}/{{B}}\n").
		aClientWithDotEnvName("staging")

	when.
		executeFile()

	then.
		noError().and().
		serverReceivedMethodAndPath(0, http.MethodGet, "/staging").and().
		serverReceivedMethodAndPath(1, http.MethodGet, "/two")
}

// TestExecuteFile_NamedEnvFilePlaceholderExpands verifies placeholders inside
// .env.<name> values are expanded too (#48 + #49 combined).
func TestExecuteFile_NamedEnvFilePlaceholderExpands(t *testing.T) {
	given, when, then := newParts(t)

	given.
		withEnv("ISS49_S", "from-process").and().
		aDotEnvFileRemoved().and().
		aDotEnvNamedFile("staging", "CID={{$processEnv ISS49_S}}\n").and().
		anEchoServer().and().
		aHttpFile("### named echo\nGET {{server}}/{{CID}}\n").
		aClientWithDotEnvName("staging")

	when.
		executeFile()

	then.
		noError().and().
		serverReceivedMethodAndPath(0, http.MethodGet, "/from-process")
}
