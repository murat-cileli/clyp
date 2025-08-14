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

var database Database

type Application struct {
	ClipboardItemsVisualLimit byte
	ClipboardItemsList        *gtk.ListBox
}

func (app *Application) init() {
	gtkApp := gtk.NewApplication("bio.murat.clyp", gio.ApplicationFlagsNone)
	gtkApp.ConnectActivate(func() { app.activate(gtkApp) })

	if code := gtkApp.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func (app *Application) activate(gtkApp *gtk.Application) {
	if err := database.init(); err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	builder := gtk.NewBuilderFromString(uiXML)
	window := builder.GetObject("GtkWindow").Cast().(*gtk.ApplicationWindow)

	app.ClipboardItemsList = builder.GetObject("clipboard_list").Cast().(*gtk.ListBox)
	app.ClipboardItemsList.Activate()
	go app.listClipboardItems()
	//app.setupListBoxEvents()

	searchEntry := builder.GetObject("search_entry").Cast().(*gtk.SearchEntry)
	searchEntry.ConnectActivate(func() {
		log.Println("Enter tuşuna basıldı, arama yapılabilir.")
	})
	searchEntry.ConnectSearchChanged(func() {
		log.Printf("Arama metni değişti: %s", searchEntry.Text())
	})

	app.ClipboardItemsVisualLimit = 50
	app.setupStyleSupport()
	app.setupAboutAction(gtkApp, window)
	window.SetApplication(gtkApp)
	window.SetVisible(true)
}

func (app *Application) listClipboardItems() {
	items, err := database.ClipboardItems()
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

		app.ClipboardItemsList.Append(row)
	}
}

func (app *Application) setupListBoxEvents() {
	app.ClipboardItemsList.ConnectRowActivated(func(row *gtk.ListBoxRow) {
		//copyRowContentToClipboard(row)
	})

	keyController := gtk.NewEventControllerKey()
	keyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_Return || keyval == gdk.KEY_KP_Enter {
			selectedRow := app.ClipboardItemsList.SelectedRow()
			if selectedRow != nil {
				//copyRowContentToClipboard(selectedRow)
				return true
			}
		}
		return false
	})

	app.ClipboardItemsList.AddController(keyController)
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

	aboutDialog.SetVersion("0.9")
	aboutDialog.SetProgramName("Clyp")
	aboutDialog.SetComments("Clipboard manager.")
	aboutDialog.SetCopyright("© 2025")
	aboutDialog.SetWebsite("https://murat.bio/")
	aboutDialog.SetWebsiteLabel("https://murat.bio")
	aboutDialog.SetLicense("MIT License")
	aboutDialog.SetAuthors([]string{"Murat Çileli"})

	aboutDialog.SetVisible(true)
}
