package restclient_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type clientParts struct {
	parts
}

func newClientParts(t *testing.T) (given, when, then *clientParts) {
	t.Helper()

	c := &clientParts{parts: parts{
		T:             t,
		require:       require.New(t),
		assert:        assert.New(t),
		baseDir:       t.TempDir(),
		trackedValues: make(map[string][]string),
	}}

	return c, c, c
}
