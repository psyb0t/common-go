package commontemporal

import "log/slog"

type SlogAdapter struct {
	logger *slog.Logger
}

func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
	if logger == nil {
		logger = slog.Default()
	}

	return &SlogAdapter{logger: logger}
}

func (l *SlogAdapter) Debug(msg string, keyvals ...any) {
	l.logger.Debug(msg, keyvals...)
}

func (l *SlogAdapter) Info(msg string, keyvals ...any) {
	l.logger.Info(msg, keyvals...)
}

func (l *SlogAdapter) Warn(msg string, keyvals ...any) {
	l.logger.Warn(msg, keyvals...)
}

func (l *SlogAdapter) Error(msg string, keyvals ...any) {
	l.logger.Error(msg, keyvals...)
}
