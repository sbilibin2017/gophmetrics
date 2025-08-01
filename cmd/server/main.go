package main

import (
	"context"
	"log"

	"github.com/sbilibin2017/gophmetrics/internal/apps"
)

func main() {
	config := parseFlags()

	app := apps.NewServer(config)

	err := app.Run(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}
