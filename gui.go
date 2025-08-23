package main

import (
	_ "embed"
	"encoding/base64"
	"log"
	"os"
	"os/exec"
	"strconv"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
)

var (
	//go:embed resources/ui/main.ui
	uiXML string
	//go:embed resources/css/style.css
	cssData string
	//go:embed resources/clyp-watcher.desktop
	watcherFile string
)

type GUI struct {
	clipboardItemsList *gtk.ListBox
	searchEntry        *gtk.SearchEntry
	searchBar          *gtk.SearchBar
	searchToggleButton *gtk.ToggleButton
	window             *gtk.ApplicationWindow
}

func (gui *GUI) init() {
	gtkApp := gtk.NewApplication(app.id, gio.ApplicationDefaultFlags)
	gtkApp.ConnectActivate(func() { gui.activate(gtkApp) })
	gtkApp.ConnectShutdown(func() { gui.shutdown(gtkApp) })
	gtkApp.ConnectAfter("activate", func() {
		go ipc.listen()
		cmd := "clyp"
		if os.Getenv("RUN_ENV") == "dev" {
			cmd = "./clyp"
		}
		watcher := *exec.Command(cmd, "watch")
		if err := watcher.Start(); err != nil {
			log.Println(err.Error())
		}
	})

	if code := gtkApp.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func (gui *GUI) activate(gtkApp *gtk.Application) {
	app.setupDataDir()
	if err := database.init(); err != nil {
		panic(err.Error())
	}
	builder := gtk.NewBuilderFromString(uiXML)
	gui.window = builder.GetObject("gtk_window").Cast().(*gtk.ApplicationWindow)
	gui.clipboardItemsList = builder.GetObject("clipboard_list").Cast().(*gtk.ListBox)
	gui.searchEntry = builder.GetObject("search_entry").Cast().(*gtk.SearchEntry)
	gui.searchBar = builder.GetObject("search_bar").Cast().(*gtk.SearchBar)
	gui.searchToggleButton = builder.GetObject("search_toggle_button").Cast().(*gtk.ToggleButton)
	gui.setupCSS()
	glib.IdleAdd(func() {
		gui.updateClipboardRows(true)
		gui.focusFirstClipboardListItem()
	})
	gui.setupEvents(gtkApp)
	gui.setupShortcutsAction(gtkApp)
	gui.setupAboutAction(gtkApp)
	gui.setupActionRunOnStartup(gtkApp)
	gui.setupStyleSupport()
	gui.window.SetApplication(gtkApp)
	gui.window.SetVisible(true)
	gui.window.SetIconName("bio.murat.clyp")
}

func (gui *GUI) shutdown(gtkApp *gtk.Application) {
	if database.db != nil {
		database.vacuum()
		database.db.Close()
	}
	gtkApp.Quit()
}

func (gui *GUI) setupCSS() {
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

func (gui *GUI) updateTitle(itemsShowing, itemsTotal string) {
	gui.window.SetTitle(app.name + " - " + itemsShowing + " / " + itemsTotal)
}

func (gui *GUI) updateClipboardRows(updateItemCount bool) {
	gui.clipboardItemsList.RemoveAll()
	items, err := clipboard.items(updateItemCount)
	if err != nil {
		log.Printf("Error getting clipboard items: %v", err)
		return
	}

	gui.updateTitle(strconv.Itoa(len(items)), strconv.Itoa(clipboard.itemCount))

	if len(items) == 0 {
		return
	}

	for _, item := range items {
		switch item.itemType {
		case 1:
			gui.addTextRow(item)
		case 2:
			gui.addImageRow(item)
		default:
			log.Printf("Unknown item type: %d", item.itemType)
		}
	}
}

func (gui *GUI) addTextRow(item ClipboardItem) {
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

	gui.clipboardItemsList.Append(row)
}

func (gui *GUI) addImageRow(item ClipboardItem) {
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
		texture := gui.loadImageFromBase64(item.content)
		if texture == nil {
			log.Printf("Failed to load image from base64 for item %d", item.id)
			image := gtk.NewImageFromIconName("image-missing")
			image.SetPixelSize(64)
			box.Append(image)
		} else {
			paintable := gdk.Paintabler(texture)
			image := gtk.NewImageFromPaintable(paintable)
			image.AddCSSClass("item-image")
			gui.scaleImageToFit(image, texture, 300)
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

	gui.clipboardItemsList.Append(row)
}

func (gui *GUI) loadImageFromBase64(base64Data string) *gdk.Texture {
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

func (gui *GUI) scaleImageToFit(image *gtk.Image, texture *gdk.Texture, maxSize int) {
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

func (gui *GUI) setupEvents(gtkApp *gtk.Application) {
	gui.setupAppEvents(gtkApp)
	gui.setupClipBoardListEvents()
	gui.setupWindowEvents()
	gui.setupSearchBarEvents()
}

func (gui *GUI) setupAppEvents(gtkApp *gtk.Application) {
	gui.window.ConnectCloseRequest(func() bool {
		gui.shutdown(gtkApp)
		return true
	})
}

func (gui *GUI) setupClipBoardListEvents() {
	clipboardListkeyController := gtk.NewEventControllerKey()

	clipboardListkeyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_Return || keyval == gdk.KEY_KP_Enter {
			selectedRow := gui.clipboardItemsList.SelectedRow()
			if selectedRow != nil {
				gui.closeSearchBar()
				clipboard.copy(selectedRow.Name())
				glib.IdleAdd(func() {
					gui.updateClipboardRows(true)
					gui.focusFirstClipboardListItem()
				})
				return true
			}
		}

		if keyval == gdk.KEY_Delete {
			selectedRow := gui.clipboardItemsList.SelectedRow()
			if selectedRow != nil {
				clipboard.removeFromDatabase(selectedRow.Name())
				glib.IdleAdd(func() {
					gui.updateClipboardRows(true)
					gui.focusFirstClipboardListItem()
				})
				return true
			}
		}

		if keyval == gdk.KEY_Escape {
			gui.closeSearchBar()
			return true
		}

		return false
	})

	gestureClick := gtk.NewGestureClick()

	gestureClick.ConnectPressed(func(nPress int, x, y float64) {
		if nPress == 2 {
			selectedRow := gui.clipboardItemsList.SelectedRow()
			if selectedRow != nil {
				gui.closeSearchBar()
				clipboard.copy(selectedRow.Name())
				glib.IdleAdd(func() {
					gui.updateClipboardRows(true)
					gui.focusFirstClipboardListItem()
				})
			}
		}
	})

	gui.clipboardItemsList.AddController(clipboardListkeyController)
	gui.clipboardItemsList.AddController(gestureClick)

}

func (gui *GUI) setupWindowEvents() {
	windowKeyController := gtk.NewEventControllerKey()

	windowKeyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if state&gdk.ControlMask != 0 && keyval == gdk.KEY_f {
			gui.toggleSearchBar()
			return true
		}
		return true
	})

	gui.window.AddController(windowKeyController)
}

func (gui *GUI) toggleSearchBar() {
	currentState := gui.searchBar.ObjectProperty("search-mode-enabled").(bool)
	gui.searchToggleButton.SetActive(!currentState)
	gui.searchBar.SetObjectProperty("search-mode-enabled", !currentState)
}

func (gui *GUI) closeSearchBar() {
	if gui.searchBar.ObjectProperty("search-mode-enabled").(bool) {
		gui.searchToggleButton.SetActive(false)
		gui.searchBar.SetObjectProperty("search-mode-enabled", false)
		gui.focusFirstClipboardListItem()
	}
}

func (gui *GUI) setupSearchBarEvents() {
	gui.searchEntry.ConnectSearchChanged(func() {
		if gui.searchEntry.Text() == "" {
			database.searchFilter = ""
			glib.IdleAdd(func() {
				gui.updateClipboardRows(true)
				gui.focusFirstClipboardListItem()
			})
			gui.closeSearchBar()
			return
		}
		database.searchFilter = gui.searchEntry.Text()
		glib.IdleAdd(func() {
			gui.updateClipboardRows(true)
		})
	})
	gui.searchBar.ConnectEntry(gui.searchEntry)
	gui.searchToggleButton.ConnectToggled(func() {
		gui.toggleSearchBar()
	})
	gui.searchEntry.ConnectActivate(func() {
		gui.focusFirstClipboardListItem()
	})
	searchEntryKeyController := gtk.NewEventControllerKey()
	searchEntryKeyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_Escape {
			gui.closeSearchBar()
			return true
		}
		if keyval == gdk.KEY_Down || keyval == gdk.KEY_KP_Down || keyval == gdk.KEY_Tab || keyval == gdk.KEY_KP_Tab {
			gui.focusFirstClipboardListItem()
			return true
		}
		return false
	})

	gui.searchEntry.AddController(searchEntryKeyController)
}

