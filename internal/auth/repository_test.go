package auth

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestRepository_AutoMigrate_RenamesUsersTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			email TEXT NOT NULL,
			phone TEXT,
			password_hash TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			must_reset_pwd BOOLEAN NOT NULL DEFAULT FALSE,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("创建旧 users 表失败: %v", err)
	}

	repo := NewRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	if db.Migrator().HasTable("users") {
		t.Fatal("旧 users 表应已被重命名")
	}
	if !db.Migrator().HasTable("platform_users") {
		t.Fatal("platform_users 表应存在")
	}
}
