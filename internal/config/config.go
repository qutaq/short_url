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

type flagValues struct {
	serverAddr      string
	baseURL         string
	fileStoragePath string
	databaseDSN     string
	auditFile       string
	auditURL        string
	enableHTTPS     bool
	configPath      string
}

func registerFlags(flagSet *flag.FlagSet) *flagValues {
	values := &flagValues{}
	flagSet.StringVar(&values.serverAddr, "a", defaultServerAddr, "HTTP server address")
	flagSet.StringVar(&values.baseURL, "b", defaultBaseURL, "Base URL for shortened links")
	flagSet.StringVar(&values.fileStoragePath, "f", defaultFileStoragePath, "Path to file storage for URLs")
	flagSet.StringVar(&values.databaseDSN, "d", defaultDatabaseDSN, "PostgreSQL connection string")
	flagSet.StringVar(&values.auditFile, "audit-file", defaultAuditFile, "Path to audit log file")
	flagSet.StringVar(&values.auditURL, "audit-url", defaultAuditURL, "Remote audit receiver URL")
	flagSet.BoolVar(&values.enableHTTPS, "s", false, "Enable HTTPS server")
	flagSet.StringVar(&values.configPath, "c", "", "Path to JSON config file")
	flagSet.StringVar(&values.configPath, "config", "", "Path to JSON config file")
	return values
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

// Load читает конфигурацию из переменных окружения, переданного набора флагов и JSON-файла.
// Приоритет значений: переменные окружения, флаги, файл конфигурации, значения по умолчанию.
func Load(flagSet *flag.FlagSet) (*Config, error) {
	values := registerFlags(flagSet)
	if !flagSet.Parsed() {
		if err := flagSet.Parse(os.Args[1:]); err != nil {
			return nil, fmt.Errorf("parse flags: %w", err)
		}
	}

	explicitFlags := make(map[string]bool)
	flagSet.Visit(func(f *flag.Flag) {
		explicitFlags[f.Name] = true
	})

	fileCfg := fileConfig{}
	if path := resolveConfigPath(explicitFlags, values.configPath); path != "" {
		var err error
		fileCfg, err = loadFileConfig(path)
		if err != nil {
			return nil, fmt.Errorf("load config file: %w", err)
		}
	}

	return &Config{
		ServerAddr:      resolve("SERVER_ADDRESS", values.serverAddr, explicitFlags["a"], fileCfg.ServerAddr, defaultServerAddr),
		BaseURL:         resolve("BASE_URL", values.baseURL, explicitFlags["b"], fileCfg.BaseURL, defaultBaseURL),
		FileStoragePath: resolve("FILE_STORAGE_PATH", values.fileStoragePath, explicitFlags["f"], fileCfg.FileStoragePath, defaultFileStoragePath),
		DatabaseDSN:     resolve("DATABASE_DSN", values.databaseDSN, explicitFlags["d"], fileCfg.DatabaseDSN, defaultDatabaseDSN),
		AuditFile:       resolve("AUDIT_FILE", values.auditFile, explicitFlags["audit-file"], fileCfg.AuditFile, defaultAuditFile),
		AuditURL:        resolve("AUDIT_URL", values.auditURL, explicitFlags["audit-url"], fileCfg.AuditURL, defaultAuditURL),
		EnableHTTPS:     resolveBool("ENABLE_HTTPS", values.enableHTTPS, explicitFlags["s"], fileCfg.EnableHTTPS, false),
	}, nil
}

func resolveConfigPath(flagSet map[string]bool, configPath string) string {
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
			return defaultVal
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
