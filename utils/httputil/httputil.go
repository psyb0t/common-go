package httputil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/psyb0t/common-go/constants"
	commonerrors "github.com/psyb0t/common-go/errors"
	"github.com/psyb0t/ctxerrors"
)

func DownloadFile(ctx context.Context, url, targetPath string) error {
	slog.Debug("downloading file", "url", url, "target", targetPath)

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
			slog.Error("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return ctxerrors.Wrapf(
			commonerrors.ErrUnexpectedHTTPStatusCode,
			"status code: %d",
			resp.StatusCode,
		)
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
			slog.Error("failed to close file", "error", err)
		}
	}()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return ctxerrors.Wrap(err, "failed to write file")
	}

	return nil
}

const downloadFilesWorkerLimit = 5

func DownloadFiles(ctx context.Context, urlsToTargetPaths map[string]string) error {
	if len(urlsToTargetPaths) == 0 {
		return nil
	}

	slog.Debug("downloading files", "count", len(urlsToTargetPaths))

	var wg sync.WaitGroup

	errChan := make(chan error, len(urlsToTargetPaths))

	workerLimit := min(downloadFilesWorkerLimit, len(urlsToTargetPaths))
	sem := make(chan struct{}, workerLimit)

	for url, targetPath := range urlsToTargetPaths {
		wg.Go(func() {
			sem <- struct{}{}

			defer func() { <-sem }()

			if err := DownloadFile(ctx, url, targetPath); err != nil {
				errChan <- ctxerrors.Wrapf(err, "failed to download %s to %s", url, targetPath)
			}
		})
	}

	wg.Wait()
	close(errChan)

	errs := make([]error, 0, len(errChan))
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
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
		return bytes.NewReader([]byte(val)), nil
	case []byte:
		return bytes.NewReader(val), nil
	case io.Reader:
		return val, nil
	case nil:
		return bytes.NewReader([]byte{}), nil
	default:
		data, err := json.Marshal(val)
		if err != nil {
			return nil, ctxerrors.Wrap(err, "failed to marshal to JSON")
		}

		return bytes.NewReader(data), nil
	}
}

func IsPointer(v any) bool {
	return v != nil && fmt.Sprintf("%T", v)[0] == '*'
}
