package restclient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

const (
	requestSeparator   = "###"
	commentPrefix      = "#"
	slashCommentPrefix = "//"
)



// loadEnvironmentFile attempts to load a specific environment's variables from a JSON file.
// It returns the variables map or nil if the environment/file is not found or on error.
func loadEnvironmentFile(filePath string, selectedEnvName string) (map[string]string, error) {
	if selectedEnvName == "" {
		return nil, nil // No environment selected, nothing to load
	}

	if _, statErr := os.Stat(filePath); statErr != nil {
		if os.IsNotExist(statErr) {
			// Environment file not found
			return nil, nil // File not found is not an error, just means no vars from this file
		}
		// Another error occurred trying to stat the file (e.g., permissions)
		slog.Warn("Error checking environment file", "error", statErr, "file", filePath)
		return nil, fmt.Errorf("checking environment file %s: %w", filePath, statErr)
	}

	envFileBytes, readErr := os.ReadFile(filePath)
	if readErr != nil {
		slog.Warn("Failed to read environment file", "error", readErr, "file", filePath)
		return nil, fmt.Errorf("reading environment file %s: %w", filePath, readErr)
	}

	var allEnvs map[string]map[string]string
	if unmarshalErr := json.Unmarshal(envFileBytes, &allEnvs); unmarshalErr != nil {
		slog.Warn("Failed to unmarshal environment file", "error", unmarshalErr, "file", filePath)
		return nil, fmt.Errorf("unmarshalling environment file %s: %w", filePath, unmarshalErr)
	}

	if selectedEnvVars, ok := allEnvs[selectedEnvName]; ok {
		return selectedEnvVars, nil
	}

	// Selected environment not found
	return nil, nil // Environment not found in this file
}

// ParseRequestFile reads a .rest or .http file and parses it into a ParsedFile struct
// containing one or more Request definitions. It requires a `client` instance to access
// programmatic and request-scoped system variables, which are used at parse time to resolve
// the values of file-level variables (e.g., `@host = {{programmatic_var}}` or `@api_key = {{$uuid}}`).
//
// The `filePath` is used for opening the file and for context in error messages.
// A .env file in the same directory as `filePath` will also be loaded and used for resolving
// `@variable` definitions if present.
func parseRequestFile(filePath string, client *Client, importStack []string) (*ParsedFile, error) {
	absFilePath, newImportStack, err := prepareParsingContext(filePath, importStack)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(absFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open request file %s: %w", absFilePath, err)
	}
	defer func() { _ = file.Close() }()

	parsingVars := setupParsingVariables(filePath, client)

	reader := bufio.NewReader(file)
	parsedFile, err := parseRequests(
		reader, absFilePath, client, parsingVars.requestScopedSystemVars,
		parsingVars.osEnvGetter, parsingVars.dotEnvVars, newImportStack)
	if err != nil {
		return nil, err
	}

	loadEnvironmentSpecificVariables(filePath, client, parsedFile)
	return parsedFile, nil
}

// parsingVariables holds variables needed for parsing
type parsingVariables struct {
	dotEnvVars               map[string]string
	osEnvGetter              func(string) (string, bool)
	requestScopedSystemVars  map[string]string
}

// prepareParsingContext prepares the file path and import stack for parsing
func prepareParsingContext(filePath string, importStack []string) (string, []string, error) {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get absolute path for %s: %w", filePath, err)
	}

	if err := checkCircularImports(absFilePath, importStack); err != nil {
		return "", nil, err
	}

	newImportStack := append(importStack, absFilePath)
	return absFilePath, newImportStack, nil
}

// checkCircularImports checks for circular import patterns
func checkCircularImports(absFilePath string, importStack []string) error {
	for _, importedPath := range importStack {
		if importedPath == absFilePath {
			return fmt.Errorf(
				"circular import detected: '%s' already in import stack %v",
				absFilePath, importStack)
		}
	}
	return nil
}

// setupParsingVariables sets up all variables needed for parsing
func setupParsingVariables(filePath string, client *Client) parsingVariables {
	return parsingVariables{
		dotEnvVars:              loadDotEnvForParsing(filePath),
		osEnvGetter:             func(key string) (string, bool) { return os.LookupEnv(key) },
		requestScopedSystemVars: generateRequestScopedVarsForParsing(client),
	}
}

// loadDotEnvForParsing loads .env variables for parsing
func loadDotEnvForParsing(filePath string) map[string]string {
	dotEnvVars := make(map[string]string)
	envFilePath := filepath.Join(filepath.Dir(filePath), ".env")
	if _, statErr := os.Stat(envFilePath); statErr == nil {
		if loadedVars, loadErr := godotenv.Read(envFilePath); loadErr == nil {
			dotEnvVars = loadedVars
		}
	}
	return dotEnvVars
}

