package main

import (
	"database/sql"
	"encoding/base64"
	"log"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type Clipboard struct {
	itemCount int
}

type ClipboardItem struct {
	id       int
	dateTime string
	content  string
	itemType byte
}

func (clipboard *Clipboard) items(updateItemCount bool, limit, offset int) ([]ClipboardItem, error) {
	var items []ClipboardItem
	var rows *sql.Rows
	var err error

	if updateItemCount {
		clipboard.count()
	}

	if database.searchFilter != "" {
		database.query = `
SELECT id, type, date_time, substr(content, 1, 101)
FROM clipboard
WHERE type = 1 AND content LIKE ?
ORDER BY date_time DESC, id DESC
LIMIT ? OFFSET ?`
		rows, err = database.db.Query(database.query, "%"+database.searchFilter+"%", limit, offset)
	} else {
		database.query = database.queryBase + " LIMIT ? OFFSET ?"
		rows, err = database.db.Query(database.query, limit, offset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item ClipboardItem
		if err := rows.Scan(&item.id, &item.itemType, &item.dateTime, &item.content); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (clipboard *Clipboard) count() {
	var rowTotalItemsCount *sql.Row
	if database.searchFilter != "" {
		rowTotalItemsCount = database.db.QueryRow(
			"SELECT COUNT(*) FROM clipboard WHERE type = 1 AND content LIKE ?",
			"%"+database.searchFilter+"%",
		)
	} else {
		rowTotalItemsCount = database.db.QueryRow("SELECT COUNT(*) FROM clipboard")
	}
	if err := rowTotalItemsCount.Scan(&clipboard.itemCount); err != nil {
		log.Printf("Failed to count clipboard items: %v", err)
		clipboard.itemCount = 0
	}
}

func (clipboard *Clipboard) enforceMaxItems() {
	if config.MaxClipboardItems <= 0 {
		return
	}

	_, err := database.db.Exec(`
DELETE FROM clipboard
WHERE id NOT IN (
	SELECT id
	FROM clipboard
	ORDER BY date_time DESC, id DESC
	LIMIT ?
)`, config.MaxClipboardItems)
	if err != nil {
		log.Printf("Failed to enforce clipboard item limit: %v", err)
	}
}

func (clipboard *Clipboard) copy(id string, gtkApp *gtk.Application) {
	if id == "" {
		return
	}

	var content string
	var itemType byte
	row := database.db.QueryRow("SELECT content, type FROM clipboard WHERE id=? LIMIT 1", id)
	row.Scan(&content, &itemType)

	clipboardInstance := gdk.DisplayGetDefault().Clipboard()

	switch itemType {
	case 1:
		clipboardInstance.SetText(content)
		clipboard.updateItemDateTime(id)
	case 2:
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			log.Printf("Failed to decode base64 image data: %v", err)
			return
		}
		texture, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(decoded))
		if err != nil {
			log.Printf("Failed to create texture from bytes: %v", err)
			return
		}
		clipboardInstance.SetTexture(texture)
		clipboard.updateItemDateTime(id)
	}

	clipboardInstance = nil
}

func (clipboard *Clipboard) updateItemDateTime(id string) {
	if id == "" {
		return
	}

	_, err := database.db.Exec("UPDATE clipboard SET date_time=CURRENT_TIMESTAMP WHERE id=?", id)
	if err != nil {
		log.Printf("Failed to update item date time: %v", err)
		return
	}
}

func (clipboard *Clipboard) removeFromDatabase(id string) {
	if id == "" {
		return
	}
	database.db.Exec("DELETE FROM clipboard WHERE id=?", id)
}

func (clipboard *Clipboard) removeAllFromDatabase() {
	_, err := database.db.Exec("DELETE FROM clipboard")
	if err != nil {
		log.Printf("Failed to clear clipboard database: %v", err)
		return
	}
	clipboard.itemCount = 0
}
