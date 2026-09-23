//go:build bindings

package main

import (
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

// The Wails binding generation step compiles this file on its own with -tags bindings; it does not start Gin or the database, so no local Postgres is needed.
func main() {
	app := NewApp()
	_ = wails.Run(&options.App{
		Title: "EnterpriseRag Lite",
		Bind:  []interface{}{app},
	})
}
