package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type watcherConfig struct {
	MaxClipboardItems int `json:"max_clipboard_items"`
}

type historyStore struct {
	db             *sql.DB
	maxItems       int
	recentHash     [sha256.Size]byte
	recentType     byte
	hasRecentValue bool
}

func openHistory() (*historyStore, error) {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get user home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".local", "share")
	}
	dataDir = filepath.Join(dataDir, appID)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	db, err := sql.Open("sqlite3", filepath.Join(dataDir, "clyp.db")+"?_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	store := &historyStore{db: db, maxItems: loadMaxItems()}
	if err := store.initialize(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (store *historyStore) initialize() error {
	if _, err := store.db.Exec(`
CREATE TABLE IF NOT EXISTS clipboard (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	"type" INTEGER DEFAULT (1) NOT NULL,
	date_time TEXT DEFAULT (CURRENT_TIMESTAMP) NOT NULL,
	content TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS clipboard_type_IDX ON clipboard ("type",content);
CREATE UNIQUE INDEX IF NOT EXISTS clipboard_content_IDX ON clipboard (content,date_time);
`); err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}

	var content string
	var itemType byte
	err := store.db.QueryRow("SELECT content, type FROM clipboard ORDER BY id DESC LIMIT 1").Scan(&content, &itemType)
	if err == nil {
		store.recentHash = sha256.Sum256([]byte(content))
		store.recentType = itemType
		store.hasRecentValue = true
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read latest clipboard item: %w", err)
	}
	return store.enforceLimit()
}

func (store *historyStore) Save(content string, itemType byte) error {
	contentHash := sha256.Sum256([]byte(content))
	if store.hasRecentValue && store.recentType == itemType && store.recentHash == contentHash {
		return nil
	}

	tx, err := store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`
INSERT INTO clipboard (content, type) VALUES (?, ?)
ON CONFLICT(content, date_time) DO NOTHING`, content, itemType)
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted > 0 && itemType == 2 {
		if _, err := tx.Exec("DELETE FROM clipboard WHERE type = 2 AND id NOT IN (SELECT id FROM clipboard WHERE type = 2 ORDER BY date_time DESC, id DESC LIMIT 3)"); err != nil {
			return err
		}
	}
	if store.maxItems > 0 {
		if _, err := tx.Exec(`
DELETE FROM clipboard
WHERE id NOT IN (
	SELECT id FROM clipboard ORDER BY date_time DESC, id DESC LIMIT ?
)`, store.maxItems); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	store.recentHash = contentHash
	store.recentType = itemType
	store.hasRecentValue = true
	if inserted > 0 {
		notifyGUI()
	}
	return nil
}

func (store *historyStore) enforceLimit() error {
	if store.maxItems <= 0 {
		return nil
	}
	_, err := store.db.Exec(`
DELETE FROM clipboard
WHERE id NOT IN (
	SELECT id FROM clipboard ORDER BY date_time DESC, id DESC LIMIT ?
)`, store.maxItems)
	return err
}

func (store *historyStore) Close() error {
	return store.db.Close()
}

func loadMaxItems() int {
	const defaultMaxItems = 500
	configDir, err := os.UserConfigDir()
	if err != nil {
		return defaultMaxItems
	}
	configData, err := os.ReadFile(filepath.Join(configDir, appID, "config.json"))
	if err != nil {
		return defaultMaxItems
	}
	var config watcherConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		log.Printf("Failed to parse config: %v", err)
		return defaultMaxItems
	}
	if config.MaxClipboardItems <= 0 {
		return defaultMaxItems
	}
	return config.MaxClipboardItems
}

func notifyGUI() {
	conn, err := net.DialTimeout("unix", filepath.Join(os.TempDir(), "clyp.sock"), 50*time.Millisecond)
	if err != nil {
		return
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("1")); err != nil {
		log.Printf("Failed to notify GUI: %v", err)
	}
}
