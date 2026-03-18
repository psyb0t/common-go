package fileutil

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/google/uuid"
	commonerrors "github.com/psyb0t/common-go/errors"
	"github.com/psyb0t/ctxerrors"
)

func FileToBase64(inputPath string) (string, error) {
	file, err := os.Open(inputPath)
	if err != nil {
		return "", ctxerrors.Wrap(err, "failed to open file")
	}

	defer func() {
		if err := file.Close(); err != nil {
			slog.Error("failed to close file", "error", err)
		}
	}()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", ctxerrors.Wrap(err, "failed to read file contents")
	}

	return base64.StdEncoding.EncodeToString(fileBytes), nil
}

func GetRandomFilename(extension string) string {
	return fmt.Sprintf("%s%s", uuid.NewString(), extension)
}

func PathExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, ctxerrors.Wrap(err, "error checking if path exists")
	}

	return true, nil
}

func ValidatePathExists(path string) error {
	exists, err := PathExists(path)
	if err != nil {
		return ctxerrors.Wrapf(err, "error checking if path exists: %s", path)
	}

	if !exists {
		return ctxerrors.Wrapf(commonerrors.ErrFileNotFound, "path: %s", path)
	}

	return nil
}

func CreateTempDir(pattern string) (string, func() error, error) {
	tempDir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", nil, ctxerrors.Wrap(err, "failed to create temp dir")
	}

	cleanupFunc := func() error {
		if err := os.RemoveAll(tempDir); err != nil {
			return ctxerrors.Wrap(err, "failed to remove temp dir")
		}

		return nil
	}

	return tempDir, cleanupFunc, nil
}
