package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"log"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

type Clipboard struct {
	clipboard     gdk.Clipboard
	itemCount     int
	recentContent string
}

type ClipboardItem struct {
	id       int
	dateTime string
	content  string
	itemType byte
}

func (clipboard *Clipboard) items(updateItemCount bool) ([]ClipboardItem, error) {
	var items []ClipboardItem
	var rows *sql.Rows
	var err error

	if database.searchFilter != "" {
		database.query = `SELECT id, type, date_time, content FROM clipboard WHERE type=1 AND content LIKE ? ORDER BY date_time DESC LIMIT 30`
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
		if err := rows.Scan(&item.id, &item.itemType, &item.dateTime, &item.content); err != nil {
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

func (clipboard *Clipboard) watch() {
	clipboard.clipboard = *gdk.DisplayGetDefault().Clipboard()
	clipboard.clipboard.ConnectChanged(func() {
		formats := clipboard.clipboard.Formats().String()
		if formats == "" {
			return
		}
		if strings.Contains(formats, "text/") {
			clipboard.readTextContent()
		} else if strings.Contains(formats, "image/") {
			clipboard.readImageContent()
		} else {
			log.Printf("Unsupported clipboard format: %s", formats)
		}
	})
}

func (clipboard *Clipboard) readTextContent() {
	clipboard.clipboard.ReadTextAsync(context.Background(), func(result gio.AsyncResulter) {
		text, err := clipboard.clipboard.ReadTextFinish(result)
		if err != nil {
			return
		}
		text = strings.TrimSpace(text)
		if text != "" {
			clipboard.saveToDatabase(text, 1)
		}
	})
}

func (clipboard *Clipboard) readImageContent() {
	clipboard.clipboard.ReadTextureAsync(context.Background(), func(result gio.AsyncResulter) {
		texture, err := clipboard.clipboard.ReadTextureFinish(result)
		if err != nil || texture == nil {
			return
		}

		imageData := clipboard.textureToBase64(texture)

		if imageData == "" {
			return
		}

		clipboard.saveToDatabase(imageData, 2)
	})
}

func (clipboard *Clipboard) textureToBase64(texture gdk.Texturer) string {
	var pngBytes *glib.Bytes

	if memTexture, ok := texture.(*gdk.MemoryTexture); ok {
		pngBytes = memTexture.SaveToPNGBytes()
	} else if gdkTexture, ok := texture.(*gdk.Texture); ok {
		pngBytes = gdkTexture.SaveToPNGBytes()
	} else {
		if textureSaver, ok := texture.(interface{ SaveToPNGBytes() *glib.Bytes }); ok {
			pngBytes = textureSaver.SaveToPNGBytes()
		} else {
			return ""
		}
	}

	if pngBytes == nil {
		return ""
	}

	pngData := pngBytes.Data()
	if len(pngData) == 0 {
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(pngData)

	return encoded
}

func (clipboard *Clipboard) updateRecentContentFromDatabase() {
	contentRow := database.db.QueryRow("SELECT content FROM clipboard ORDER BY id DESC LIMIT 1")
	contentRow.Scan(&clipboard.recentContent)
}

func (clipboard *Clipboard) saveToDatabase(content string, itemType byte) {
	if len(content) == 0 || content == clipboard.recentContent {
		return
	}

	if itemType == 2 {
		database.db.Exec("DELETE FROM clipboard WHERE type=2")
	}

	_, err := database.db.Exec("INSERT INTO clipboard (content, type) VALUES (?, ?)", content, itemType)
	if err == nil {
		clipboard.recentContent = content
		ipc.notify()
	}
}

func (clipboard *Clipboard) copy(id string) {
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
