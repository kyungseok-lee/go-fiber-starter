package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"go-fiber-starter/internal/config"
	"go-fiber-starter/internal/database"
	"go-fiber-starter/internal/modules/task"
	"go-fiber-starter/internal/router"
)

// commit은 Dockerfile 빌드 시 -ldflags "-X main.commit=..."로 주입된다.
var commit = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := newLogger(cfg)
	slog.SetDefault(log)
	log.Info("starting",
		"commit", commit,
		"app", cfg.AppName, "env", cfg.Env, "port", cfg.Port, "db_driver", cfg.DBDriver,
	)
	// 빌드 식별자를 Config로 주입해 router 이하가 cfg만 의존하게 한다(시그니처 무변경).
	cfg.BuildCommit = commit

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.New(ctx, cfg, database.NewLogger(log, log.Enabled(context.Background(), slog.LevelDebug), cfg.DBSlowQueryThreshold))
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
			log.Info("database connections closed")
		}
	}()

	if err := database.AutoMigrateIfNeeded(ctx, cfg, db, &task.Task{}); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	app := router.New(cfg, db, log, newMetricsRegistry())

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Info("http server listening", "addr", addr)

	// GracefulContext가 SIGTERM/SIGINT를 받으면 Fiber가 신규 수신을 끊고
	// 진행 중 요청 완료 후 Listen이 반환된다(ShutdownTimeout 초과 시 에러 반환).
	err = app.Listen(addr, fiber.ListenConfig{
		GracefulContext: ctx,
		ShutdownTimeout: cfg.ShutdownGracePeriod,
	})
	if err != nil {
		return fmt.Errorf("server: %w", err)
	}
	log.Info("shut down gracefully")
	return nil
}

func newLogger(cfg *config.Config) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}

// newMetricsRegistry는 앱 전용 Prometheus 레지스트리를 만든다.
// Go 런타임/프로세스 지표를 포함해 /metrics 하나로 운영 관측이 가능하게 한다.
func newMetricsRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return reg
}
