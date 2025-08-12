package main

import (
	"os"

	_ "embed"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
)

//go:embed main.ui
var uiXML string

func main() {
	app := gtk.NewApplication("com.github.diamondburned.gotk4-examples.gtk4.simple", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() { activate(app) })

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate(app *gtk.Application) {
	builder := gtk.NewBuilderFromString(uiXML)
	window := builder.GetObject("GtkWindow").Cast().(*gtk.ApplicationWindow)

	// GNOME tema desteği için ayarları yapılandır
	setupThemeSupport()

	// ListBox'ı al ve örnek içerik ekle
	listBox := builder.GetObject("clipboard_list").Cast().(*gtk.ListBox)
	setupListBox(listBox)

	window.SetApplication(app)
	window.Show()
}

// setupListBox ListBox'a örnek içerik ekler
func setupListBox(listBox *gtk.ListBox) {
	// Örnek liste öğeleri
	items := []string{
		"Hoş geldiniz! Bu uygulama GNOME tema desteği ile gelir.",
		"Sistem temanızı değiştirdiğinizde uygulama otomatik olarak güncellenir.",
		"Karanlık tema için: gsettings set org.gnome.desktop.interface color-scheme 'prefer-dark'",
		"Açık tema için: gsettings set org.gnome.desktop.interface color-scheme 'default'",
		"Bu liste GTK4'ün modern tasarım dilini kullanır.",
	}

	for _, item := range items {
		// Her öğe için bir label oluştur
		label := gtk.NewLabel(item)
		label.SetWrap(true)
		label.SetWrapMode(pango.WrapWord)
		label.SetXAlign(0) // Sol hizalama
		label.SetMarginTop(12)
		label.SetMarginBottom(12)
		label.SetMarginStart(12)
		label.SetMarginEnd(12)

		// ListBoxRow oluştur ve label'ı ekle
		row := gtk.NewListBoxRow()
		row.SetChild(label)

		listBox.Append(row)
	}
}

// setupThemeSupport GNOME'un karanlık/açık tema tercihini destekler
func setupThemeSupport() {
	// GTK ayarlarını al
	gtkSettings := gtk.SettingsGetDefault()
	if gtkSettings == nil {
		return
	}

	// GNOME'un tema tercihini kontrol et
	// Bu, gsettings'den org.gnome.desktop.interface color-scheme değerini okur
	// Değerler: 'default' (açık), 'prefer-dark' (karanlık)

	// GNOME desktop interface ayarlarını al
	gnomeSettings := gio.NewSettings("org.gnome.desktop.interface")
	if gnomeSettings != nil {
		// GNOME color-scheme ayarını oku
		colorScheme := gnomeSettings.String("color-scheme")

		// Tema tercihini ayarla
		preferDark := colorScheme == "prefer-dark"
		gtkSettings.SetObjectProperty("gtk-application-prefer-dark-theme", preferDark)

		// GNOME ayarlarındaki değişiklikleri dinle
		gnomeSettings.Connect("changed::color-scheme", func() {
			newColorScheme := gnomeSettings.String("color-scheme")
			newPreferDark := newColorScheme == "prefer-dark"
			gtkSettings.SetObjectProperty("gtk-application-prefer-dark-theme", newPreferDark)
		})
	} else {
		// GNOME ayarları mevcut değilse, GTK tema adından çıkarım yap
		gtkSettings.SetObjectProperty("gtk-application-prefer-dark-theme", false) // Varsayılan olarak açık tema
	}

	// Sistem tema değişikliklerini dinle
	gtkSettings.Connect("notify::gtk-theme-name", func() {
		// Tema değiştiğinde gerekli işlemleri yap
		handleThemeChange(gtkSettings)
	})

	// Karanlık tema tercihini de dinle
	gtkSettings.Connect("notify::gtk-application-prefer-dark-theme", func() {
		// Karanlık tema tercihi değiştiğinde
		handleThemeChange(gtkSettings)
	})

	// İlk tema kontrolü
	handleThemeChange(gtkSettings)
}

// handleThemeChange tema değişikliklerini işler
func handleThemeChange(settings *gtk.Settings) {
	// Mevcut tema adını al
	themeName := settings.ObjectProperty("gtk-theme-name")
	if themeNameStr, ok := themeName.(string); ok {
		// Adwaita-dark teması karanlık tema tercihini gösterir
		if themeNameStr == "Adwaita-dark" {
			settings.SetObjectProperty("gtk-application-prefer-dark-theme", true)
		} else if themeNameStr == "Adwaita" {
			// Sistem tema tercihini kontrol et
			// GNOME'da gsettings get org.gnome.desktop.interface color-scheme
			// komutu ile kontrol edilebilir
			settings.SetObjectProperty("gtk-application-prefer-dark-theme", false)
		}
	}
}
