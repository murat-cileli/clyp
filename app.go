package main

import (
	_ "embed"
	"os"
	"strconv"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	_ "github.com/mattn/go-sqlite3"
)

var (
	//go:embed main.ui
	uiXML     string
	database  Database
	clipboard Clipboard
)

type Application struct {
	clipboardItemsList *gtk.ListBox
	window             *gtk.ApplicationWindow
	name               string
	itemsVisibleLimit  int
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

	app.name = "Clyp"
	app.itemsVisibleLimit = 50

	builder := gtk.NewBuilderFromString(uiXML)
	app.window = builder.GetObject("gtk_window").Cast().(*gtk.ApplicationWindow)

	app.clipboardItemsList = builder.GetObject("clipboard_list").Cast().(*gtk.ListBox)
	app.clipboardItemsList.Activate()
	go app.listClipboardItems(true)
	//app.setupListBoxEvents()

	app.setupSearchBar(builder)

	app.setupStyleSupport()
	app.setupAboutAction(gtkApp, app.window)
	app.window.SetApplication(gtkApp)
	app.window.SetVisible(true)
}

func (app *Application) updateTitle(itemsShowing, itemsTotal string) {
	app.window.SetTitle(app.name + " - " + itemsShowing + " / " + itemsTotal)
}

func (app *Application) listClipboardItems(updateItemCount bool) {
	app.clipboardItemsList.RemoveAll()
	items, _ := clipboard.items(updateItemCount)

	if len(items) == 0 {
		// TODO: Boş veri mesajı ekle.
	}

	app.updateTitle(strconv.Itoa(len(items)), strconv.Itoa(clipboard.itemCount))

	for _, item := range items {
		box := gtk.NewBox(gtk.OrientationVertical, 6)
		box.SetMarginTop(12)
		box.SetMarginBottom(12)
		box.SetMarginStart(12)
		box.SetMarginEnd(12)

		contentLabel := gtk.NewLabel(item.content)
		contentLabel.SetWrap(true)
		contentLabel.SetWrapMode(pango.WrapWord)
		contentLabel.SetXAlign(0)
		contentLabel.AddCSSClass("title")

		dateLabel := gtk.NewLabel(item.dateTime)
		dateLabel.SetXAlign(0)
		dateLabel.AddCSSClass("subtitle")

		box.Append(contentLabel)
		box.Append(dateLabel)

		row := gtk.NewListBoxRow()
		row.SetName(string(item.id))
		row.SetChild(box)

		app.clipboardItemsList.Append(row)
	}
}

func (app *Application) setupSearchBar(builder *gtk.Builder) {
	searchEntry := builder.GetObject("search_entry").Cast().(*gtk.SearchEntry)
	searchEntry.ConnectSearchChanged(func() {
		database.searchFilter = searchEntry.Text()
		go app.listClipboardItems(false)
	})

	searchBar := builder.GetObject("search_bar").Cast().(*gtk.SearchBar)
	searchBar.ConnectEntry(searchEntry)

	searchToggleButton := builder.GetObject("search_toggle_button").Cast().(*gtk.ToggleButton)
	searchToggleButton.ConnectToggled(func() {
		searchBar.SetObjectProperty("search-mode-enabled", searchToggleButton.Active())
	})
}

func (app *Application) setupListBoxEvents() {
	app.clipboardItemsList.ConnectRowActivated(func(row *gtk.ListBoxRow) {
		//copyRowContentToClipboard(row)
	})

	keyController := gtk.NewEventControllerKey()
	keyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_Return || keyval == gdk.KEY_KP_Enter {
			selectedRow := app.clipboardItemsList.SelectedRow()
			if selectedRow != nil {
				//copyRowContentToClipboard(selectedRow)
				return true
			}
		}
		return false
	})

	app.clipboardItemsList.AddController(keyController)
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
