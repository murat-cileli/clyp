package main

import (
	"database/sql"
	"log"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var app Application

// ClipboardItem veritabanından okunan clipboard verisini temsil eder
type ClipboardItem struct {
	ID          int
	Type        int
	IsPinned    int
	IsEncrypted int
	Content     string
	DateTime    string
}

func main() {
	app.init()
}

// copyRowContentToClipboard seçili satırın içeriğini panoya kopyalar
func copyRowContentToClipboard(row *gtk.ListBoxRow) {
	if row == nil {
		return
	}

	// Row'un child widget'ını al (Box)
	child := row.Child()
	if child == nil {
		return
	}

	// Box'ı cast et
	box, ok := child.(*gtk.Box)
	if !ok {
		return
	}

	// Box'ın ilk child'ını al (content label)
	firstChild := box.FirstChild()
	if firstChild == nil {
		return
	}

	// Label'ı cast et
	contentLabel, ok := firstChild.(*gtk.Label)
	if !ok {
		return
	}

	// Label'ın text'ini al
	content := contentLabel.Text()
	if content == "" {
		return
	}

	// Clipboard'a kopyala (Wayland uyumlu)
	display := gdk.DisplayGetDefault()
	if display != nil {
		clipboard := display.Clipboard()
		if clipboard != nil {
			clipboard.SetText(content)
			log.Printf("Panoya kopyalandı: %s", content)
		}
	}
}

// getClipboardItems veritabanından clipboard tablosundaki content ve date_time verilerini alır
func getClipboardItems() ([]ClipboardItem, error) {
	// Veritabanı dosyasının yolu
	dbPath := "./clipboard.db"

	// Veritabanı bağlantısını aç
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Veritabanı bağlantısını test et
	if err := db.Ping(); err != nil {
		return nil, err
	}

	rows, err := db.Query(`
		SELECT *
		FROM clipboard
		ORDER BY date_time DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clipboardItems []ClipboardItem
	var clipboardItem ClipboardItem
	for rows.Next() {
		if err := rows.Scan(&clipboardItem.ID, &clipboardItem.Type, &clipboardItem.DateTime, &clipboardItem.IsPinned, &clipboardItem.Content, &clipboardItem.IsEncrypted); err != nil {
			continue
		}
		clipboardItems = append(clipboardItems, clipboardItem)
	}

	return clipboardItems, nil
}

// createClipboardTable clipboard tablosunu oluşturur
func createClipboardTable(db *sql.DB) error {
	query := `
	CREATE TABLE clipboard (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := db.Exec(query)
	return err
}
