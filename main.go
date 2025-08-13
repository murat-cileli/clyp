package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed main.ui
var uiXML string

//go:embed style.css
var cssData string

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
	app := gtk.NewApplication("com.github.diamondburned.gotk4-examples.gtk4.simple", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() { activate(app) })

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate(app *gtk.Application) {
	// CSS stillerini yükle
	loadCSS()

	builder := gtk.NewBuilderFromString(uiXML)
	window := builder.GetObject("GtkWindow").Cast().(*gtk.ApplicationWindow)

	// GNOME tema desteği için ayarları yapılandır
	setupStyleSupport()

	// ListBox'ı al ve örnek içerik ekle
	listBox := builder.GetObject("clipboard_list").Cast().(*gtk.ListBox)
	setupListBox(listBox)

	// ListBox için event handler'ları ekle
	setupListBoxEvents(listBox)

	window.SetApplication(app)
	window.Show()
}

// loadCSS CSS stillerini yükler
func loadCSS() {
	cssProvider := gtk.NewCSSProvider()
	cssProvider.LoadFromData(cssData)

	display := gdk.DisplayGetDefault()
	if display != nil {
		gtk.StyleContextAddProviderForDisplay(display, cssProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	}
}

// setupListBox ListBox'ı veritabanından gelen verilerle doldurur
func setupListBox(listBox *gtk.ListBox) {
	items, err := getClipboardItems()
	if err != nil {
		log.Printf("Veritabanından veri alınırken hata: %v", err)
		return
	}

	if len(items) == 0 {
		// TODO: Boş veri mesajı ekle.
	}

	for _, item := range items {
		// Her öğe için bir Box oluştur (dikey)
		box := gtk.NewBox(gtk.OrientationVertical, 6)
		box.SetMarginTop(12)
		box.SetMarginBottom(12)
		box.SetMarginStart(12)
		box.SetMarginEnd(12)

		// Ana içerik label'ı
		contentLabel := gtk.NewLabel(item.Content)
		contentLabel.SetWrap(true)
		contentLabel.SetWrapMode(pango.WrapWord)
		contentLabel.SetXAlign(0) // Sol hizalama
		contentLabel.SetSelectable(true)
		contentLabel.AddCSSClass("content-label")

		// Tarih label'ı (alt başlık)
		dateLabel := gtk.NewLabel(formatDateTime(item.DateTime))
		dateLabel.SetXAlign(0) // Sol hizalama
		dateLabel.AddCSSClass("dim-label")
		dateLabel.AddCSSClass("caption")

		// Box'a label'ları ekle
		box.Append(contentLabel)
		box.Append(dateLabel)

		// ListBoxRow oluştur ve box'ı ekle
		row := gtk.NewListBoxRow()
		row.SetChild(box)

		listBox.Append(row)
	}
}

// setupListBoxEvents ListBox için event handler'ları ayarlar
func setupListBoxEvents(listBox *gtk.ListBox) {
	// Çift tıklama ve Enter tuşu için row-activated sinyalini dinle
	listBox.ConnectRowActivated(func(row *gtk.ListBoxRow) {
		copyRowContentToClipboard(row)
	})

	// Klavye event'leri için key controller ekle
	keyController := gtk.NewEventControllerKey()
	keyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		// Enter tuşuna basıldığında seçili satırı kopyala
		if keyval == gdk.KEY_Return || keyval == gdk.KEY_KP_Enter {
			selectedRow := listBox.SelectedRow()
			if selectedRow != nil {
				copyRowContentToClipboard(selectedRow)
				return true // Event'i consume et
			}
		}
		return false // Event'i başka handler'lara geçir
	})

	listBox.AddController(keyController)
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
		LIMIT 30
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clipboardItems []ClipboardItem
	var clipboardItem ClipboardItem
	for rows.Next() {
		if err := rows.Scan(&clipboardItem.ID, &clipboardItem.Type, &clipboardItem.DateTime, &clipboardItem.IsPinned, &clipboardItem.Content, &clipboardItem.IsEncrypted); err != nil {
			log.Printf("Satır okunurken hata: %v", err)
			continue
		}
		clipboardItems = append(clipboardItems, clipboardItem)
	}

	return clipboardItems, nil
}

// formatDateTime tarih string'ini kullanıcı dostu formata çevirir
func formatDateTime(dateTimeStr string) string {
	// SQLite'dan gelen tarih formatını parse et
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.000",
	}

	var parsedTime time.Time
	var err error

	for _, layout := range layouts {
		parsedTime, err = time.Parse(layout, dateTimeStr)
		if err == nil {
			break
		}
	}

	if err != nil {
		// Parse edilemezse orijinal string'i döndür
		return dateTimeStr
	}

	// Türkçe tarih formatında döndür
	now := time.Now()
	diff := now.Sub(parsedTime)

	if diff < time.Minute {
		return "Az önce"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		return fmt.Sprintf("%d dakika önce", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%d saat önce", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d gün önce", days)
	} else {
		return parsedTime.Format("02.01.2006 15:04")
	}
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

func setupStyleSupport() {
	gtkSettings := gtk.SettingsGetDefault()
	gnomeSettings := gio.NewSettings("org.gnome.desktop.interface")

	gnomeSettings.Connect("changed::color-scheme", func() {
		handleStyleChange(gtkSettings, gnomeSettings)
	})

	handleStyleChange(gtkSettings, gnomeSettings)
}

func handleStyleChange(gtkSettings *gtk.Settings, gnomeSettings *gio.Settings) {
	colorScheme := gnomeSettings.String("color-scheme")
	preferDark := colorScheme == "prefer-dark"
	gtkSettings.SetObjectProperty("gtk-application-prefer-dark-theme", preferDark)
}
