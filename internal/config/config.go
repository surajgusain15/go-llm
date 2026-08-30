package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	OllamaBaseURL string
	Model         string
	MySQLDSN      string

	MySQLQueryTimeout     time.Duration
	MySQLMaxRows          int
	MySQLMaxResultBytes   int
	MySQLSchemaCacheTTL   time.Duration
	MySQLMaxJoins         int
	MySQLMaxUnionBranches int
	MySQLMaxSubqueryDepth int

	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	DBConnMaxIdleTime time.Duration
}

func Load() Config {

	// Load .env if present.
	// Existing environment variables take precedence.
	_ = godotenv.Load()

	return Config{
		OllamaBaseURL: getEnv(
			"OLLAMA_BASE_URL",
			"http://localhost:11434",
		),

		Model: getEnv(
			"OLLAMA_MODEL",
			"qwen3:8b",
		),

		MySQLDSN: getEnv(
			"MYSQL_DSN",
			"root:password@tcp(localhost:3306)/chat",
		),

		MySQLQueryTimeout: getDurationEnv(
			"MYSQL_QUERY_TIMEOUT",
			10*time.Second,
		),

		MySQLMaxRows: getIntEnv(
			"MYSQL_MAX_ROWS",
			100,
		),

		MySQLMaxResultBytes: getIntEnv(
			"MYSQL_MAX_RESULT_BYTES",
			1048576,
		),

		MySQLSchemaCacheTTL: getDurationEnv(
			"MYSQL_SCHEMA_CACHE_TTL",
			5*time.Minute,
		),

		MySQLMaxJoins: getIntEnv(
			"MYSQL_MAX_JOINS",
			3,
		),

		MySQLMaxUnionBranches: getIntEnv(
			"MYSQL_MAX_UNION_BRANCHES",
			3,
		),

		MySQLMaxSubqueryDepth: getIntEnv(
			"MYSQL_MAX_SUBQUERY_DEPTH",
			3,
		),

		DBMaxOpenConns: getIntEnv(
			"MYSQL_MAX_OPEN_CONNS",
			20,
		),

		DBMaxIdleConns: getIntEnv(
			"MYSQL_MAX_IDLE_CONNS",
			10,
		),

		DBConnMaxLifetime: getDurationEnv(
			"MYSQL_CONN_MAX_LIFETIME",
			30*time.Minute,
		),

		DBConnMaxIdleTime: getDurationEnv(
			"MYSQL_CONN_MAX_IDLE_TIME",
			10*time.Minute,
		),
	}
}

func getEnv(
	key string,
	fallback string,
) string {

	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}

func getDurationEnv(
	key string,
	fallback time.Duration,
) time.Duration {

	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return duration
}

func getIntEnv(
	key string,
	fallback int,
) int {

	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	result, err := strconv.Atoi(value)
	if err != nil || result <= 0 {
		return fallback
	}

	return result
}
