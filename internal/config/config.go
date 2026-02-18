package config

import (
	"flag"
	"os"
)

const (
	defaultServerAddr = ":8080"
	defaultBaseURL    = "http://localhost:8080"
)

var (
	serverAddr = flag.String("a", defaultServerAddr, "HTTP server address")
	baseURL    = flag.String("b", defaultBaseURL, "Base URL for shortened links")
)

type Config struct {
	ServerAddr string
	BaseURL    string
}

func Load() *Config {
	flag.Parse()

	flagSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		flagSet[f.Name] = true
	})

	return &Config{
		ServerAddr: resolve("SERVER_ADDRESS", *serverAddr, flagSet["a"], defaultServerAddr),
		BaseURL:    resolve("BASE_URL", *baseURL, flagSet["b"], defaultBaseURL),
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
