package config

import "os"

type Config struct {
	OllamaBaseURL string
	Model         string
}

func Load() Config {
	return Config{
		OllamaBaseURL: getEnv(
			"OLLAMA_BASE_URL",
			"http://localhost:11434",
		),
		Model: getEnv(
			"OLLAMA_MODEL",
			"qwen3:8b",
		),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
