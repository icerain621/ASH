package config

import (
	"os"
)

type Config struct {
	HTTPAddr     string
	DataDir      string
	ScenariosDir string
	WebDir       string
}

func Load() Config {
	cfg := Config{
		HTTPAddr:     envOr("ASH_HTTP_ADDR", ":8080"),
		DataDir:      envOr("ASH_DATA_DIR", ".ash"),
		ScenariosDir: envOr("ASH_SCENARIOS_DIR", resolveScenariosDir()),
		WebDir:       envOr("ASH_WEB_DIR", resolveWebDir()),
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func resolveScenariosDir() string {
	for _, p := range []string{"scenarios", "../scenarios", "backend/scenarios"} {
		if isDir(p) {
			return p
		}
	}
	return "scenarios"
}

func resolveWebDir() string {
	for _, p := range []string{
		"frontend/dist",
		"../frontend/dist",
		"frontend/public",
		"../frontend/public",
	} {
		if isDir(p) {
			return p
		}
	}
	return "frontend/dist"
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
