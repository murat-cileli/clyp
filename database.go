package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db           *sql.DB
	query        string
	queryBase    string
	searchFilter string
}

func (database *Database) init() error {
	database.searchFilter = ""
	database.queryBase = "SELECT id, type, date_time, content FROM clipboard ORDER BY date_time DESC LIMIT 50"
	if err := database.connect(); err != nil {
		return err
	}

	return nil
}

func (database *Database) connect() error {
	dbPath := app.dataDir + "/clyp.db"

	var err error
	database.db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	if err := database.db.Ping(); err != nil {
		return err
	}

	database.create()

	return nil
}

func (database *Database) create() {
	database.db.Exec(`
CREATE TABLE clipboard (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	"type" INTEGER DEFAULT (1) NOT NULL,
	date_time TEXT DEFAULT (CURRENT_TIMESTAMP) NOT NULL,
	content TEXT NOT NULL
);
CREATE INDEX clipboard_type_IDX ON clipboard ("type",content);
CREATE UNIQUE INDEX clipboard_content_IDX ON clipboard (content,date_time);
`)
}

func (database *Database) vacuum() {
	database.db.Exec(`VACUUM;`)
}
