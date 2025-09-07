package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/psyb0t/common-go/constants"
	commonerrors "github.com/psyb0t/common-go/errors"
	"github.com/psyb0t/ctxerrors"
	"github.com/sirupsen/logrus"
)

func IsPointer(v any) bool {
	if v == nil {
		return false
	}

	t := reflect.TypeOf(v)

	return t.Kind() == reflect.Ptr
}

func GetPointer[T any](to T) *T {
	return &to
}

func DownloadFile(ctx context.Context, url, targetPath string) error {
	logrus.Debugf("downloading file from URL: %s", url)

	httpClient := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ctxerrors.Wrap(err, "failed to create HTTP request")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return ctxerrors.Wrap(err, "failed to download file")
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.Errorf("failed to close response body: %s", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return ctxerrors.New(fmt.Sprintf("unexpected status code: %d", resp.StatusCode))
	}

	out, err := os.OpenFile(
		targetPath,
		os.O_WRONLY|os.O_CREATE,
		constants.DefaultFilePermissions,
	)
	if err != nil {
		return ctxerrors.Wrap(err, "failed to create local file")
	}

	defer func() {
		if err := out.Close(); err != nil {
			logrus.Errorf("failed to close file: %s", err)
		}
	}()

	logrus.Debugf("writing file to: %s", targetPath)

	if _, err = io.Copy(out, resp.Body); err != nil {
		return ctxerrors.Wrap(err, "failed to write file to temp dir")
	}

	return nil
}

// TODO: refactor
//
//nolint:godox
const downloadFilesWorkerLimit = 5

// DownloadFiles downloads multiple files from URLs to target paths in parallel
func DownloadFiles(ctx context.Context, urlsToTargetPaths map[string]string) error {
	if len(urlsToTargetPaths) == 0 {
		return nil
	}

	logrus.Debugf("downloading %d files in parallel", len(urlsToTargetPaths))

	var wg sync.WaitGroup

	errChan := make(chan error, len(urlsToTargetPaths))

	// Create a worker pool with reasonable concurrency limit
	workerLimit := min(downloadFilesWorkerLimit, len(urlsToTargetPaths))

	// Create a semaphore channel to limit concurrent downloads
	sem := make(chan struct{}, workerLimit)

	for url, targetPath := range urlsToTargetPaths {
		wg.Add(1)

		// Capture variables for goroutine
		url := url
		targetPath := targetPath

		go func() {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}

			defer func() { <-sem }() // Release semaphore

			// Download individual file
			if err := DownloadFile(ctx, url, targetPath); err != nil {
				errChan <- ctxerrors.Wrapf(err, "failed to download file from %s to %s", url, targetPath)
			}
		}()
	}

	// Wait for all downloads to complete
	wg.Wait()
	close(errChan)

	// Check if any errors occurred
	errs := make([]error, 0, len(errChan))
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		// Combine all errors into a single error message
		var errMsgs []string
		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}

		return ctxerrors.Wrapf(
			commonerrors.ErrCouldNotDownloadFiles,
			"errors downloading files: %s",
			strings.Join(errMsgs, "; "),
		)
	}

	return nil
}

func CreateBase64ImageURI(base64Data string, contentType string) string {
	return fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)
}

func AnyToReader(v any) (io.Reader, error) {
	switch val := v.(type) {
	case string:
		// Convert string directly to byte slice
		return bytes.NewReader([]byte(val)), nil

	case []byte:
		// Already a byte slice, just return a reader
		return bytes.NewReader(val), nil

	case io.Reader:
		// Already a reader, return as is
		return val, nil

	case nil:
		// Return empty reader for nil
		return bytes.NewReader([]byte{}), nil

	default:
		// Try to determine if it's a simple type or complex type
		switch val := v.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
			return bytes.NewReader(fmt.Appendf([]byte{}, "%v", val)), nil

		default:
			// For complex types (struct, map, slice, etc.), use JSON marshaling
			data, err := json.Marshal(val)
			if err != nil {
				return nil, ctxerrors.Wrap(err, "failed to marshal to JSON")
			}

			return bytes.NewReader(data), nil
		}
	}
}

func ZeroOrPtrVal[T any](ptr *T) T { //nolint:ireturn
	var zero T

	if ptr != nil {
		return *ptr
	}

	return zero
}

func DefaultOrPtrVal[T any](ptr *T, defaultVal T) T { //nolint:ireturn
	if ptr != nil {
		return *ptr
	}

	return defaultVal
}
