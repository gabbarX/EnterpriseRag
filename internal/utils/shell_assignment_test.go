package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskCommandAssignmentsHidesValues(t *testing.T) {
	command := `export VENDOR_API_KEY="sk-abc.def"; TOKEN=plain ./run --model imagegen-4`

	got := MaskCommandAssignments(command)

	assert.NotContains(t, got, "sk-abc.def")
	assert.NotContains(t, got, "plain")
	assert.Contains(t, got, "VENDOR_API_KEY=***")
	assert.Contains(t, got, "TOKEN=***")
	assert.Contains(t, got, "--model imagegen-4")
}

func TestExtractShellAssignmentsFromExportAndPrefix(t *testing.T) {
	command := `unset HTTP_PROXY; export VENDOR_API_KEY="sk-abc.def"; cd /workspace && ./run`

	got := ExtractShellAssignments(command)

	require.Equal(t, "sk-abc.def", got["VENDOR_API_KEY"])
	assert.NotContains(t, got, "HTTP_PROXY")
}
