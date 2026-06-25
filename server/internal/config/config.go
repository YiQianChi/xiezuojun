package config

import (
	"os"
	"strconv"
)

// Config 应用配置
type Config struct {
	DB       DBConfig
	JWT      JWTConfig
	AI       AIConfig
	Server   ServerConfig
	DataRoot string
}

type DBConfig struct {
	DSN string
}

type JWTConfig struct {
	Secret string
}

type AIConfig struct {
	APIKey  string
	BaseURL string
}

type ServerConfig struct {
	Port string
}

func Load() *Config {
	return &Config{
		DB: DBConfig{
			DSN: getEnv("DB_DSN", "root:password@tcp(127.0.0.1:3306)/xiezuojun?charset=utf8mb4&parseTime=True&loc=Local"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "change-me-in-production"),
		},
		AI: AIConfig{
			APIKey:  getEnv("AI_API_KEY", ""),
			BaseURL: getEnv("AI_BASE_URL", "https://open.bigmodel.cn/api/paas/v4"),
		},
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
		},
		DataRoot: getEnv("DATA_ROOT", "./data"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
