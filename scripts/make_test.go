package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const fakeCoverageTool = `#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" == test ]]; then
    printf 'mode: set\n' > coverage.txt
elif [[ -n "${COVERAGE_PERCENT:-}" ]]; then
    printf 'total: (statements) %s%%\n' "$COVERAGE_PERCENT"
fi
`

func TestCoverageGate(t *testing.T) {
	t.Parallel()
	scriptDirectory, err := os.Getwd()
	require.NoError(t, err)
	for _, tc := range []struct {
		name      string
		coverage  string
		wantError bool
	}{
		{name: "exact threshold", coverage: "60.0"},
		{name: "below threshold", coverage: "59.99", wantError: true},
		{name: "missing coverage", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			work := t.TempDir()
			bin := filepath.Join(work, "bin")
			require.NoError(t, os.Mkdir(bin, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(bin, "go"), []byte(fakeCoverageTool), 0o700))
			command := exec.CommandContext(t.Context(), "make",
				"-f", filepath.Join(scriptDirectory, "..", "Makefile"),
				"COMMON_GO_IN_DEV=1", "MIN_TEST_COVERAGE=60", "test-coverage",
			)
			command.Dir = work
			command.Env = append(os.Environ(),
				"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
				"COVERAGE_PERCENT="+tc.coverage,
			)
			output, err := command.CombinedOutput()
			if tc.wantError {
				require.Error(t, err, "%s", output)
			} else {
				require.NoError(t, err, "%s", output)
			}
			_, err = os.Stat(filepath.Join(work, "coverage.txt"))
			require.ErrorIs(t, err, os.ErrNotExist)
			if tc.coverage != "" {
				percentage, err := os.ReadFile(filepath.Join(work, "coverage-percent.txt"))
				require.NoError(t, err)
				require.Equal(t, tc.coverage+"\n", string(percentage))
			}
		})
	}
}
