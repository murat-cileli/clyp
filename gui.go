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
	runOnStartupAction *gio.SimpleAction
	closeOnCopyAction  *gio.SimpleAction
}

func (gui *GUI) init() {
	gtkApp := gtk.NewApplication(app.id, gio.ApplicationDefaultFlags)
	gtkApp.ConnectActivate(func() { gui.activate(gtkApp) })
	gtkApp.ConnectShutdown(func() { gui.shutdown(gtkApp) })

	if code := gtkApp.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func (gui *GUI) activate(gtkApp *gtk.Application) {
	builder := gtk.NewBuilderFromString(uiXML)
	gui.window = builder.GetObject("gtk_window").Cast().(*gtk.ApplicationWindow)
	gui.clipboardItemsList = builder.GetObject("clipboard_list").Cast().(*gtk.ListBox)
	gui.searchEntry = builder.GetObject("search_entry").Cast().(*gtk.SearchEntry)
	gui.searchBar = builder.GetObject("search_bar").Cast().(*gtk.SearchBar)
	gui.searchToggleButton = builder.GetObject("search_toggle_button").Cast().(*gtk.ToggleButton)
	gui.window.SetApplication(gtkApp)
	gui.setupCSS()
	gui.updateClipboardRows(true)
	gui.window.SetVisible(true)
	if config.FocusWindowOnOpen {
		gui.window.Present()
		gui.focusClipboardItemByIndex(0)
	}
	gui.setupEvents(gtkApp)
	gui.setupShortcutsAction(gtkApp)
	gui.setupSettingsAction(gtkApp)
	gui.setupAboutAction(gtkApp)
	gui.setupActionRunOnStartup(gtkApp)
	gui.setupCloseOnCopy(gtkApp)
	gui.setupStyleSupport()
	gui.window.SetIconName(app.id)
	// Always update startup entry.
	if gui.startupEntryControl("check") {
		gui.startupEntryControl("add")
	}
	gui.startWatcher()
	go ipc.listen()
}

func (gui *GUI) shutdown(gtkApp *gtk.Application) {
	if database.db != nil {
		database.vacuum()
		database.close()
	}
	gtkApp.Quit()
}

func (gui *GUI) startWatcher() {
	cmd := "clyp"
	if os.Getenv("RUN_ENV") == "dev" {
		cmd = "./clyp"
	}
	watcher := exec.Command(cmd, " --watch")
	watcher.Env = os.Environ()
	watcher.Env = append(watcher.Env, "GDK_BACKEND=x11")
	if err := watcher.Start(); err != nil {
		log.Printf("Failed to start watcher: %v", err)
	}
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
	gui.setupClipBoardListEvents(gtkApp)
	gui.setupWindowEvents()
	gui.setupSearchBarEvents()
}

func (gui *GUI) setupAppEvents(gtkApp *gtk.Application) {
	gui.window.ConnectCloseRequest(func() bool {
		gui.shutdown(gtkApp)
		return true
	})
}

func (gui *GUI) setupClipBoardListEvents(gtkApp *gtk.Application) {
	clipboardListkeyController := gtk.NewEventControllerKey()

	clipboardListkeyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_Return || keyval == gdk.KEY_KP_Enter {
			selectedRow := gui.clipboardItemsList.SelectedRow()
			if selectedRow != nil {
				gui.searchBarControl("hide")
				clipboard.copy(selectedRow.Name(), gtkApp)
				if config.CloseOnCopy {
					gui.shutdown(gtkApp)
					return true
				}
				gui.updateClipboardRows(true)
				gui.focusClipboardItemByIndex(0)
				return true
			}
		}

		if keyval == gdk.KEY_Delete {
			selectedRow := gui.clipboardItemsList.SelectedRow()
			selectedRowIndex := selectedRow.Index()
			if selectedRow != nil {
				clipboard.removeFromDatabase(selectedRow.Name())
				gui.updateClipboardRows(true)
				gui.focusClipboardItemByIndex(selectedRowIndex)
				return true
			}
		}

		if keyval == gdk.KEY_Escape {
			if gui.searchBarControl("is-active").(bool) {
				gui.searchBarControl("hide")
				return true
			} else {
				gui.shutdown(gtkApp)
			}
		}

		return false
	})

	gestureClick := gtk.NewGestureClick()

	gestureClick.ConnectPressed(func(nPress int, x, y float64) {
		if nPress == 2 {
			selectedRow := gui.clipboardItemsList.SelectedRow()
			if selectedRow != nil {
				gui.searchBarControl("hide")
				clipboard.copy(selectedRow.Name(), gtkApp)
				if config.CloseOnCopy {
					gui.shutdown(gtkApp)
					return
				}
				gui.updateClipboardRows(true)
				gui.focusClipboardItemByIndex(0)
			}
		}
	})

	gui.clipboardItemsList.AddController(clipboardListkeyController)
	gui.clipboardItemsList.AddController(gestureClick)

}

