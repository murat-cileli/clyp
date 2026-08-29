package main

import (
	"log"
	"net"
	"os"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

type IPC struct{}

func (ipc *IPC) notify() {
	conn, err := net.Dial("unix", "/tmp/clyp.sock")
	if err != nil {
		log.Printf("Failed to connect to socket: %v", err)
		return
	}
	defer conn.Close()
	_, err = conn.Write([]byte("1"))
	if err != nil {
		log.Printf("Failed to write to socket: %v", err)
		return
	}
}

func (ipc *IPC) listen() {
	os.Remove("/tmp/clyp.sock")
	listener, err := net.Listen("unix", "/tmp/clyp.sock")
	if err != nil {
		log.Printf("Failed to listen on socket: %v", err)
		return
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		b := make([]byte, 1)
		_, err = conn.Read(b)
		if err != nil {
			log.Printf("Failed to read from socket: %v", err)
			continue
		}
		glib.IdleAdd(func() {
			gui.updateClipboardRows(true)
			gui.focusClipboardItemByIndex(0)
		})
		conn.Close()
	}
}
