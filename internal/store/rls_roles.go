package store

import (
	"fmt"
	"strings"
)

// postgresRoleDDL runs idempotent role bootstrap statements (Postgres only).
func postgresRoleDDL(db *DB, roleName, password string) error {
	if db == nil || db.Dialect() != "postgres" {
		return nil
	}
	roleName = strings.TrimSpace(roleName)
	if roleName == "" {
		return fmt.Errorf("role name is required")
	}
	// Password is dev-only default; production should rotate via secrets manager.
	password = strings.TrimSpace(password)
	if password == "" {
		password = roleName
	}
	stmts := []string{
		fmt.Sprintf(`DO $$ BEGIN CREATE ROLE %s LOGIN PASSWORD '%s' NOINHERIT NOBYPASSRLS; EXCEPTION WHEN duplicate_object THEN NULL; END $$`, quoteIdent(roleName), escapeSQLLiteral(password)),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO %s`, quoteIdent(roleName)),
		fmt.Sprintf(`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %s`, quoteIdent(roleName)),
		fmt.Sprintf(`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s`, quoteIdent(roleName)),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %s`, quoteIdent(roleName)),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO %s`, quoteIdent(roleName)),
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("role %s: %w", roleName, err)
		}
	}
	return nil
}

func escapeSQLLiteral(v string) string {
	return strings.ReplaceAll(v, "'", "''")
}

// EnsurePostgresAppRole creates the non-owner application role for production RLS (ash_app).
func EnsurePostgresAppRole(db *DB) error {
	return postgresRoleDDL(db, "ash_app", "ash_app")
}

// EnsurePostgresRLSTestRole creates ash_rls_tester for integration tests (NOBYPASSRLS).
func EnsurePostgresRLSTestRole(db *DB) error {
	if db == nil || db.Dialect() != "postgres" {
		return nil
	}
	if err := db.Exec(`DROP ROLE IF EXISTS ash_rls_tester`).Error; err != nil {
		return err
	}
	return postgresRoleDDL(db, "ash_rls_tester", "ash_rls_tester")
}

// PostgresAppRoleExists reports whether ash_app is present.
func PostgresAppRoleExists(db *DB) (bool, error) {
	return postgresRoleExists(db, "ash_app")
}

// PostgresAppDatabaseURL builds the default dev ash_app connection string.
func PostgresAppDatabaseURL(host string, port int, database string) string {
	if strings.TrimSpace(host) == "" {
		host = "127.0.0.1"
	}
	if port <= 0 {
		port = 5432
	}
	if strings.TrimSpace(database) == "" {
		database = "ash"
	}
	return fmt.Sprintf("postgres://ash_app:ash_app@%s:%d/%s?sslmode=disable", host, port, database)
}

func postgresRoleExists(db *DB, roleName string) (bool, error) {
	if db == nil || db.Dialect() != "postgres" {
		return false, nil
	}
	var count int64
	err := db.Raw(`SELECT COUNT(*) FROM pg_roles WHERE rolname = ?`, roleName).Scan(&count).Error
	return count > 0, err
}
