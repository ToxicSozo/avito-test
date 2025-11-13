package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/ToxicSozo/avito-test/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Fatalf("marshal config: %v", err)
	}

	fmt.Println(string(data))
}
