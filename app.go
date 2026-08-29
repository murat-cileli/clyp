package main

import (
	_ "embed"
	"os"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	_ "github.com/mattn/go-sqlite3"
)

var (
	clipboard Clipboard
	config    Config
)

type Application struct {
	name      string
	id        string
	version   string
	dataDir   string
	configDir string
	helpURL   string
}

func (app *Application) init() {
	app.id = "bio.murat.clyp"
	app.name = "Clyp"
	app.version = "0.9.6"
	app.helpURL = "https://github.com/murat-cileli/clyp"
	app.setupDataDir()
	app.setupConfigDir()
	config.init()
	database.init()
	clipboard.enforceMaxItems()
}

func (app *Application) setupConfigDir() {
	app.configDir = glib.GetUserConfigDir() + "/" + app.id
	if _, err := os.Stat(app.configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(app.configDir, 0755); err != nil {
			panic(err.Error())
		}
	}
}

func (app *Application) setupDataDir() {
	app.dataDir = glib.GetUserDataDir() + "/" + app.id
	if _, err := os.Stat(app.dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(app.dataDir, 0755); err != nil {
			panic(err.Error())
		}
	}
}

func (app *Application) getConfig() {
	config.init()
}
