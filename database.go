package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	Db           *sql.DB
	Query        string
	SearchFilter string
}

type ClipboardItem struct {
	ID       int
	Type     byte
	DateTime string
	Content  string
}

func (database *Database) init() error {
	database.SearchFilter = ""
	database.Query = "SELECT content, date_time FROM clipboard ORDER BY date_time DESC LIMIT 50"
	if err := database.connect(); err != nil {
		return err
	}
	return nil
}

func (database *Database) connect() error {
	dbPath := "./clipboard.db"

	var err error
	database.Db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	if err := database.Db.Ping(); err != nil {
		return err
	}

	return nil
}

func (database *Database) ClipboardItems() ([]ClipboardItem, error) {
	var items []ClipboardItem
	rows, err := database.Db.Query(database.Query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item ClipboardItem
		if err := rows.Scan(&item.Content, &item.DateTime); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}
