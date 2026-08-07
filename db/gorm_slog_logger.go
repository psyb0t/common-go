package db

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gorm.io/gorm/logger"
)

type GormSlogLogger struct {
	LogLevel logger.LogLevel
}

func NewGormSlogLogger() logger.Interface { //nolint:ireturn
	return &GormSlogLogger{
		LogLevel: logger.Info,
	}
}

func (l *GormSlogLogger) LogMode(level logger.LogLevel) logger.Interface { //nolint:ireturn
	newLogger := *l
	newLogger.LogLevel = level

	return &newLogger
}

func (l *GormSlogLogger) Info(_ context.Context, msg string, data ...any) {
	if l.LogLevel >= logger.Info {
		slog.Info(fmt.Sprintf(msg, data...))
	}
}

func (l *GormSlogLogger) Warn(_ context.Context, msg string, data ...any) {
	if l.LogLevel >= logger.Warn {
		slog.Warn(fmt.Sprintf(msg, data...))
	}
}

func (l *GormSlogLogger) Error(_ context.Context, msg string, data ...any) {
	if l.LogLevel >= logger.Error {
		slog.Error(fmt.Sprintf(msg, data...))
	}
}

const slowQueryThreshold = 200 * time.Millisecond

const maxSQLPreviewBytes = 2048

type sqlSummary struct {
	Preview   string
	Bytes     int
	Truncated bool
}

func summarizeSQL(query string) sqlSummary {
	normalized := redactSQLValues(query)
	preview, truncated := truncateUTF8(normalized, maxSQLPreviewBytes)

	return sqlSummary{
		Preview:   preview,
		Bytes:     len(query),
		Truncated: truncated,
	}
}

func redactSQLValues(query string) string {
	var out strings.Builder

	out.Grow(min(len(query), maxSQLPreviewBytes))

	lastWasSpace := false

	for i := 0; i < len(query); {
		r, size := utf8.DecodeRuneInString(query[i:])

		switch {
		case r == '\'':
			out.WriteByte('?')

			i = skipQuotedLiteral(query, i+size)
			lastWasSpace = false
		case unicode.IsSpace(r):
			// Leading whitespace is dropped rather than collapsed, so the
			// preview never opens with a space that TrimSpace would remove
			// anyway.
			if !lastWasSpace && out.Len() > 0 {
				out.WriteByte(' ')
			}

			i += size
			lastWasSpace = true
		case unicode.IsDigit(r):
			out.WriteByte('?')

			i = skipNumericLiteral(query, i+size)
			lastWasSpace = false
		default:
			out.WriteRune(r)

			i += size
			lastWasSpace = false
		}
	}

	return strings.TrimSpace(out.String())
}

// skipQuotedLiteral returns the index just past the single-quoted literal whose
// opening quote has already been consumed. Two consecutive single quotes are
// SQL's escape for a literal quote, so they continue the string rather than
// ending it — getting that wrong would close the literal early and spill the
// rest of a value into the preview unredacted.
func skipQuotedLiteral(query string, i int) int {
	for i < len(query) {
		next, size := utf8.DecodeRuneInString(query[i:])
		i += size

		if next != '\'' {
			continue
		}

		if i < len(query) && query[i] == '\'' {
			i++

			continue
		}

		return i
	}

	return i
}

// skipNumericLiteral returns the index just past the numeric literal whose
// first digit has already been consumed. A '.' is consumed as part of the
// number so a decimal is redacted whole rather than leaving a trailing
// fragment.
func skipNumericLiteral(query string, i int) int {
	for i < len(query) {
		next, size := utf8.DecodeRuneInString(query[i:])
		if !unicode.IsDigit(next) && next != '.' {
			return i
		}

		i += size
	}

	return i
}

func truncateUTF8(value string, maxBytes int) (string, bool) {
	if len(value) <= maxBytes {
		return value, false
	}

	end := maxBytes
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}

	return value[:end] + "…", true
}

func (l *GormSlogLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	summary := summarizeSQL(sql)
	attrs := []any{
		"duration", elapsed,
		"rows", rows,
		"sql_preview", summary.Preview,
		"sql_bytes", summary.Bytes,
		"sql_truncated", summary.Truncated,
	}

	switch {
	case err != nil && l.LogLevel >= logger.Error:
		slog.Error("gorm error", append([]any{"error", err}, attrs...)...)
	case elapsed > slowQueryThreshold && l.LogLevel >= logger.Warn:
		slog.Warn("SLOW QUERY", attrs...)
	case l.LogLevel >= logger.Info:
		slog.Debug("SQL EXECUTED", attrs...)
	}
}
