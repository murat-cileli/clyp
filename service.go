package main

import (
	"os"

	_ "github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type Service struct{}

func (service *Service) init() {
	gtkServiceApp := gtk.NewApplication("bio.murat.clyp", gio.ApplicationDefaultFlags)
	gtkServiceApp.ConnectActivate(func() { service.activate(gtkServiceApp) })

	if code := gtkServiceApp.Run(nil); code > 0 {
		os.Exit(code)
	}
}

func (service *Service) activate(gtkServiceApp *gtk.Application) {
	database.vacuum()
	clipboard.updateRecentContentFromDatabase()
	clipboard.watch()
	gtkServiceApp.Hold()
}
