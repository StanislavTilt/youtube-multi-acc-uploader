package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Open(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	DB.SetMaxOpenConns(1)
	return DB.Ping()
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}

func RunMigrations(migrationsDir string) error {

	_, err := DB.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
		filename TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, fname := range files {
		var applied int
		DB.QueryRow("SELECT COUNT(*) FROM _migrations WHERE filename = ?", fname).Scan(&applied)
		if applied > 0 {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, fname))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", fname, err)
		}

		tx, err := DB.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", fname, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", fname, err)
		}
		if _, err := tx.Exec("INSERT INTO _migrations (filename) VALUES (?)", fname); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", fname, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", fname, err)
		}
		log.Printf("Applied migration: %s", fname)
	}
	return nil
}
