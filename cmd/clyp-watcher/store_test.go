package main

import (
	"crypto/sha256"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSaveTreatsUniqueConflictAsRecentContent(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
CREATE TABLE clipboard (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	"type" INTEGER DEFAULT (1) NOT NULL,
	date_time TEXT DEFAULT '2026-10-09 20:14:13' NOT NULL,
	content TEXT NOT NULL
);
CREATE UNIQUE INDEX clipboard_content_IDX ON clipboard (content, date_time);
INSERT INTO clipboard (content, type) VALUES ('duplicate', 1);
`)
	if err != nil {
		t.Fatal(err)
	}

	store := &historyStore{db: db, maxItems: 500}
	if err := store.Save("duplicate", 1); err != nil {
		t.Fatalf("Save returned a unique constraint error: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM clipboard").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one stored row, got %d", count)
	}
	if !store.hasRecentValue || store.recentType != 1 || store.recentHash != sha256.Sum256([]byte("duplicate")) {
		t.Fatal("ignored duplicate was not recorded as the recent clipboard value")
	}
}
