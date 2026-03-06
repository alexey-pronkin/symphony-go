package main

import (
	"log"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
