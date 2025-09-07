package utils

import (
	"os"

	"github.com/psyb0t/ctxerrors"
)

type TempDirCleanupFunc = func() error

func CreateTempDir(pattern string) (string, TempDirCleanupFunc, error) {
	tempDir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", nil, ctxerrors.Wrap(
			err,
			"failed to create temp dir",
		)
	}

	cleanupFunc := func() error {
		if err := os.RemoveAll(tempDir); err != nil {
			return ctxerrors.Wrap(err, "failed to remove temp dir")
		}

		return nil
	}

	return tempDir, cleanupFunc, nil
}
