package main

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
)

type IPC struct {
	dbus          *dbus.Conn
	interfaceName string
	objectPath    string
	sender        string
}

func (ipc *IPC) init() {
	ipc.interfaceName = "bio.murat.clyp"
	ipc.objectPath = "/bio/murat/clyp"
	ipc.sender = "bio.murat.clyp-service"

	var err error
	ipc.dbus, err = dbus.ConnectSessionBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to connect to session bus:", err)
		os.Exit(1)
	}
	defer ipc.dbus.Close()

	if err = ipc.dbus.AddMatchSignal(
		dbus.WithMatchObjectPath(dbus.ObjectPath(ipc.objectPath)),
		dbus.WithMatchInterface(ipc.interfaceName),
		dbus.WithMatchSender(ipc.sender),
	); err != nil {
		panic(err)
	}

	c := make(chan *dbus.Signal, 10)
	ipc.dbus.Signal(c)
	for v := range c {
		fmt.Println(v)
	}
}
