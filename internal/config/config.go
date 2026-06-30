package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr    string
	DataDir string
}

func Load() (Config, error) {
	dataDir, ok := os.LookupEnv("DATA_DIR")
	if !ok {
		dataDir = "./data"
	}

	if _, err := os.Stat(dataDir); err != nil {
		return Config{}, fmt.Errorf("data directory %q not accessible: %w", dataDir, err)
	}

	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	host, ok := os.LookupEnv("HOST")
	if !ok {
		host = "0.0.0.0"
	}

	return Config{
		Addr:    host + ":" + port,
		DataDir: dataDir,
	}, nil
}
