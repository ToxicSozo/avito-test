package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/ToxicSozo/avito-test/internal/config"
)

func main() {
	cfg := config.Load()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Fatalf("marshal config: %v", err)
	}

	fmt.Println(string(data))
}
