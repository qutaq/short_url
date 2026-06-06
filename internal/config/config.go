package config

import (
	"flag"
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
	serverAddr      = flag.String("a", defaultServerAddr, "HTTP server address")
	baseURL         = flag.String("b", defaultBaseURL, "Base URL for shortened links")
	fileStoragePath = flag.String("f", defaultFileStoragePath, "Path to file storage for URLs")
	databaseDSN     = flag.String("d", defaultDatabaseDSN, "PostgreSQL connection string")
	auditFile       = flag.String("audit-file", defaultAuditFile, "Path to audit log file")
	auditURL        = flag.String("audit-url", defaultAuditURL, "Remote audit receiver URL")
	enableHTTPS     = flag.Bool("s", false, "Enable HTTPS server")
)

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

// Load читает конфигурацию из переменных окружения и флагов командной строки.
// Переменные окружения имеют приоритет над явно переданными флагами.
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
		AuditFile:       resolve("AUDIT_FILE", *auditFile, flagSet["audit-file"], defaultAuditFile),
		AuditURL:        resolve("AUDIT_URL", *auditURL, flagSet["audit-url"], defaultAuditURL),
		EnableHTTPS:     resolveBool("ENABLE_HTTPS", *enableHTTPS, flagSet["s"], false),
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

func resolveBool(envKey string, flagVal, flagExplicit, defaultVal bool) bool {
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
	return defaultVal
}
