package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port			string
	Host			string
	DBPath			string
	GoogleClientID		string
	GoogleClientSecret	string
	VideosPending		string
	VideosUploaded		string
	MaxConcurrency		int
	LogLevel		string
}

func (c *Config) CallbackURL() string {
	return "http://localhost:" + c.Port + "/callback"
}

func Load() *Config {
	loadEnvFile(".env")
	conc, _ := strconv.Atoi(envOr("MAX_CONCURRENCY", "3"))
	return &Config{
		Port:			envOr("PORT", "3000"),
		Host:			envOr("HOST", "0.0.0.0"),
		DBPath:			envOr("DB_PATH", "./data.db"),
		GoogleClientID:		envOr("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:	envOr("GOOGLE_CLIENT_SECRET", ""),
		VideosPending:		envOr("VIDEOS_PENDING", "./videos/pending"),
		VideosUploaded:		envOr("VIDEOS_UPLOADED", "./videos/uploaded"),
		MaxConcurrency:		conc,
		LogLevel:		envOr("LOG_LEVEL", "info"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
