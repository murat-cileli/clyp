package main

import (
	"log"
	"os"

	_ "github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var (
	clipboard Clipboard
)

type Service struct{}

func (service *Service) init() {
	gtkServiceApp := gtk.NewApplication("bio.murat.clyp-service", gio.ApplicationDefaultFlags)
	gtkServiceApp.ConnectActivate(func() { service.activate(gtkServiceApp) })

	if code := gtkServiceApp.Run(nil); code > 0 {
		os.Exit(code)
	}

	log.Println("Service init()")
}

func (service *Service) activate(gtkServiceApp *gtk.Application) {
	log.Println("Service activate()")
	database.vacuum()
	clipboard.updateRecentContentFromDatabase()
	clipboard.watch()
	gtkServiceApp.Hold()
}
