package middleware_test

import "log/slog"

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
