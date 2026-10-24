package db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxscope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSummarizeSQL(t *testing.T) {
	t.Parallel()

	secret := "secret-value-that-must-not-be-logged"
	testCases := []struct {
		name          string
		query         string
		wantPreview   string
		wantTruncated bool
	}{
		{
			name: "redacts string and numeric values",
			query: fmt.Sprintf(
				"SELECT * FROM users WHERE token = '%s' AND id = 42",
				secret,
			),
			wantPreview: "SELECT * FROM users WHERE token = ? AND id = ?",
		},
		{
			name:        "collapses whitespace",
			query:       "SELECT  *\nFROM users\tWHERE id = 7",
			wantPreview: "SELECT * FROM users WHERE id = ?",
		},
		{
			name:        "redacts postgres escape string",
			query:       `SELECT E'secret\'tail'`,
			wantPreview: "SELECT ?",
		},
		{
			name:        "redacts postgres dollar string",
			query:       "SELECT $payload$secret-value$payload$",
			wantPreview: "SELECT ?",
		},
		{
			name:          "bounds large multibyte query",
			query:         "SELECT " + strings.Repeat("界", maxSQLPreviewBytes),
			wantTruncated: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := summarizeSQL(tc.query)
			assert.Equal(t, len(tc.query), got.Bytes)
			assert.Equal(t, tc.wantTruncated, got.Truncated)
			assert.True(t, utf8.ValidString(got.Preview))
			assert.NotContains(t, got.Preview, secret)
			if tc.wantPreview != "" {
				assert.Equal(t, tc.wantPreview, got.Preview)
			}
			assert.LessOrEqual(t, len(got.Preview), maxSQLPreviewBytes)
		})
	}
}

func TestGormSlogLoggerTraceSkipsDisabledRecord(t *testing.T) {
	var logs bytes.Buffer

	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	fcCalls := 0
	l := &GormSlogLogger{LogLevel: logger.Info}
	l.Trace(
		context.Background(),
		time.Now(),
		func() (string, int64) {
			fcCalls++

			return "SELECT 1", 1
		},
		nil,
	)

	assert.Zero(t, fcCalls)
	assert.Empty(t, logs.String())
}

func TestGormSlogLoggerTraceBoundsSQL(t *testing.T) {
	var logs bytes.Buffer

	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	query := fmt.Sprintf(
		"INSERT INTO things (payload) VALUES ('%s')",
		strings.Repeat("sensitive", maxSQLPreviewBytes),
	)

	l := &GormSlogLogger{LogLevel: logger.Error}
	l.Trace(
		context.Background(),
		time.Now(),
		func() (string, int64) {
			return query, 1
		},
		assert.AnError,
	)

	output := logs.String()
	require.NotEmpty(t, output)
	assert.Contains(t, output, `"msg":"gorm error"`)
	assert.Contains(t, output, `"sql_bytes":`)
	assert.NotContains(t, output, strings.Repeat("sensitive", 10))
}

// captureGormRecords sends the default logger's records at every level to a
// buffer for one test and decodes them afterwards.
func captureGormRecords(t *testing.T) func() []map[string]any {
	t.Helper()

	var logs bytes.Buffer

	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs,
		&slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	return func() []map[string]any {
		records := []map[string]any{}

		for line := range strings.SplitSeq(strings.TrimSpace(logs.String()),
			"\n") {
			if line == "" {
				continue
			}

			record := map[string]any{}
			require.NoError(t, json.Unmarshal([]byte(line), &record))

			records = append(records, record)
		}

		return records
	}
}

func traceGormQuery(ctx context.Context, l logger.Interface, err error) {
	l.Trace(ctx, time.Now(), func() (string, int64) {
		return "SELECT * FROM things WHERE id = 7", 0
	}, err)
}

