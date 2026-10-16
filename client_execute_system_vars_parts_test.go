package restclient_test

import (
	"os"
	"path/filepath"
)

// expectedResponseFile writes the given .hresp content as the expected responses file.
func (p *parts) expectedResponseFile(content string) *parts {
	path := filepath.Join(p.baseDir, "expected.hresp")
	p.require.NoError(os.WriteFile(path, []byte(content), 0644))
	p.expectedFilePath = path

	return p
}
