package main

import (
	"database/sql"
)

type Clipboard struct {
	itemCount int
}

type ClipboardItem struct {
	id       int
	dateTime string
	content  string
}

func (clipboard *Clipboard) items(updateItemCount bool) ([]ClipboardItem, error) {
	var items []ClipboardItem
	var rows *sql.Rows
	var err error

	if database.searchFilter != "" {
		database.query = `SELECT id, content, date_time FROM clipboard WHERE type=1 AND content LIKE ? ORDER BY date_time DESC LIMIT 50`
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
		if err := rows.Scan(&item.id, &item.content, &item.dateTime); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if updateItemCount {
		clipboard.count()
	}

	return items, nil
}

func (clipboard *Clipboard) count() {
	rowTotalItemsCount := database.db.QueryRow("SELECT COUNT(*) as total_items FROM clipboard")
	rowTotalItemsCount.Scan(&clipboard.itemCount)
}
