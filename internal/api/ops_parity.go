package api

import "fmt"

// AssertReadyzScaleParity checks shared ops fields between /readyz and /api/v1/scale/readiness.
func AssertReadyzScaleParity(readyz HealthResponse, scale ScaleReadinessResponse) error {
	if readyz.Dialect != scale.DatabaseDialect {
		return fmt.Errorf("dialect readyz=%q scale=%q", readyz.Dialect, scale.DatabaseDialect)
	}
	if readyz.OtelEnabled != scale.OtelEnabled {
		return fmt.Errorf("otel readyz=%v scale=%v", readyz.OtelEnabled, scale.OtelEnabled)
	}
	if readyz.MetricsEventReplayEnabled != scale.MetricsEventReplayEnabled {
		return fmt.Errorf("metricsReplay readyz=%v scale=%v", readyz.MetricsEventReplayEnabled, scale.MetricsEventReplayEnabled)
	}
	if (scale.SQLMigrationExpected > 0 || readyz.SQLMigrationExpected > 0) &&
		scale.SQLMigrationExpected != readyz.SQLMigrationExpected {
		return fmt.Errorf("sqlExpected readyz=%d scale=%d", readyz.SQLMigrationExpected, scale.SQLMigrationExpected)
	}
	if readyz.SchemaMode != "" && scale.SchemaMode != "" && readyz.SchemaMode != scale.SchemaMode {
		return fmt.Errorf("schemaMode readyz=%q scale=%q", readyz.SchemaMode, scale.SchemaMode)
	}
	if scale.PostgresRLSEnabled && readyz.PostgresRLSEnabled {
		if scale.PostgresRLSPolicyExpected > 0 && readyz.PostgresRLSPolicyExpected > 0 &&
			scale.PostgresRLSPolicyExpected != readyz.PostgresRLSPolicyExpected {
			return fmt.Errorf("rlsExpected readyz=%d scale=%d", readyz.PostgresRLSPolicyExpected, scale.PostgresRLSPolicyExpected)
		}
	}
	return nil
}
