package main

import (
	"log"

	"github.com/ToxicSozo/avito-test/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

}
