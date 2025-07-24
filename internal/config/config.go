package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Env			string
	DevMode		bool
	PostgresURL string
	ValkeyURL 	string
	SkinsCDNURL string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	config := &Config{
		Env: getEnv("ENV", "dev"),
		DevMode: getEnvAsBool("DEV_MODE", true),
		SkinsCDNURL: getEnv("SKINS_CDN_URL", ""),
	}

	if config.DevMode {
		config.PostgresURL = getEnv("DEV_POSTGRES_URL", "")
		config.ValkeyURL = getEnv("DEV_VALKEY_URL", "")
	} else {
		config.PostgresURL = getEnv("DEV_POSTGRES_URL", "")
		config.ValkeyURL = getEnv("DEV_VALKEY_URL", "")
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) validate() error {
	if c.PostgresURL == "" {
		return fmt.Errorf("database URL not set")
	}
	if c.ValkeyURL == "" {
		return fmt.Errorf("valkey URL not set")
	}
	if c.SkinsCDNURL == "" {
		return fmt.Errorf("skins CDN URL not set")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
}