// A query that finds no record is an ordinary query by default, so a caller
// polling for absent work logs no errors.
func TestGormSlogLoggerTraceRecordNotFound(t *testing.T) {
	testCases := []struct {
		name      string
		ignore    bool
		err       error
		wantLevel string
		wantMsg   string
		wantError bool
	}{
		{
			name:      "ignored by default",
			ignore:    true,
			err:       gorm.ErrRecordNotFound,
			wantLevel: slog.LevelDebug.String(),
			wantMsg:   gormQueryLog,
		},
		{
			name:      "ignored when wrapped",
			ignore:    true,
			err:       ctxerrors.Wrap(gorm.ErrRecordNotFound, "find thing"),
			wantLevel: slog.LevelDebug.String(),
			wantMsg:   gormQueryLog,
		},
		{
			name:      "logged when not ignored",
			err:       gorm.ErrRecordNotFound,
			wantLevel: slog.LevelError.String(),
			wantMsg:   gormErrorLog,
			wantError: true,
		},
		{
			name:      "other errors still logged",
			ignore:    true,
			err:       assert.AnError,
			wantLevel: slog.LevelError.String(),
			wantMsg:   gormErrorLog,
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			records := captureGormRecords(t)

			gormLogger := NewGormSlogLogger()
			gormLogger.(*GormSlogLogger).IgnoreRecordNotFoundError = tc.ignore

			traceGormQuery(context.Background(), gormLogger, tc.err)

			logged := records()
			require.Len(t, logged, 1)
			assert.Equal(t, tc.wantLevel, logged[0][slog.LevelKey])
			assert.Equal(t, tc.wantMsg, logged[0][slog.MessageKey])

			_, hasError := logged[0][gormErrorKey]
			assert.Equal(t, tc.wantError, hasError)
		})
	}
}

func TestNewGormSlogLoggerIgnoresRecordNotFound(t *testing.T) {
	gormLogger, ok := NewGormSlogLogger().(*GormSlogLogger)
	require.True(t, ok)
	assert.True(t, gormLogger.IgnoreRecordNotFoundError)
	assert.Equal(t, logger.Info, gormLogger.LogLevel)

	quiet, ok := gormLogger.LogMode(logger.Warn).(*GormSlogLogger)
	require.True(t, ok)
	assert.True(t, quiet.IgnoreRecordNotFoundError)
}

// Every record carries the caller's ctxscope attributes.
func TestGormSlogLoggerCarriesTheContextScope(t *testing.T) {
	const (
		scopeKey   = "request_id"
		scopeValue = "request-7"
	)

	records := captureGormRecords(t)
	ctx := ctxscope.Set(context.Background(),
		ctxscope.Attr(scopeKey, scopeValue))

	gormLogger := NewGormSlogLogger()
	traceGormQuery(ctx, gormLogger, assert.AnError)
	gormLogger.Warn(ctx, "replacing callback %s", "create")

	logged := records()
	require.Len(t, logged, 2)

	for _, record := range logged {
		assert.Equal(t, scopeValue, record[scopeKey])
	}
}

// GORM's own messages log under a fixed message with the rendered text in
// detail, at the level GORM asked for.
func TestGormSlogLoggerMessages(t *testing.T) {
	const format = "replacing callback %s"

	records := captureGormRecords(t)
	gormLogger := NewGormSlogLogger()

	gormLogger.Info(context.Background(), format, "create")
	gormLogger.Warn(context.Background(), format, "update")
	gormLogger.Error(context.Background(), format, "delete")

	logged := records()
	require.Len(t, logged, 3)

	wantLevels := []string{
		slog.LevelInfo.String(),
		slog.LevelWarn.String(),
		slog.LevelError.String(),
	}
	wantDetails := []string{
		fmt.Sprintf(format, "create"),
		fmt.Sprintf(format, "update"),
		fmt.Sprintf(format, "delete"),
	}

	for index, record := range logged {
		assert.Equal(t, gormMessageLog, record[slog.MessageKey])
		assert.Equal(t, wantLevels[index], record[slog.LevelKey])
		assert.Equal(t, wantDetails[index], record[gormDetailKey])
	}
}
