package main

import (
	"fmt"
	"net"
	"os"
)

type IPC struct{}

func (ipc *IPC) notify() {
	conn, err := net.Dial("unix", "/tmp/clyp.sock")
	if err != nil {
		return
	}
	defer conn.Close()
	conn.Write([]byte("1"))
}

func (ipc *IPC) listen() {
	os.Remove("/tmp/clyp.sock")
	listener, err := net.Listen("unix", "/tmp/clyp.sock")
	if err != nil {
		fmt.Printf("%v", err)
		return
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		b := make([]byte, 1)
		conn.Read(b)
		if b[0] == '1' {
			gui.updateClipboardRows(true)
		}
		conn.Close()
	}
}