func (gui *GUI) setupWindowEvents() {
	windowKeyController := gtk.NewEventControllerKey()

	windowKeyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		// Show search bar when Ctrl+F is pressed
		if state&gdk.ControlMask != 0 && keyval == gdk.KEY_f {
			gui.searchBarControl("toggle")
			return true
		}
		// Type to search
		if gui.isPrintableKey(keyval, state) {
			gui.searchBarControl("show")
			gui.searchEntry.GrabFocus()
			currentText := gui.searchEntry.Text()
			newText := currentText + string(rune(keyval))
			gui.searchEntry.SetText(newText)
			gui.searchEntry.SetPosition(-1)
			return true
		}
		return false
	})

	gui.window.AddController(windowKeyController)
}

func (gui *GUI) isPrintableKey(keyval uint, state gdk.ModifierType) bool {
	// Ignore modifier keys
	if state&(gdk.ControlMask|gdk.AltMask|gdk.SuperMask) != 0 {
		return false
	}
	// Ignore special keys
	switch keyval {
	case gdk.KEY_Return, gdk.KEY_KP_Enter,
		gdk.KEY_Tab, gdk.KEY_KP_Tab,
		gdk.KEY_Escape,
		gdk.KEY_BackSpace,
		gdk.KEY_Delete,
		gdk.KEY_Insert,
		gdk.KEY_Home, gdk.KEY_End,
		gdk.KEY_Page_Up, gdk.KEY_Page_Down,
		gdk.KEY_Up, gdk.KEY_Down, gdk.KEY_Left, gdk.KEY_Right,
		gdk.KEY_KP_Up, gdk.KEY_KP_Down, gdk.KEY_KP_Left, gdk.KEY_KP_Right,
		gdk.KEY_F1, gdk.KEY_F2, gdk.KEY_F3, gdk.KEY_F4, gdk.KEY_F5, gdk.KEY_F6,
		gdk.KEY_F7, gdk.KEY_F8, gdk.KEY_F9, gdk.KEY_F10, gdk.KEY_F11, gdk.KEY_F12,
		gdk.KEY_Shift_L, gdk.KEY_Shift_R,
		gdk.KEY_Control_L, gdk.KEY_Control_R,
		gdk.KEY_Alt_L, gdk.KEY_Alt_R,
		gdk.KEY_Super_L, gdk.KEY_Super_R,
		gdk.KEY_Menu, gdk.KEY_Print, gdk.KEY_Scroll_Lock, gdk.KEY_Pause,
		gdk.KEY_Caps_Lock, gdk.KEY_Num_Lock:
		return false
	}
	return true
}

func (gui *GUI) searchBarControl(action string) interface{} {
	currentState := gui.searchBar.ObjectProperty("search-mode-enabled").(bool)
	switch action {
	case "show":
		if !currentState {
			gui.searchToggleButton.SetActive(true)
			gui.searchBar.SetObjectProperty("search-mode-enabled", true)
			gui.searchEntry.GrabFocus()
		}
	case "hide":
		if currentState {
			gui.searchToggleButton.SetActive(false)
			gui.searchBar.SetObjectProperty("search-mode-enabled", false)
			gui.focusClipboardItemByIndex(0)
		}
	case "toggle":
		if currentState {
			gui.searchBarControl("hide")
		} else {
			gui.searchBarControl("show")
		}
	case "is-active":
		return currentState
	}
	return nil
}

func (gui *GUI) setupSearchBarEvents() {
	gui.searchEntry.ConnectSearchChanged(func() {
		if gui.searchEntry.Text() == "" {
			database.searchFilter = ""
			gui.updateClipboardRows(false)
			gui.focusClipboardItemByIndex(0)
			gui.searchBarControl("hide")
			return
		}
		database.searchFilter = gui.searchEntry.Text()
		gui.updateClipboardRows(false)
	})
	gui.searchBar.ConnectEntry(gui.searchEntry)
	gui.searchToggleButton.ConnectToggled(func() {
		gui.searchBarControl("toggle")
	})
	gui.searchEntry.ConnectActivate(func() {
		gui.focusClipboardItemByIndex(0)
	})
	searchEntryKeyController := gtk.NewEventControllerKey()
	searchEntryKeyController.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_Escape {
			gui.searchBarControl("hide")
			return true
		}
		if keyval == gdk.KEY_Down || keyval == gdk.KEY_KP_Down || keyval == gdk.KEY_Tab || keyval == gdk.KEY_KP_Tab {
			gui.focusClipboardItemByIndex(0)
			return true
		}
		return false
	})

	gui.searchEntry.AddController(searchEntryKeyController)
}

