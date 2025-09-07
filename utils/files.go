package utils

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	commonerrors "github.com/psyb0t/common-go/errors"
	"github.com/psyb0t/ctxerrors"
	"github.com/sirupsen/logrus"
)

func FileToBase64(inputPath string) (string, error) {
	// Open the file
	file, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			logrus.Errorf("failed to close file: %s", err)
		}
	}()

	// Read the file contents
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file contents: %w", err)
	}

	// Encode the file contents to Base64
	base64String := base64.StdEncoding.EncodeToString(fileBytes)

	return base64String, nil
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
		return ctxerrors.Wrapf(
			err,
			"error checking if path exists: %s",
			path,
		)
	}

	if !exists {
		return ctxerrors.Wrapf(
			commonerrors.ErrFileNotFound,
			"path: %s",
			path,
		)
	}

	return nil
}
