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
	lower := strings.ToLower(dsn)
	for _, key := range []string{"password=", "secret="} {
		if idx := strings.Index(lower, key); idx >= 0 {
			return dsn[:idx+len(key)] + "***"
		}
	}
	return dsn
}
