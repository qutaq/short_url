package config

import (
	"flag"
	"os"
)

const (
	defaultServerAddr      = ":8080"
	defaultBaseURL         = "http://localhost:8080"
	defaultFileStoragePath = "urls.json"
	defaultDatabaseDSN     = ""
)

var (
	serverAddr      = flag.String("a", defaultServerAddr, "HTTP server address")
	baseURL         = flag.String("b", defaultBaseURL, "Base URL for shortened links")
	fileStoragePath = flag.String("f", defaultFileStoragePath, "Path to file storage for URLs")
	databaseDSN     = flag.String("d", defaultDatabaseDSN, "PostgreSQL connection string")
)

type Config struct {
	ServerAddr      string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
}

func Load() *Config {
	flag.Parse()

	flagSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		flagSet[f.Name] = true
	})

	return &Config{
		ServerAddr:      resolve("SERVER_ADDRESS", *serverAddr, flagSet["a"], defaultServerAddr),
		BaseURL:         resolve("BASE_URL", *baseURL, flagSet["b"], defaultBaseURL),
		FileStoragePath: resolve("FILE_STORAGE_PATH", *fileStoragePath, flagSet["f"], defaultFileStoragePath),
		DatabaseDSN:     resolve("DATABASE_DSN", *databaseDSN, flagSet["d"], defaultDatabaseDSN),
	}
}

func resolve(envKey, flagVal string, flagExplicit bool, defaultVal string) string {
	if env, ok := os.LookupEnv(envKey); ok {
		return env
	}
	if flagExplicit {
		return flagVal
	}
	return defaultVal
}
