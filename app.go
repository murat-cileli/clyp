package main

import (
	_ "embed"
	"encoding/base64"
	"log"
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
	searchEntry        *gtk.SearchEntry
	searchBar          *gtk.SearchBar
	searchToggleButton *gtk.ToggleButton
	window             *gtk.ApplicationWindow
	name               string
	id                 string
	dataDir            string
}

func (app *Application) init() {
	app.id = "bio.murat.clyp"
	gtkApp := gtk.NewApplication(app.id, gio.ApplicationFlagsNone)
	gtkApp.ConnectActivate(func() { app.activate(gtkApp) })

	if code := gtkApp.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func (app *Application) activate(gtkApp *gtk.Application) {
	app.setupDataDir()

	if err := database.init(); err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	app.name = "Clyp"
	builder := gtk.NewBuilderFromString(uiXML)
	app.window = builder.GetObject("gtk_window").Cast().(*gtk.ApplicationWindow)
	app.clipboardItemsList = builder.GetObject("clipboard_list").Cast().(*gtk.ListBox)
	app.searchEntry = builder.GetObject("search_entry").Cast().(*gtk.SearchEntry)
	app.searchBar = builder.GetObject("search_bar").Cast().(*gtk.SearchBar)
	app.searchToggleButton = builder.GetObject("search_toggle_button").Cast().(*gtk.ToggleButton)
	app.updateClipboardRows(true)
	app.setupEvents()
	app.setupStyleSupport()
	app.setupAboutAction(gtkApp)
	app.window.SetApplication(gtkApp)
	app.window.SetVisible(true)
	app.window.SetIconName("bio.murat.clyp")
	clipboard.updateRecentContentFromDatabase()
	clipboard.watch()
}

func (app *Application) setupDataDir() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get user home dir: %v", err)
		os.Exit(1)
	}

	app.dataDir = homeDir + "/.local/share/" + app.id

	if _, err := os.Stat(app.dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(app.dataDir, 0755); err != nil {
			log.Printf("Failed to create data dir: %v", err)
			os.Exit(1)
		}
	}
}

func (app *Application) updateTitle(itemsShowing, itemsTotal string) {
	app.window.SetTitle(app.name + " - " + itemsShowing + " / " + itemsTotal)
}

func (app *Application) updateClipboardRows(updateItemCount bool) {
	app.clipboardItemsList.RemoveAll()
	items, err := clipboard.items(updateItemCount)
	if err != nil {
		log.Printf("Error getting clipboard items: %v", err)
		return
	}

	log.Printf("Retrieved %d items from database", len(items))
	app.updateTitle(strconv.Itoa(len(items)), strconv.Itoa(clipboard.itemCount))

	if len(items) == 0 {
		log.Printf("No items to display")
		return
	}

	for _, item := range items {
		switch item.itemType {
		case 1:
			app.addTextRow(item)
		case 2:
			app.addImageRow(item)
		default:
			log.Printf("Unknown item type: %d", item.itemType)
		}
	}
}

func (app *Application) addTextRow(item ClipboardItem) {
	box := gtk.NewBox(gtk.OrientationVertical, 6)
	box.SetMarginTop(12)
	box.SetMarginBottom(12)
	box.SetMarginStart(12)
	box.SetMarginEnd(12)

	if len(item.content) > 100 {
		item.content = item.content[:100] + "\n..."
	}
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
	row.SetName(strconv.Itoa(item.id))
	row.SetChild(box)

	app.clipboardItemsList.Append(row)
}

func (app *Application) addImageRow(item ClipboardItem) {
	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.SetMarginTop(0)
	box.SetMarginBottom(0)
	box.SetMarginStart(12)
	box.SetMarginEnd(12)

	if len(item.content) == 0 {
		log.Printf("Empty content for image item %d", item.id)
		image := gtk.NewImageFromIconName("image-missing")
		image.SetPixelSize(64)
		box.Append(image)
	} else {
		texture := app.loadImageFromBase64(item.content)
		if texture == nil {
			log.Printf("Failed to load image from base64 for item %d", item.id)
			image := gtk.NewImageFromIconName("image-missing")
			image.SetPixelSize(64)
			box.Append(image)
		} else {
			paintable := gdk.Paintabler(texture)
			image := gtk.NewImageFromPaintable(paintable)
			image.SetHAlign(gtk.AlignCenter)
			image.SetVAlign(gtk.AlignCenter)
			image.SetHExpand(false)
			image.SetVExpand(false)
			app.scaleImageToFit(image, texture, 300)
			box.Append(image)
		}
	}

	dateLabel := gtk.NewLabel(item.dateTime)
	dateLabel.SetXAlign(0)
	dateLabel.AddCSSClass("subtitle")
	box.Append(dateLabel)

	row := gtk.NewListBoxRow()
	row.SetName(strconv.Itoa(item.id))
	row.SetChild(box)

	app.clipboardItemsList.Append(row)
}

