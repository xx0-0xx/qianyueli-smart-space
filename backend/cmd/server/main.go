package main

import (
	"flag"
	"log"

	"smart-space-management/backend/config"
	"smart-space-management/backend/internal/app"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	a, err := app.New(cfg)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	if err := a.Run(); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
