package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/glebarez/sqlite"

	"go-fiber-starter/internal/config"
)

// New는 설정 기반으로 *gorm.DB를 생성하고 연결을 검증한다.
func New(ctx context.Context, cfg *config.Config, logger *Logger) (*gorm.DB, error) {
	var dial gorm.Dialector
	switch cfg.DBDriver {
	case config.DBDriverPostgres:
		dial = postgres.Open(cfg.DBDSN)
	case config.DBDriverMySQL:
		dial = mysql.Open(cfg.DBDSN)
	case config.DBDriverSQLite:
		abs, err := filepath.Abs(cfg.DBDSN)
		if err != nil {
			return nil, err
		}
		// 파일 경로의 부모 디렉터리가 없으면 sqlite 오픈이 실패한다(클론 직후 실행 대응).
		// sqlite 파일은 dev 전용 로컬 데이터다. 타 사용자 traversal 차단을 위해 디렉터리만 0o700로 충분하다.
		if dir := filepath.Dir(abs); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return nil, fmt.Errorf("create sqlite directory: %w", err)
			}
		}
		dial = sqlite.Open(abs)
	default:
		return nil, errors.New("unsupported DB driver: " + string(cfg.DBDriver))
	}

	db, err := gorm.Open(dial, &gorm.Config{
		Logger: logger,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.DBConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, err
	}
	return db, nil
}

// Ping은 readiness 체크용으로 사용한다.
func Ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return sqlDB.PingContext(ctx)
}

// WithTx는 트랜잭션 경계 헬퍼다. fn이 error를 반환하면 롤백한다.
//
// 저장소 어댑터 안에서 요청 context를 DB에 적용해 호출한다.
// 서비스에는 *gorm.DB 대신 트랜잭션 작업을 나타내는 인터페이스를 노출한다.
//
//	database.WithTx(r.db.WithContext(ctx), func(tx *gorm.DB) error { return repoTx(tx).Do(ctx) })
func WithTx(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}
