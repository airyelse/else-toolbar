package database

import (
	"errors"
	"else-toolbox/internal/models"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init 初始化数据库
func Init(dataDir string) error {
	var err error
	dbPath := filepath.Join(dataDir, "vault.db")
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}

	// 自动迁移
	err = DB.AutoMigrate(
		&models.PasswordEntry{},
		&models.MasterKey{},
		&models.Category{},
		&models.Tag{},
	)
	if err != nil {
		return err
	}

	// 迁移旧数据：将 category 字符串迁移为 Category 记录
	migrateCategoryStrings()

	return nil
}

// migrateCategoryStrings 将 PasswordEntry 的旧 category 字符串字段迁移为 Category 记录
func migrateCategoryStrings() {
	// Newer databases no longer have the legacy `category` column.
	// Skip the migration entirely in that case so startup stays quiet.
	if !DB.Migrator().HasColumn(&models.PasswordEntry{}, "category") {
		return
	}

	tx := DB.Begin()
	if tx.Error != nil {
		return
	}

	// 用原生 SQL 查询旧字段（因为模型已改，GORM 无法映射旧 string 字段）
	type legacyRow struct {
		ID       uint
		Category string
	}
	var rows []legacyRow
	if err := tx.Raw("SELECT id, category FROM password_entries WHERE category != ''").Scan(&rows).Error; err != nil {
		tx.Rollback()
		return
	}
	if len(rows) == 0 {
		// 旧字段没有数据，直接删除
		if err := tx.Migrator().DropColumn(&models.PasswordEntry{}, "category"); err != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
		return
	}

	categoryMap := make(map[string]uint)
	for _, row := range rows {
		if _, ok := categoryMap[row.Category]; !ok {
			var cat models.Category
			result := tx.Where("name = ?", row.Category).First(&cat)
			if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				tx.Rollback()
				return
			}
			if result.RowsAffected == 0 {
				cat = models.Category{Name: row.Category, Order: len(categoryMap)}
				if err := tx.Create(&cat).Error; err != nil {
					tx.Rollback()
					return
				}
			}
			categoryMap[row.Category] = cat.ID
		}
		if err := tx.Exec("UPDATE password_entries SET category_id = ? WHERE id = ?", categoryMap[row.Category], row.ID).Error; err != nil {
			tx.Rollback()
			return
		}
	}

	if err := tx.Migrator().DropColumn(&models.PasswordEntry{}, "category"); err != nil {
		tx.Rollback()
		return
	}
	tx.Commit()
}

// Close 关闭数据库
func Close() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