func (gui *GUI) focusFirstClipboardListItem() {
	if gui.clipboardItemsList.RowAtIndex(0) == nil {
		return
	}
	firstItem := gui.clipboardItemsList.RowAtIndex(0)
	gui.clipboardItemsList.SelectRow(firstItem)
	gtk.ListBoxRow(*firstItem).Cast().(*gtk.ListBoxRow).GrabFocus()

}

func (gui *GUI) setupStyleSupport() {
	gtkSettings := gtk.SettingsGetDefault()
	gnomeSettings := gio.NewSettings("org.gnome.desktop.interface")
	gui.handleStyleChange(gtkSettings, gnomeSettings)
	gnomeSettings.Connect("changed::color-scheme", func() {
		gui.handleStyleChange(gtkSettings, gnomeSettings)
	})
}

func (gui *GUI) handleStyleChange(gtkSettings *gtk.Settings, gnomeSettings *gio.Settings) {
	gtkSettings.SetObjectProperty("gtk-application-prefer-dark-theme", gnomeSettings.String("color-scheme") == "prefer-dark")
}

func (gui *GUI) setupShortcutsAction(gtkApp *gtk.Application) {
	shortcutsAction := gio.NewSimpleAction("shortcuts", nil)
	shortcutsAction.ConnectActivate(func(parameter *glib.Variant) {
		gui.showShortcutsWindow(gui.window)
	})
	gtkApp.AddAction(shortcutsAction)
}

