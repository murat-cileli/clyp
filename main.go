package main

var (
	app      Application
	gui      GUI
	database Database
	ipc      IPC
)

func main() {
	app.init()
	gui.init()
}
