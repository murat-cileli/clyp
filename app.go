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
	window             *gtk.ApplicationWindow
	name               string
	dataDir            string
}

func (app *Application) init() {
	gtkApp := gtk.NewApplication("bio.murat.clyp", gio.ApplicationFlagsNone)
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
	app.updateClipboardRows(true)
	app.setupClipBoardListEvents()
	app.setupSearchBar(builder)
	app.setupStyleSupport()
	app.setupAboutAction(gtkApp, app.window)
	app.window.SetApplication(gtkApp)
	app.window.SetVisible(true)
	clipboard.updateRecentContentFromDatabase()
	clipboard.watch()
}

func (app *Application) setupDataDir() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get user home dir: %v", err)
		os.Exit(1)
	}

	app.dataDir = homeDir + "/.local/share/clyp"

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
	box.SetMarginEnd(0)

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
			image.SetHAlign(gtk.AlignFill)
			image.SetVAlign(gtk.AlignFill)
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

func (app *Application) scaleImageToFit(image *gtk.Image, texture *gdk.Texture, maxHeight int) {
	width := texture.Width()
	height := texture.Height()

	if height > maxHeight {
		ratio := float64(maxHeight) / float64(height)
		newWidth := int(float64(width) * ratio)
		newHeight := maxHeight
		image.SetSizeRequest(newWidth, newHeight)
	} else {
		image.SetSizeRequest(width, height)
	}
}

func (app *Application) setupSearchBar(builder *gtk.Builder) {
	searchEntry := builder.GetObject("search_entry").Cast().(*gtk.SearchEntry)
	searchEntry.ConnectSearchChanged(func() {
		database.searchFilter = searchEntry.Text()
		app.updateClipboardRows(false)
	})

	searchBar := builder.GetObject("search_bar").Cast().(*gtk.SearchBar)
	searchBar.ConnectEntry(searchEntry)

	searchToggleButton := builder.GetObject("search_toggle_button").Cast().(*gtk.ToggleButton)
	searchToggleButton.ConnectToggled(func() {
		searchBar.SetObjectProperty("search-mode-enabled", searchToggleButton.Active())
	})
}

func (app *Application) setupClipBoardListEvents() {
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

		if keyval == gdk.KEY_Delete {
			selectedRow := app.clipboardItemsList.SelectedRow()
			if selectedRow != nil {
				clipboard.removeFromDatabase(selectedRow.Name())
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
