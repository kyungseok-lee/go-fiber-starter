package database

import (
	"context"
	"strings"
	"testing"
	"time"

	"go-fiber-starter/internal/config"
	"go-fiber-starter/internal/modules/task"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func testConfig(driver config.DBDriver, env config.Env, autoMigrate bool) *config.Config {
	return &config.Config{
		Env:                  env,
		DBDriver:             driver,
		DBDSN:                "unused-by-policy-tests",
		DBAutoMigrate:        autoMigrate,
		DBSlowQueryThreshold: time.Second,
	}
}

// 정책 경로는 AutoMigrate 호출 전에 반환되므로 실DB 없이 검증 가능해야 한다.
func TestAutoMigrateIfNeeded_ProdPostgres_HardBlocked(t *testing.T) {
	cfg := testConfig(config.DBDriverPostgres, config.EnvProd, true)

	err := AutoMigrateIfNeeded(context.Background(), cfg, nil, &task.Task{})
	if err == nil || !strings.Contains(err.Error(), "DB_AUTO_MIGRATE") {
		t.Fatalf("prod pg automigrate must be blocked, got %v", err)
	}
}

func TestAutoMigrateIfNeeded_Disabled_Noop(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(config.DBDriverSQLite, config.EnvLocal, false)
	cfg.DBDSN = dir + "/app.db"

	db, err := gorm.Open(sqlite.Open(cfg.DBDSN), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrateIfNeeded(context.Background(), cfg, db, &task.Task{}); err != nil {
		t.Fatalf("disabled flag must noop, got %v", err)
	}
	if db.Migrator().HasTable(&task.Task{}) {
		t.Fatal("disabled auto-migrate created tasks table")
	}
}

func TestAutoMigrateIfNeeded_SQLiteDev_CreatesTables(t *testing.T) {
	dir := t.TempDir()
	cfg := testConfig(config.DBDriverSQLite, config.EnvLocal, true)
	cfg.DBDSN = dir + "/app.db"

	db, err := gorm.Open(sqlite.Open(cfg.DBDSN), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrateIfNeeded(context.Background(), cfg, db, &task.Task{}); err != nil {
		t.Fatalf("sqlite dev should migrate: %v", err)
	}
	if !db.Migrator().HasTable(&task.Task{}) {
		t.Fatal("tasks table not created")
	}
}
