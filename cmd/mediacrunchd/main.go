package main

import (
	"fmt"
	"os"

	"mediacrunch/internal/app"
	"mediacrunch/pkg/codec/image"
)

func main() {
	application := app.New(image.NewTranscoder())
	if err := application.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
