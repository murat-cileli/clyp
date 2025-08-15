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

type ClipboardItem struct {
	id       int
	itemType byte
	dateTime string
	content  string
}

func (database *Database) init() error {
	database.searchFilter = ""
	database.queryBase = "SELECT content, date_time FROM clipboard ORDER BY date_time DESC LIMIT 50"
	if err := database.connect(); err != nil {
		return err
	}
	return nil
}

func (database *Database) connect() error {
	dbPath := "./clipboard.db"

	var err error
	database.db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	if err := database.db.Ping(); err != nil {
		return err
	}

	return nil
}

func (database *Database) ClipboardItems() ([]ClipboardItem, error) {
	var items []ClipboardItem

	var rows *sql.Rows
	var err error

	if database.searchFilter != "" {
		database.query = `SELECT content, date_time FROM clipboard WHERE type=1 AND content LIKE ? ORDER BY date_time DESC LIMIT 50`
		rows, err = database.db.Query(database.query, "%"+database.searchFilter+"%")
	} else {
		database.query = database.queryBase
		rows, err = database.db.Query(database.query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item ClipboardItem
		if err := rows.Scan(&item.content, &item.dateTime); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}
