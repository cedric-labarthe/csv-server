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

	// TODO: Check the need for a configurable host address, e.g. for running in a container.
	// This one should be fine to run the server locally
	localAddr := "127.0.0.1"

	return Config{
		Addr:    localAddr + ":" + port,
		DataDir: dataDir,
	}, nil
}
