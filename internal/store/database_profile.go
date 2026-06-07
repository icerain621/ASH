package store

import (
	"os"
	"strings"
)

// DatabaseTarget describes a resolved database connection.
type DatabaseTarget struct {
	Dialect string
	DSN     string
}

// DatabaseProfileInfo summarizes the active or configured database backend.
type DatabaseProfileInfo struct {
	Dialect            string `json:"dialect"`
	PostgresConfigured bool   `json:"postgresConfigured"`
	PostgresAppURL     bool   `json:"postgresAppUrlConfigured,omitempty"`
	MigrationReady     bool   `json:"migrationReady"`
	PostgresRLSEnabled bool   `json:"postgresRLSEnabled"`
	PostgresRLSForce   bool   `json:"postgresRLSForce"`
	PostgresRLSPolicyCount int64 `json:"postgresRLSPolicyCount,omitempty"`
	DSNHint            string `json:"dsnHint,omitempty"`
}

// ParseDatabaseTarget resolves ASH_DATABASE_URL (or dev SQLite default).
func ParseDatabaseTarget(dataDir, databaseURL string) (DatabaseTarget, error) {
	target, err := resolveDatabaseTarget(dataDir, databaseURL)
	if err != nil {
		return DatabaseTarget{}, err
	}
	return DatabaseTarget{Dialect: target.dialect, DSN: target.dsn}, nil
}

// DatabaseProfile builds a migration/readiness snapshot for doctor and ops.
func DatabaseProfile(dataDir, databaseURL string) (DatabaseProfileInfo, error) {
	raw := strings.TrimSpace(databaseURL)
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("ASH_DATABASE_URL"))
	}
	target, err := ParseDatabaseTarget(dataDir, raw)
	if err != nil {
		return DatabaseProfileInfo{}, err
	}
	profile := DatabaseProfileInfo{
		Dialect:            target.Dialect,
		PostgresConfigured: target.Dialect == "postgres",
		PostgresAppURL:     strings.TrimSpace(os.Getenv("ASH_DATABASE_APP_URL")) != "",
		MigrationReady:     target.Dialect == "sqlite" || target.Dialect == "postgres",
	}
	if target.Dialect == "postgres" {
		profile.DSNHint = redactDSN(target.DSN)
		profile.PostgresRLSEnabled = PostgresRLSEnabled()
		profile.PostgresRLSForce = PostgresRLSForce()
	}
	return profile, nil
}

func redactDSN(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return ""
	}
	if i := strings.Index(dsn, "://"); i >= 0 {
		rest := dsn[i+3:]
		if at := strings.Index(rest, "@"); at > 0 {
			creds := rest[:at]
			if colon := strings.Index(creds, ":"); colon >= 0 {
				return dsn[:i+3] + creds[:colon+1] + "***" + rest[at:]
			}
		}
	}
	lower := strings.ToLower(dsn)
	for _, key := range []string{"password=", "secret="} {
		if idx := strings.Index(lower, key); idx >= 0 {
			return dsn[:idx+len(key)] + "***"
		}
	}
	return dsn
}

// WorkerConnectionRole reports how the worker DB handle is expected to connect.
func WorkerConnectionRole() string {
	if strings.TrimSpace(os.Getenv("ASH_DATABASE_APP_URL")) != "" {
		return "ash_app"
	}
	if raw := strings.TrimSpace(os.Getenv("ASH_DATABASE_URL")); raw != "" {
		if target, err := ParseDatabaseTarget("", raw); err == nil && target.Dialect == "postgres" {
			return "owner"
		}
	}
	return "sqlite"
}
