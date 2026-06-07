package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
)

const (
	defaultServerAddr      = ":8080"
	defaultBaseURL         = "http://localhost:8080"
	defaultFileStoragePath = "urls.json"
	defaultDatabaseDSN     = ""
	defaultAuditFile       = ""
	defaultAuditURL        = ""
)

var (
	serverAddr      *string
	baseURL         *string
	fileStoragePath *string
	databaseDSN     *string
	auditFile       *string
	auditURL        *string
	enableHTTPS     *bool
	configPath      string
)

func init() {
	registerFlags(flag.CommandLine)
}

func registerFlags(flagSet *flag.FlagSet) {
	configPath = ""
	serverAddr = flagSet.String("a", defaultServerAddr, "HTTP server address")
	baseURL = flagSet.String("b", defaultBaseURL, "Base URL for shortened links")
	fileStoragePath = flagSet.String("f", defaultFileStoragePath, "Path to file storage for URLs")
	databaseDSN = flagSet.String("d", defaultDatabaseDSN, "PostgreSQL connection string")
	auditFile = flagSet.String("audit-file", defaultAuditFile, "Path to audit log file")
	auditURL = flagSet.String("audit-url", defaultAuditURL, "Remote audit receiver URL")
	enableHTTPS = flagSet.Bool("s", false, "Enable HTTPS server")
	flagSet.StringVar(&configPath, "c", "", "Path to JSON config file")
	flagSet.StringVar(&configPath, "config", "", "Path to JSON config file")
}

// Config содержит настройки запуска сервера сокращения ссылок.
type Config struct {
	// ServerAddr содержит адрес, на котором HTTP-сервер принимает запросы.
	ServerAddr string
	// BaseURL содержит публичный префикс для формирования коротких ссылок.
	BaseURL string
	// FileStoragePath содержит путь к файловому JSONL-хранилищу.
	FileStoragePath string
	// DatabaseDSN содержит строку подключения к PostgreSQL.
	DatabaseDSN string
	// AuditFile содержит необязательный путь для локальных событий аудита.
	AuditFile string
	// AuditURL содержит необязательную удалённую точку приёма событий аудита.
	AuditURL string
	// EnableHTTPS включает запуск веб-сервера по HTTPS.
	EnableHTTPS bool
}

type fileConfig struct {
	ServerAddr      *string `json:"server_address"`
	BaseURL         *string `json:"base_url"`
	FileStoragePath *string `json:"file_storage_path"`
	DatabaseDSN     *string `json:"database_dsn"`
	AuditFile       *string `json:"audit_file"`
	AuditURL        *string `json:"audit_url"`
	EnableHTTPS     *bool   `json:"enable_https"`
}

// Load читает конфигурацию из переменных окружения, флагов командной строки и JSON-файла.
// Приоритет значений: переменные окружения, флаги, файл конфигурации, значения по умолчанию.
func Load() *Config {
	flag.Parse()

	flagSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		flagSet[f.Name] = true
	})

	fileCfg := fileConfig{}
	if path := resolveConfigPath(flagSet); path != "" {
		var err error
		fileCfg, err = loadFileConfig(path)
		if err != nil {
			panic(err)
		}
	}

	return &Config{
		ServerAddr:      resolve("SERVER_ADDRESS", *serverAddr, flagSet["a"], fileCfg.ServerAddr, defaultServerAddr),
		BaseURL:         resolve("BASE_URL", *baseURL, flagSet["b"], fileCfg.BaseURL, defaultBaseURL),
		FileStoragePath: resolve("FILE_STORAGE_PATH", *fileStoragePath, flagSet["f"], fileCfg.FileStoragePath, defaultFileStoragePath),
		DatabaseDSN:     resolve("DATABASE_DSN", *databaseDSN, flagSet["d"], fileCfg.DatabaseDSN, defaultDatabaseDSN),
		AuditFile:       resolve("AUDIT_FILE", *auditFile, flagSet["audit-file"], fileCfg.AuditFile, defaultAuditFile),
		AuditURL:        resolve("AUDIT_URL", *auditURL, flagSet["audit-url"], fileCfg.AuditURL, defaultAuditURL),
		EnableHTTPS:     resolveBool("ENABLE_HTTPS", *enableHTTPS, flagSet["s"], fileCfg.EnableHTTPS, false),
	}
}

func resolveConfigPath(flagSet map[string]bool) string {
	if env, ok := os.LookupEnv("CONFIG"); ok {
		return env
	}
	if flagSet["c"] || flagSet["config"] {
		return configPath
	}
	return ""
}

func loadFileConfig(path string) (fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}, fmt.Errorf("read config file %q: %w", path, err)
	}

	cfg := fileConfig{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fileConfig{}, fmt.Errorf("parse config file %q: %w", path, err)
	}

	return cfg, nil
}

func resolve(envKey, flagVal string, flagExplicit bool, configVal *string, defaultVal string) string {
	if env, ok := os.LookupEnv(envKey); ok {
		return env
	}
	if flagExplicit {
		return flagVal
	}
	if configVal != nil {
		return *configVal
	}
	return defaultVal
}

func resolveBool(envKey string, flagVal, flagExplicit bool, configVal *bool, defaultVal bool) bool {
	if env, ok := os.LookupEnv(envKey); ok {
		if env == "" {
			return true
		}
		parsed, err := strconv.ParseBool(env)
		if err != nil {
			return defaultVal
		}
		return parsed
	}
	if flagExplicit {
		return flagVal
	}
	if configVal != nil {
		return *configVal
	}
	return defaultVal
}
