package main

import (
	"os"
)

var (
	app      Application
	gui      GUI
	service  Service
	database Database
)

func main() {
	app.id = "bio.murat.clyp"
	app.setupDataDir()
	database.init()

	switch len(os.Args) {
	case 1:
		gui.init()
	case 2:
		switch os.Args[1] {
		case "watch":
			service.init()
		}
	}
}
