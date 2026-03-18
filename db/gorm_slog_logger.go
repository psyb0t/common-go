package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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

func (l *GormSlogLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && l.LogLevel >= logger.Error:
		slog.Error("gorm error",
			"error", err,
			"duration", elapsed,
			"rows", rows,
			"sql", sql,
		)
	case elapsed > slowQueryThreshold && l.LogLevel >= logger.Warn:
		slog.Warn("SLOW QUERY",
			"duration", elapsed,
			"rows", rows,
			"sql", sql,
		)
	case l.LogLevel >= logger.Info:
		slog.Debug("SQL EXECUTED",
			"duration", elapsed,
			"rows", rows,
			"sql", sql,
		)
	}
}
