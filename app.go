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
	//go:embed resources/ui/main.ui
	uiXML string
	//go:embed resources/css/style.css
	cssData   string
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
	gtkApp.ConnectShutdown(func() { app.shutdown(gtkApp) })

	if code := gtkApp.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func (app *Application) activate(gtkApp *gtk.Application) {
	app.setupDataDir()

	if err := database.init(); err != nil {
		panic(err.Error())
	}

	app.name = "Clyp"
	builder := gtk.NewBuilderFromString(uiXML)
	app.window = builder.GetObject("gtk_window").Cast().(*gtk.ApplicationWindow)
	app.clipboardItemsList = builder.GetObject("clipboard_list").Cast().(*gtk.ListBox)
	app.searchEntry = builder.GetObject("search_entry").Cast().(*gtk.SearchEntry)
	app.searchBar = builder.GetObject("search_bar").Cast().(*gtk.SearchBar)
	app.searchToggleButton = builder.GetObject("search_toggle_button").Cast().(*gtk.ToggleButton)
	app.setupCSS()
	app.updateClipboardRows(true)
	database.vacuum()
	app.setupEvents(gtkApp)
	app.setupShortcutsAction(gtkApp)
	app.setupAboutAction(gtkApp)
	app.setupStyleSupport()
	app.window.SetApplication(gtkApp)
	app.window.SetVisible(true)
	app.window.SetIconName("bio.murat.clyp")
	clipboard.updateRecentContentFromDatabase()
	clipboard.watch()
}

func (app *Application) shutdown(gtkApp *gtk.Application) {
	confirmDialog := gtk.NewMessageDialog(
		gtkApp.ActiveWindow(),
		gtk.DialogModal|gtk.DialogDestroyWithParent,
		gtk.MessageQuestion,
		gtk.ButtonsYesNo,
	)
	confirmDialog.SetTitle("Warning")
	confirmDialog.Buildable.SetObjectProperty("text", "Clipboard will not be monitored.")
	confirmDialog.Buildable.SetObjectProperty("use-markup", true)
	confirmDialog.Buildable.SetObjectProperty("secondary-text", "Are you sure you want to quit?")
	confirmDialog.Buildable.SetObjectProperty("secondary-use-markup", true)
	confirmDialog.Buildable.SetObjectProperty("modal", true)
	confirmDialog.Buildable.SetObjectProperty("resizable", false)
	confirmDialog.Buildable.SetObjectProperty("deletable", false)
	confirmDialog.SetTransientFor(gtkApp.ActiveWindow())
	confirmDialog.SetVisible(true)
	confirmDialog.ConnectResponse(func(response int) {
		switch response {
		case int(gtk.ResponseYes):
			if database.db != nil {
				database.vacuum()
				database.db.Close()
			}
			gtkApp.Quit()
		case int(gtk.ResponseNo):
			confirmDialog.Close()
			return
		}
	})
}

func (app *Application) setupCSS() {
	if len(cssData) == 0 {
		return
	}

	cssProvider := gtk.NewCSSProvider()
	cssProvider.LoadFromString(cssData)

	display := gdk.DisplayGetDefault()
	gtk.StyleContextAddProviderForDisplay(
		display,
		cssProvider,
		gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
	)
}