// generateRequestScopedVarsForParsing generates request-scoped system variables
func generateRequestScopedVarsForParsing(client *Client) map[string]string {
	if client != nil {
		return client.generateRequestScopedSystemVariables()
	}
	return make(map[string]string)
}

// loadEnvironmentSpecificVariables loads environment-specific variables from
// http-client.env.json and http-client.private.env.json based on the client's
// selected environment. It updates parsedFile.EnvironmentVariables.
// originalFilePath is the path originally passed to parseRequestFile, used for resolving .env.json files.
func loadEnvironmentSpecificVariables(originalFilePath string, client *Client, parsedFile *ParsedFile) {
	if client == nil || client.selectedEnvironmentName == "" || parsedFile == nil {
		return
	}

	fileDir := filepath.Dir(originalFilePath)
	mergedEnvVars := loadEnvironmentFiles(fileDir, client.selectedEnvironmentName)

	if len(mergedEnvVars) > 0 {
		parsedFile.EnvironmentVariables = mergedEnvVars
	} else {
		ensureEnvironmentVariablesInitialized(parsedFile, client.selectedEnvironmentName, fileDir)
	}
}

// loadEnvironmentFiles loads variables from both public and private environment files
func loadEnvironmentFiles(fileDir, selectedEnvName string) map[string]string {
	mergedEnvVars := make(map[string]string)
	
	loadPublicEnvFile(fileDir, selectedEnvName, mergedEnvVars)
	loadPrivateEnvFile(fileDir, selectedEnvName, mergedEnvVars)
	
	return mergedEnvVars
}

// loadPublicEnvFile loads variables from http-client.env.json
func loadPublicEnvFile(fileDir, selectedEnvName string, mergedEnvVars map[string]string) {
	publicEnvFile := filepath.Join(fileDir, "http-client.env.json")
	if publicVars, err := loadEnvironmentFile(publicEnvFile, selectedEnvName); err == nil && publicVars != nil {
		for k, v := range publicVars {
			mergedEnvVars[k] = v
		}
		// Environment variables loaded from public file
	}
}

// loadPrivateEnvFile loads variables from http-client.private.env.json (overrides public ones)
func loadPrivateEnvFile(fileDir, selectedEnvName string, mergedEnvVars map[string]string) {
	privateEnvFile := filepath.Join(fileDir, "http-client.private.env.json")
	if privateVars, err := loadEnvironmentFile(privateEnvFile, selectedEnvName); err == nil && privateVars != nil {
		for k, v := range privateVars {
			mergedEnvVars[k] = v
		}
		// Environment variables loaded from private file
	}
}

// ensureEnvironmentVariablesInitialized ensures the EnvironmentVariables map is initialized
func ensureEnvironmentVariablesInitialized(parsedFile *ParsedFile, _, _ string) {
	if parsedFile.EnvironmentVariables == nil {
		parsedFile.EnvironmentVariables = make(map[string]string)
	}
	// No environment-specific variables found
}

// parseRequests reads HTTP requests from a reader and parses them into a ParsedFile struct.
// It's used by parseRequestFile to process individual HTTP request files.
func parseRequests(reader *bufio.Reader, filePath string, client *Client,
	requestScopedSystemVars map[string]string, osEnvGetter func(string) (string, bool),
	dotEnvVars map[string]string, importStack []string) (*ParsedFile, error) {
	parserState := initializeParserState(filePath, client, requestScopedSystemVars, 
		osEnvGetter, dotEnvVars, importStack)
	
	if err := processFileLines(reader, parserState); err != nil {
		return nil, err
	}
	
	finalizeParseResults(parserState)
	return parserState.parsedFile, nil
}

// initializeParserState creates and initializes the parser state
func initializeParserState(filePath string, client *Client, requestScopedSystemVars map[string]string,
	osEnvGetter func(string) (string, bool), dotEnvVars map[string]string, importStack []string) *requestParserState {
	return &requestParserState{
		filePath:                filePath,
		client:                  client,
		requestScopedSystemVars: requestScopedSystemVars,
		osEnvGetter:             osEnvGetter,
		dotEnvVars:              dotEnvVars,
		importStack:             importStack,
		parsedFile: &ParsedFile{
			Requests: make([]*Request, 0),
			FileVariables: make(map[string]string),
			FilePath: filePath,
		},
		currentFileVariables:    make(map[string]string),
		lineNumber:              0,
	}
}










































