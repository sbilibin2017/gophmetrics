package main

import (
	"context"
	"log"
)

// main is the entry point of the application. It parses command-line flags,
// runs the application with the parsed configuration, and logs any fatal errors.
func main() {
	config := parseFlags()

	err := run(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}
}
