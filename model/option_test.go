package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCleanupDeprecatedOptions(t *testing.T) {
	oldDB := DB
	t.Cleanup(func() { DB = oldDB })

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	DB = db
	if err := DB.AutoMigrate(&Option{}); err != nil {
		t.Fatalf("migrate options: %v", err)
	}
	options := []Option{
		{Key: DeprecatedFrontendOptionKey, Value: "default"},
		{Key: "SystemName", Value: "New API"},
	}
	if err := DB.Create(&options).Error; err != nil {
		t.Fatalf("create options: %v", err)
	}

	if err := CleanupDeprecatedOptions(); err != nil {
		t.Fatalf("cleanup deprecated options: %v", err)
	}

	var deprecatedCount int64
	if err := DB.Model(&Option{}).Where("key = ?", DeprecatedFrontendOptionKey).Count(&deprecatedCount).Error; err != nil {
		t.Fatalf("count deprecated options: %v", err)
	}
	if deprecatedCount != 0 {
		t.Fatalf("deprecated option count = %d, want 0", deprecatedCount)
	}

	var systemName Option
	if err := DB.First(&systemName, "key = ?", "SystemName").Error; err != nil {
		t.Fatalf("load retained option: %v", err)
	}
	if systemName.Value != "New API" {
		t.Fatalf("retained option value = %q, want %q", systemName.Value, "New API")
	}
}