func (gui *GUI) showShortcutsWindow(parent *gtk.ApplicationWindow) {
	builder := gtk.NewBuilderFromString(uiXML)
	shortcutsWindow := builder.GetObject("shortcuts").Cast().(*gtk.ShortcutsWindow)
	shortcutsWindow.SetTransientFor(&parent.Window)
	shortcutsWindow.SetModal(true)
	shortcutsWindow.SetVisible(true)
}

func (gui *GUI) setupAboutAction(gtkApp *gtk.Application) {
	aboutAction := gio.NewSimpleAction("about", nil)
	aboutAction.ConnectActivate(func(parameter *glib.Variant) {
		gui.showAboutDialog(gui.window)
	})
	gtkApp.AddAction(aboutAction)
}

func (gui *GUI) showAboutDialog(parent *gtk.ApplicationWindow) {
	aboutDialog := gtk.NewAboutDialog()
	aboutDialog.SetTransientFor(&parent.Window)
	aboutDialog.SetLogoIconName("bio.murat.clyp")
	aboutDialog.SetModal(true)
	aboutDialog.SetVersion("0.9.2")
	aboutDialog.SetProgramName("Clyp")
	aboutDialog.SetComments("Clipboard manager.")
	aboutDialog.SetWebsite("https://github.com/murat-cileli/clyp")
	aboutDialog.SetWebsiteLabel("https://github.com/murat-cileli/clyp")
	aboutDialog.SetVisible(true)
}

func (gui *GUI) setupActionRunOnStartup(gtkApp *gtk.Application) {
	hasStartupEntry := gui.hasStartupEntry()
	initialState := glib.NewVariantBoolean(hasStartupEntry)
	actionRunOnStartup := gio.NewSimpleActionStateful("run_on_startup", nil, initialState)
	actionRunOnStartup.ConnectActivate(func(parameter *glib.Variant) {
		gui.handleRunOnStartup(actionRunOnStartup)
	})
	gtkApp.AddAction(actionRunOnStartup)
	if !hasStartupEntry {
		glib.IdleAdd(func() {
			glib.TimeoutAdd(1000, func() bool {
				gui.showAddToStartupToast()
				return false
			})
		})
	}
}

func (gui *GUI) handleRunOnStartup(action *gio.SimpleAction) {
	currentState := action.State().Boolean()
	newState := glib.NewVariantBoolean(!currentState)
	action.SetState(newState)
	if newState.Boolean() {
		gui.addStartupEntry()
	} else {
		gui.removeStartupEntry()
	}
}

func (gui *GUI) addStartupEntry() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get user home directory: %v", err)
		return
	}
	autoStartdesktopFile := userHomeDir + "/.config/autostart/clyp-watcher.desktop"
	err = os.WriteFile(autoStartdesktopFile, []byte(watcherFile), 0644)
	if err != nil {
		log.Printf("Failed to write desktop file: %v", err)
	}
}

func (gui *GUI) removeStartupEntry() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get user home directory: %v", err)
		return
	}
	autoStartdesktopFile := userHomeDir + "/.config/autostart/clyp-watcher.desktop"
	err = os.Remove(autoStartdesktopFile)
	if err != nil {
		log.Printf("Failed to remove desktop file: %v", err)
	}
}

func (gui *GUI) hasStartupEntry() bool {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get user home directory: %v", err)
		return false
	}
	autoStartdesktopFile := userHomeDir + "/.config/autostart/clyp-watcher.desktop"
	_, err = os.Stat(autoStartdesktopFile)
	if err == nil {
		return true
	}
	return false
}

func (gui *GUI) showAddToStartupToast() {
	revealer := gtk.NewRevealer()
	revealer.SetTransitionType(gtk.RevealerTransitionTypeSlideDown)
	revealer.SetTransitionDuration(300)

	toastBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	toastBox.SetHAlign(gtk.AlignCenter)
	toastBox.SetMarginTop(10)
	toastBox.SetMarginBottom(10)
	toastBox.SetMarginStart(20)
	toastBox.SetMarginEnd(20)
	toastBox.AddCSSClass("toast")

	label := gtk.NewLabel("Go the menu to add Clyp to the system startup.")
	label.SetHAlign(gtk.AlignCenter)
	toastBox.Append(label)

	closeButton := gtk.NewButtonFromIconName("window-close-symbolic")
	closeButton.SetHasFrame(false)
	toastBox.Append(closeButton)

	revealer.SetChild(toastBox)

	mainBox := gui.window.Child().(*gtk.Box)
	mainBox.Prepend(revealer)

	revealer.SetRevealChild(true)

	glib.TimeoutAdd(3000, func() bool {
		revealer.SetRevealChild(false)
		glib.TimeoutAdd(300, func() bool {
			mainBox.Remove(revealer)
			return false
		})
		return false
	})

	closeButton.ConnectClicked(func() {
		revealer.SetRevealChild(false)
		glib.TimeoutAdd(300, func() bool {
			mainBox.Remove(revealer)
			return false
		})
	})
}