func (app *Application) loadImageFromBase64(base64Data string) *gdk.Texture {
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		log.Printf("Failed to decode base64 image data: %v", err)
		return nil
	}

	texture, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(imageData))
	if err != nil {
		return nil
	}

	return texture
}

func (app *Application) scaleImageToFit(image *gtk.Image, texture *gdk.Texture, maxSize int) {
	width := texture.Width()
	height := texture.Height()

	if height > maxSize {
		ratio := float64(maxSize) / float64(height)
		newWidth := int(float64(width) * ratio)
		newHeight := maxSize
		image.SetSizeRequest(newWidth, newHeight)
	} else if width > maxSize {
		ratio := float64(maxSize) / float64(width)
		newWidth := maxSize
		newHeight := int(float64(height) * ratio)
		image.SetSizeRequest(newWidth, newHeight)
	} else {
		image.SetSizeRequest(width, height)
	}
}

func (app *Application) setupEvents() {
	app.setupClipBoardListEvents()
	app.setupWindowEvents()
	app.setupSearchBarEvents()
}

func (app *Application) setupClipBoardListEvents() {
	clipboardListkeyController := gtk.NewEventControllerKey()

	clipboardListkeyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_Return || keyval == gdk.KEY_KP_Enter {
			selectedRow := app.clipboardItemsList.SelectedRow()
			if selectedRow != nil {
				clipboard.copy(selectedRow.Name())
				return true
			}
		}

		if keyval == gdk.KEY_Delete {
			selectedRow := app.clipboardItemsList.SelectedRow()
			if selectedRow != nil {
				clipboard.removeFromDatabase(selectedRow.Name())
			}
		}

		return false
	})

	app.clipboardItemsList.AddController(clipboardListkeyController)
}

func (app *Application) setupWindowEvents() {
	windowKeyController := gtk.NewEventControllerKey()

	windowKeyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if state&gdk.ControlMask != 0 && keyval == gdk.KEY_f {
			app.toggleSearchBar()
		}
		return false
	})

	app.window.AddController(windowKeyController)
}

func (app *Application) toggleSearchBar() {
	currentState := app.searchBar.ObjectProperty("search-mode-enabled").(bool)
	app.searchToggleButton.SetActive(!currentState)
	app.searchBar.SetObjectProperty("search-mode-enabled", !currentState)
}

func (app *Application) setupSearchBarEvents() {
	app.searchEntry.ConnectSearchChanged(func() {
		if app.searchEntry.Text() == "" {
			database.searchFilter = ""
			app.updateClipboardRows(false)
			app.toggleSearchBar()
			return
		}
		database.searchFilter = app.searchEntry.Text()
		app.updateClipboardRows(false)
	})
	app.searchBar.ConnectEntry(app.searchEntry)
	app.searchToggleButton.ConnectToggled(func() {
		app.searchBar.SetObjectProperty("search-mode-enabled", app.searchToggleButton.Active())
	})
	app.searchEntry.ConnectActivate(func() {
		if app.clipboardItemsList.RowAtIndex(0) == nil {
			return
		}
		app.clipboardItemsList.SelectRow(app.clipboardItemsList.RowAtIndex(0))
	})
	searchEntryKeyController := gtk.NewEventControllerKey()
	searchEntryKeyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_Escape {
			app.searchEntry.SetText("")
			app.toggleSearchBar()
			return true
		}
		return false
	})

	app.searchEntry.AddController(searchEntryKeyController)
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

func (app *Application) setupAboutAction(gtkApp *gtk.Application) {
	aboutAction := gio.NewSimpleAction("about", nil)
	aboutAction.ConnectActivate(func(parameter *glib.Variant) {
		app.showAboutDialog(app.window)
	})
	gtkApp.AddAction(aboutAction)
}

func (app *Application) showAboutDialog(parent *gtk.ApplicationWindow) {
	aboutDialog := gtk.NewAboutDialog()
	aboutDialog.SetTransientFor(&parent.Window)
	aboutDialog.SetLogoIconName("bio.murat.clyp")
	aboutDialog.SetModal(true)
	aboutDialog.SetVersion("0.9.0")
	aboutDialog.SetProgramName("Clyp")
	aboutDialog.SetCopyright("Developer: Murat Çileli\nIcon: Freepik from flaticon.com")
	aboutDialog.SetWebsite("https://github.com/murat-cileli/clyp")
	aboutDialog.SetWebsiteLabel("https://github.com/murat-cileli/clyp")
	aboutDialog.SetVisible(true)
}
