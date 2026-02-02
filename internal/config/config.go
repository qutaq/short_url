package config

import "flag"

var (
	serverAddr = flag.String("a", ":8080", "HTTP server address")
	baseURL    = flag.String("b", "http://localhost:8080", "Base URL for shortened links")
)

type Config struct {
	ServerAddr string
	BaseURL    string
}

func Load() *Config {
	flag.Parse()
	return &Config{
		ServerAddr: *serverAddr,
		BaseURL:    *baseURL,
	}
}