func (gui *GUI) focusClipboardItemByIndex(index int) {
	if gui.clipboardItemsList.RowAtIndex(index) == nil {
		return
	}
	item := gui.clipboardItemsList.RowAtIndex(index)
	gui.clipboardItemsList.SelectRow(item)
	gtk.ListBoxRow(*item).Cast().(*gtk.ListBoxRow).GrabFocus()
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

func (gui *GUI) setupSettingsAction(gtkApp *gtk.Application) {
	settingsAction := gio.NewSimpleAction("settings", nil)
	settingsAction.ConnectActivate(func(parameter *glib.Variant) {
		gui.showSettingsDialog(gui.window)
	})
	gtkApp.AddAction(settingsAction)
}

func (gui *GUI) showSettingsDialog(parent *gtk.ApplicationWindow) {
	settingsDialog := gtk.NewDialog()
	settingsDialog.SetTransientFor(&parent.Window)
	settingsDialog.SetModal(true)
	settingsDialog.SetTitle("Settings")
	settingsDialog.SetDefaultSize(420, 120)
	settingsDialog.AddButton("Close", int(gtk.ResponseClose))

	contentArea := settingsDialog.ContentArea()
	contentArea.SetMarginTop(16)
	contentArea.SetMarginBottom(16)
	contentArea.SetMarginStart(16)
	contentArea.SetMarginEnd(16)
	contentArea.SetSpacing(8)

	runOnStartupCheckButton := gtk.NewCheckButtonWithLabel("Run on Startup")
	runOnStartupCheckButton.SetActive(gui.startupEntryControl("check"))
	runOnStartupCheckButton.ConnectToggled(func() {
		runOnStartup := runOnStartupCheckButton.Active()
		if runOnStartup {
			gui.startupEntryControl("add")
		} else {
			gui.startupEntryControl("remove")
		}
		if gui.runOnStartupAction != nil {
			gui.runOnStartupAction.SetState(glib.NewVariantBoolean(runOnStartup))
		}
	})

	closeOnCopyCheckButton := gtk.NewCheckButtonWithLabel("Close on Copy")
	closeOnCopyCheckButton.SetActive(config.CloseOnCopy)
	closeOnCopyCheckButton.ConnectToggled(func() {
		config.CloseOnCopy = closeOnCopyCheckButton.Active()
		config.save()
		if gui.closeOnCopyAction != nil {
			gui.closeOnCopyAction.SetState(glib.NewVariantBoolean(config.CloseOnCopy))
		}
	})

	focusWindowCheckButton := gtk.NewCheckButtonWithLabel("Focus the application window when opening it")
	focusWindowCheckButton.SetActive(config.FocusWindowOnOpen)
	focusWindowCheckButton.ConnectToggled(func() {
		config.FocusWindowOnOpen = focusWindowCheckButton.Active()
		config.save()
	})

	maxItemsBox := gtk.NewBox(gtk.OrientationHorizontal, 12)
	maxItemsLabel := gtk.NewLabel("Maximum saved clipboard items")
	maxItemsLabel.SetXAlign(0)
	maxItemsLabel.SetHExpand(true)

	maxItemsSpinButton := gtk.NewSpinButtonWithRange(1, 100000, 1)
	maxItemsSpinButton.SetValue(float64(config.MaxClipboardItems))
	maxItemsSpinButton.SetNumeric(true)
	maxItemsSpinButton.ConnectValueChanged(func() {
		config.MaxClipboardItems = maxItemsSpinButton.ValueAsInt()
		config.save()
		clipboard.enforceMaxItems()
		gui.updateClipboardRows(true)
	})

	clearClipboardButton := gtk.NewButtonWithLabel("Clear Clipboard")
	clearClipboardButton.AddCSSClass("destructive-action")
	clearClipboardButton.ConnectClicked(func() {
		gui.showClearClipboardDialog(settingsDialog)
	})

	maxItemsBox.Append(maxItemsLabel)
	maxItemsBox.Append(maxItemsSpinButton)

	contentArea.Append(runOnStartupCheckButton)
	contentArea.Append(closeOnCopyCheckButton)
	contentArea.Append(focusWindowCheckButton)
	contentArea.Append(maxItemsBox)
	contentArea.Append(clearClipboardButton)
	settingsDialog.ConnectResponse(func(responseId int) {
		settingsDialog.Close()
	})
	settingsDialog.SetVisible(true)
}

func (gui *GUI) showClearClipboardDialog(parent *gtk.Dialog) {
	confirmDialog := gtk.NewDialog()
	confirmDialog.SetTransientFor(&parent.Window)
	confirmDialog.SetModal(true)
	confirmDialog.SetTitle("Clear Clipboard")
	confirmDialog.AddButton("Cancel", int(gtk.ResponseCancel))
	confirmDialog.AddButton("Clear", int(gtk.ResponseAccept))
	confirmDialog.SetDefaultResponse(int(gtk.ResponseCancel))

	contentArea := confirmDialog.ContentArea()
	contentArea.SetMarginTop(16)
	contentArea.SetMarginBottom(16)
	contentArea.SetMarginStart(16)
	contentArea.SetMarginEnd(16)

	messageLabel := gtk.NewLabel("Clear all saved clipboard items?")
	messageLabel.SetWrap(true)
	messageLabel.SetXAlign(0)
	contentArea.Append(messageLabel)

	confirmDialog.ConnectResponse(func(responseId int) {
		if responseId == int(gtk.ResponseAccept) {
			clipboard.removeAllFromDatabase()
			gui.updateClipboardRows(true)
		}
		confirmDialog.Close()
	})
	confirmDialog.SetVisible(true)
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
	aboutDialog.SetLogoIconName(app.id)
	aboutDialog.SetModal(true)
	aboutDialog.SetVersion(app.version)
	aboutDialog.SetProgramName(app.name)
	aboutDialog.SetComments("Clipboard manager.")
	aboutDialog.SetWebsite(app.helpURL)
	aboutDialog.SetWebsiteLabel(app.helpURL)
	aboutDialog.SetVisible(true)
}

func (gui *GUI) setupActionRunOnStartup(gtkApp *gtk.Application) {
	hasStartupEntry := gui.startupEntryControl("check")
	initialState := glib.NewVariantBoolean(hasStartupEntry)
	actionRunOnStartup := gio.NewSimpleActionStateful("run_on_startup", nil, initialState)
	actionRunOnStartup.ConnectActivate(func(parameter *glib.Variant) {
		gui.handleRunOnStartup(actionRunOnStartup)
	})
	gui.runOnStartupAction = actionRunOnStartup
	gtkApp.AddAction(actionRunOnStartup)
	if !hasStartupEntry {
		glib.TimeoutAdd(1000, func() bool {
			gui.showAddToStartupToast()
			return false
		})
	}
}

func (gui *GUI) setupCloseOnCopy(gtkApp *gtk.Application) {
	initialState := glib.NewVariantBoolean(config.CloseOnCopy)
	actionCloseOnCopy := gio.NewSimpleActionStateful("close_on_copy", nil, initialState)
	actionCloseOnCopy.ConnectActivate(func(parameter *glib.Variant) {
		gui.handleCloseOnCopy(actionCloseOnCopy)
	})
	gui.closeOnCopyAction = actionCloseOnCopy
	gtkApp.AddAction(actionCloseOnCopy)
}

func (gui *GUI) handleCloseOnCopy(action *gio.SimpleAction) {
	currentState := action.State().Boolean()
	newState := glib.NewVariantBoolean(!currentState)
	action.SetState(newState)
	config.CloseOnCopy = newState.Boolean()
	config.save()
}

func (gui *GUI) handleRunOnStartup(action *gio.SimpleAction) {
	currentState := action.State().Boolean()
	newState := glib.NewVariantBoolean(!currentState)
	action.SetState(newState)
	if newState.Boolean() {
		gui.startupEntryControl("add")
	} else {
		gui.startupEntryControl("remove")
	}
}

func (gui *GUI) startupEntryControl(action string) bool {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get user home directory: %v", err)
		return false
	}
	autoStartdesktopFile := userHomeDir + "/.config/autostart/clyp-watcher.desktop"
	switch action {
	case "add":
		err = os.WriteFile(autoStartdesktopFile, []byte(watcherFile), 0644)
	case "remove":
		err = os.Remove(autoStartdesktopFile)
	case "check":
		_, err = os.Stat(autoStartdesktopFile)
	}
	return err == nil
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

	label := gtk.NewLabel("Go to Settings to add Clyp to the system startup.")
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
