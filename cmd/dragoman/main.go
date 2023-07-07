package main

import (
	"log"

	"github.com/modernice/dragoman/internal/cli"
)

func main() {
	log.SetFlags(0)
	cli.New().Run()
}
