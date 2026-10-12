package main

import (
	"context"
	"os"

	"github.com/coalaura/plain/minimal"
)

//go:generate go run ./internal/generate

var Version = "dev"

var log = minimal.New()

func main() {
	err := NewCommand().Run(context.Background(), os.Args)
	if err != nil {
		log.Warnln(err)

		os.Exit(1)
	}
}
