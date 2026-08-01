package database

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	migrationfiles "github.com/QUTCraft/qutc-platform/apps/api/migrations"
	"gorm.io/gorm"
)

const firstVersionedUpgrade = "009_organization_profiles.sql"

func runMigrations(db *gorm.DB) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
        version VARCHAR(255) PRIMARY KEY,
        applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`).Error; err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}

	entries, err := fs.ReadDir(migrationfiles.Files, ".")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			versions = append(versions, entry.Name())
		}
	}
	sort.Strings(versions)

	var appliedCount int64
	if err := db.Table("schema_migrations").Count(&appliedCount).Error; err != nil {
		return fmt.Errorf("count migrations: %w", err)
	}
	var organizationTableCount int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'organizations'").Scan(&organizationTableCount).Error; err != nil {
		return fmt.Errorf("inspect legacy schema: %w", err)
	}
	// Releases before the migration ledger used GORM AutoMigrate on every
	// startup. If such a database is detected, record the known legacy baseline
	// and apply only subsequent ordered migrations.
	if appliedCount == 0 && organizationTableCount > 0 {
		for _, version := range versions {
			if version >= firstVersionedUpgrade {
				break
			}
			if err := db.Exec("INSERT IGNORE INTO schema_migrations (version) VALUES (?)", version).Error; err != nil {
				return fmt.Errorf("record legacy migration %s: %w", version, err)
			}
		}
	}

	for _, version := range versions {
		var count int64
		if err := db.Table("schema_migrations").Where("version = ?", version).Count(&count).Error; err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if count > 0 {
			continue
		}
		contents, err := migrationfiles.Files.ReadFile(version)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}
		for _, statement := range strings.Split(string(contents), ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			if err := db.Exec(statement).Error; err != nil {
				return fmt.Errorf("apply migration %s: %w", version, err)
			}
		}
		if err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version).Error; err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
	}
	return nil
}
