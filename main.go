package main

import (
	"iec104/config"
	"iec104/ui"
)

func main() {
	// Initialize configuration
	cfg := config.LoadFromDisk()

	// Initialize and start the UI
	app := ui.NewApp(cfg)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
