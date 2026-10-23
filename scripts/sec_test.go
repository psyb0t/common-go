package scripts

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// These fixtures test scanner orchestration, not vulnerability detection.
const fakeScanner = `#!/usr/bin/env bash
set -euo pipefail
name=${0##*/}
if [[ "$name" == go ]]; then
    name=$2
fi
output=""
previous=""
for argument in "$@"; do
    if [[ "$previous" == --report-path || "$previous" == --output ]]; then
        output=$argument
    fi
    previous=$argument
done
report='{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"fixture"}},"results":[]}]}'
if [[ "$name" == trivy && "${FAIL_SCANNER:-}" == broken ]]; then
    exit 9
fi
if [[ -n "$output" ]]; then
    printf '%s\n' "$report" > "$output"
elif [[ "$*" == *-format=sarif* ]]; then
    printf '%s\n' "$report"
fi
if [[ "${FAIL_SCANNER:-}" == "$name" ]]; then
    exit 7
fi
`

func TestSecurityScript(t *testing.T) {
	t.Parallel()
	scriptDirectory, err := os.Getwd()
	require.NoError(t, err)

	for _, tc := range []struct {
		name       string
		failure    string
		wantError  bool
		wantReport bool
	}{
		{name: "clean", wantReport: true},
		{name: "secret finding", failure: "gitleaks", wantError: true, wantReport: true},
		{name: "dependency finding", failure: "govulncheck", wantError: true, wantReport: true},
		{name: "manifest finding", failure: "trivy", wantError: true, wantReport: true},
		{name: "scanner operational failure", failure: "broken", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			work := t.TempDir()
			bin := filepath.Join(work, "bin")
			require.NoError(t, os.Mkdir(bin, 0o700))
			for _, name := range []string{"go", "trivy"} {
				require.NoError(t, os.WriteFile(filepath.Join(bin, name), []byte(fakeScanner), 0o700))
			}
			command := exec.CommandContext(t.Context(), "bash", filepath.Join(scriptDirectory, "sec.sh"))
			command.Dir = work
			command.Env = append(os.Environ(),
				"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
				"FAIL_SCANNER="+tc.failure,
				"LOG_FILE="+filepath.Join(work, "scan.log"),
			)
			output, err := command.CombinedOutput()
			if tc.wantError {
				require.Error(t, err, "%s", output)
			} else {
				require.NoError(t, err, "%s", output)
			}
			if tc.wantReport {
				data, err := os.ReadFile(filepath.Join(work, "sec.sarif"))
				require.NoError(t, err)
				var report struct {
					Version string
					Runs    []json.RawMessage
				}
				require.NoError(t, json.Unmarshal(data, &report))
				require.Equal(t, "2.1.0", report.Version)
				require.Len(t, report.Runs, 4)
			}
		})
	}
}