func (app *Application) setupDataDir() {
	app.dataDir = glib.GetUserDataDir() + "/" + app.id

	if _, err := os.Stat(app.dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(app.dataDir, 0755); err != nil {
			panic(err.Error())
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

	app.updateTitle(strconv.Itoa(len(items)), strconv.Itoa(clipboard.itemCount))

	if len(items) == 0 {
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
	box.AddCSSClass("item-box")

	if len(item.content) > 100 {
		item.content = item.content[:100] + "\n..."
	}
	contentLabel := gtk.NewLabel(item.content)
	contentLabel.SetWrap(true)
	contentLabel.SetWrapMode(pango.WrapWordChar)
	contentLabel.SetXAlign(0)
	contentLabel.AddCSSClass("title")

	dateLabel := gtk.NewLabel(item.dateTime)
	dateLabel.SetXAlign(0)
	dateLabel.AddCSSClass("subtitle")

	box.Append(contentLabel)
	box.Append(dateLabel)

	row := gtk.NewListBoxRow()
	row.SetName(strconv.Itoa(item.id))
	row.AddCSSClass("item-row")
	row.SetChild(box)

	app.clipboardItemsList.Append(row)
}

func (app *Application) addImageRow(item ClipboardItem) {
	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.SetMarginTop(12)
	box.SetMarginBottom(12)
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
			image.AddCSSClass("item-image")
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
	newWidth := width
	newHeight := height

	if width > maxSize || height > maxSize {
		var ratio float64
		if width > height {
			ratio = float64(maxSize) / float64(width)
		} else {
			ratio = float64(maxSize) / float64(height)
		}
		newWidth = int(float64(width) * ratio)
		newHeight = int(float64(height) * ratio)
		image.SetSizeRequest(newWidth, newHeight)
	}
}

func (app *Application) setupEvents(gtkApp *gtk.Application) {
	app.setupAppEvents(gtkApp)
	app.setupClipBoardListEvents()
	app.setupWindowEvents()
	app.setupSearchBarEvents()
}

func (app *Application) setupAppEvents(gtkApp *gtk.Application) {
	app.window.ConnectCloseRequest(func() bool {
		app.shutdown(gtkApp)
		return true
	})
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
				return true
			}
		}

		if keyval == gdk.KEY_Escape {
			app.closeSearchBar()
			return true
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
			return true
		}
		return true
	})

	app.window.AddController(windowKeyController)
}

func (app *Application) toggleSearchBar() {
	currentState := app.searchBar.ObjectProperty("search-mode-enabled").(bool)
	app.searchToggleButton.SetActive(!currentState)
	app.searchBar.SetObjectProperty("search-mode-enabled", !currentState)
}

func (app *Application) closeSearchBar() {
	if app.searchBar.ObjectProperty("search-mode-enabled").(bool) {
		app.searchToggleButton.SetActive(false)
		app.searchBar.SetObjectProperty("search-mode-enabled", false)
		if app.clipboardItemsList.RowAtIndex(0) != nil {
			app.clipboardItemsList.SelectRow(app.clipboardItemsList.RowAtIndex(0))
		}
	}
}

func (app *Application) setupSearchBarEvents() {
	app.searchEntry.ConnectSearchChanged(func() {
		if app.searchEntry.Text() == "" {
			database.searchFilter = ""
			app.updateClipboardRows(false)
			app.closeSearchBar()
			return
		}
		database.searchFilter = app.searchEntry.Text()
		app.updateClipboardRows(false)
	})
	app.searchBar.ConnectEntry(app.searchEntry)
	app.searchToggleButton.ConnectToggled(func() {
		app.toggleSearchBar()
	})
	app.searchEntry.ConnectActivate(func() {
		if app.clipboardItemsList.RowAtIndex(0) != nil {
			app.clipboardItemsList.SelectRow(app.clipboardItemsList.RowAtIndex(0))
		}
	})
	searchEntryKeyController := gtk.NewEventControllerKey()
	searchEntryKeyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_Escape {
			app.closeSearchBar()
			return true
		}
		if keyval == gdk.KEY_Down || keyval == gdk.KEY_KP_Down || keyval == gdk.KEY_Tab || keyval == gdk.KEY_KP_Tab {
			if app.clipboardItemsList.RowAtIndex(0) != nil {
				app.clipboardItemsList.SelectRow(app.clipboardItemsList.RowAtIndex(0))
				app.clipboardItemsList.GrabFocus()
			}
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

func (app *Application) setupShortcutsAction(gtkApp *gtk.Application) {
	shortcutsAction := gio.NewSimpleAction("shortcuts", nil)
	shortcutsAction.ConnectActivate(func(parameter *glib.Variant) {
		app.showShortcutsWindow(app.window)
	})
	gtkApp.AddAction(shortcutsAction)
}

func (app *Application) showShortcutsWindow(parent *gtk.ApplicationWindow) {
	builder := gtk.NewBuilderFromString(uiXML)
	shortcutsWindow := builder.GetObject("shortcuts").Cast().(*gtk.ShortcutsWindow)
	shortcutsWindow.SetTransientFor(&parent.Window)
	shortcutsWindow.SetModal(true)
	shortcutsWindow.SetVisible(true)
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
