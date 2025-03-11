package main

import (
	"hashcash/internal/app"
	"log"
)

func main() {
	app := app.New()
	if err := app.Run(); err != nil {
		log.Fatalf("Could't start an application: %v", err)
	}
}
