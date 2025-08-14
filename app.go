package main

import (
	_ "embed"
	"log"
	"os"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed main.ui
var uiXML string

type Application struct {
}

func (app *Application) init() {
	gtkApp := gtk.NewApplication("com.github.diamondburned.gotk4-examples.gtk4.simple", gio.ApplicationFlagsNone)
	gtkApp.ConnectActivate(func() { app.activate(gtkApp) })

	if code := gtkApp.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func (app *Application) activate(gtkApp *gtk.Application) {
	builder := gtk.NewBuilderFromString(uiXML)
	window := builder.GetObject("GtkWindow").Cast().(*gtk.ApplicationWindow)

	listBox := builder.GetObject("clipboard_list").Cast().(*gtk.ListBox)
	app.setupListBox(listBox)
	app.setupListBoxEvents(listBox)

	app.setupStyleSupport()
	app.setupAboutAction(gtkApp, window)
	window.SetApplication(gtkApp)
	window.SetVisible(true)
}

func (app *Application) setupListBox(listBox *gtk.ListBox) {
	items, err := getClipboardItems()
	if err != nil {
		log.Printf("Veritabanından veri alınırken hata: %v", err)
		return
	}

	if len(items) == 0 {
		// TODO: Boş veri mesajı ekle.
	}

	for i := 0; i < len(items); i++ {
		box := gtk.NewBox(gtk.OrientationVertical, 6)
		box.SetMarginTop(12)
		box.SetMarginBottom(12)
		box.SetMarginStart(12)
		box.SetMarginEnd(12)

		contentLabel := gtk.NewLabel(items[i].Content)
		contentLabel.SetWrap(true)
		contentLabel.SetWrapMode(pango.WrapWord)
		contentLabel.SetXAlign(0)
		contentLabel.AddCSSClass("title")

		dateLabel := gtk.NewLabel(items[i].DateTime)
		dateLabel.SetXAlign(0)
		dateLabel.AddCSSClass("subtitle")

		box.Append(contentLabel)
		box.Append(dateLabel)

		row := gtk.NewListBoxRow()
		row.SetChild(box)

		listBox.Append(row)
	}
}

func (app *Application) setupListBoxEvents(listBox *gtk.ListBox) {
	listBox.ConnectRowActivated(func(row *gtk.ListBoxRow) {
		copyRowContentToClipboard(row)
	})

	keyController := gtk.NewEventControllerKey()
	keyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
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

func (app *Application) setupStyleSupport() {
	gtkSettings := gtk.SettingsGetDefault()
	gnomeSettings := gio.NewSettings("org.gnome.desktop.interface")
	app.handleStyleChange(gtkSettings, gnomeSettings)
	gnomeSettings.Connect("changed::color-scheme", func() {
		app.handleStyleChange(gtkSettings, gnomeSettings)
	})
}

func (app *Application) handleStyleChange(gtkSettings *gtk.Settings, gnomeSettings *gio.Settings) {
	gtkSettings.SetObjectProperty("gtk-application-prefer-dark-theme", gnomeSettings.String("color-scheme") == "prefer-dark")
}

func (app *Application) setupAboutAction(gtkApp *gtk.Application, window *gtk.ApplicationWindow) {
	aboutAction := gio.NewSimpleAction("about", nil)
	aboutAction.ConnectActivate(func(parameter *glib.Variant) {
		app.showAboutDialog(window)
	})
	gtkApp.AddAction(aboutAction)
}

func (app *Application) showAboutDialog(parent *gtk.ApplicationWindow) {
	aboutDialog := gtk.NewAboutDialog()
	aboutDialog.SetTransientFor(&parent.Window)
	aboutDialog.SetLogoIconName("clipboard")
	aboutDialog.SetModal(true)

	aboutDialog.SetProgramName("Clipto")
	aboutDialog.SetVersion("1.0.0")
	aboutDialog.SetComments("Clipboard manager.")
	aboutDialog.SetCopyright("© 2025")
	aboutDialog.SetWebsite("https://murat.bio/")
	aboutDialog.SetWebsiteLabel("https://murat.bio")
	aboutDialog.SetLicense("MIT License")
	aboutDialog.SetAuthors([]string{"Murat Çileli"})

	aboutDialog.SetVisible(true)
}
