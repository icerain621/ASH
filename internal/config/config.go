package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr       string
	DataDir        string
	ScenariosDir   string
	WebDir         string
	AgentExecutor  string
	CodexBin       string
	ExecGoURL      string
	RuntimeURL     string
	PluginGRPCAddr string
	AuthMode       string
	JWTSecret      string
	SecretKey      string
	DatabaseURL    string
	ArtifactStore  string
}

func Load() Config {
	cfg := Config{
		HTTPAddr:       envOr("ASH_HTTP_ADDR", ":8080"),
		DataDir:        envOr("ASH_DATA_DIR", ".ash"),
		ScenariosDir:   envOr("ASH_SCENARIOS_DIR", resolveScenariosDir()),
		WebDir:         envOr("ASH_WEB_DIR", resolveWebDir()),
		AgentExecutor:  envOr("ASH_AGENT_EXECUTOR", "execgo_codex"),
		CodexBin:       envOr("ASH_CODEX_BIN", "codex"),
		ExecGoURL:      envOr("EXECGO_URL", "http://127.0.0.1:8080"),
		RuntimeURL:     envOr("EXECGO_RUNTIME_URL", "http://127.0.0.1:18080"),
		PluginGRPCAddr: envOr("ASH_PLUGIN_GRPC_ADDR", defaultPluginGRPCAddr()),
		AuthMode:       envOr("ASH_AUTH_MODE", "dev"),
		JWTSecret:      envOr("ASH_JWT_SECRET", "dev-secret-change-me"),
		SecretKey:      envOr("ASH_SECRET_KEY", envOr("ASH_JWT_SECRET", "dev-secret-change-me")),
		DatabaseURL:    envOr("ASH_DATABASE_URL", ""),
		ArtifactStore:  envOr("ASH_ARTIFACT_STORE", "fs"),
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Region returns ASH_REGION for single-region identity (DX23).
// Empty / unset → "default". Not multi-region Active-Active.
func Region() string {
	v := strings.TrimSpace(os.Getenv("ASH_REGION"))
	if v == "" {
		return "default"
	}
	return v
}

func defaultPluginGRPCAddr() string {
	if envOr("ASH_AUTH_MODE", "dev") != "dev" {
		return ""
	}
	return "127.0.0.1:19091"
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
