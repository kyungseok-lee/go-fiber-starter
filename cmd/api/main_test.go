package main

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"go-fiber-starter/internal/config"
)

func TestNewLogger_LogLevelCaseInsensitive(t *testing.T) {
	for _, tc := range []struct {
		name    string
		minimum slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
	} {
		for _, input := range []string{tc.name, strings.ToUpper(tc.name)} {
			t.Run(input, func(t *testing.T) {
				log := newLogger(&config.Config{LogLevel: input})
				for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
					want := level >= tc.minimum
					if got := log.Enabled(context.Background(), level); got != want {
						t.Errorf("LOG_LEVEL=%q: Enabled(%s) = %v, want %v", input, level, got, want)
					}
				}
			})
		}
	}
}
