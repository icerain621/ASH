package sqlmigrations

import "fmt"

// VerifyApplied checks golang-migrate state for Postgres.
func VerifyApplied(dsn string) error {
	v, dirty, err := VersionPostgres(dsn)
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("schema_migrations dirty at version %d", v)
	}
	if v < ExpectedVersion() {
		return fmt.Errorf("sql migration version %d < expected %d", v, ExpectedVersion())
	}
	return nil
}
