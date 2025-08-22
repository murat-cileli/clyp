package main

import (
	_ "embed"
	"os"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	_ "github.com/mattn/go-sqlite3"
)

var clipboard Clipboard

type Application struct {
	name    string
	id      string
	dataDir string
}

func (app *Application) setupDataDir() {
	app.dataDir = glib.GetUserDataDir() + "/" + app.id

	if _, err := os.Stat(app.dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(app.dataDir, 0755); err != nil {
			panic(err.Error())
		}
	}
}
