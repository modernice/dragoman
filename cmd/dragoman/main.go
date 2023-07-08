package main

import (
	_ "embed"
	"log"

	"github.com/modernice/dragoman"
	"github.com/modernice/dragoman/internal/cli"
)

func main() {
	log.SetFlags(0)
	cli.New(dragoman.Version()).Run()
}
