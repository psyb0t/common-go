package db

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.LessOrEqual(t, len(got.Preview), maxSQLPreviewBytes+len("…"))
		})
	}
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
