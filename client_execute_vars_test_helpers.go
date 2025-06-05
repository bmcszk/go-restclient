package restclient

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// createTempHTTPFileFromString creates a temporary .http file with the given content.
// It returns the path to the file and registers a cleanup function to remove the temp directory.
func createTempHTTPFileFromString(t *testing.T, content string) string {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "test-http-helpers-") // Changed prefix for clarity
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir %s: %v", tempDir, err)
		}
	})

	filePath := filepath.Join(tempDir, "test_helper.http") // Changed filename for clarity
	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	return filePath
}
