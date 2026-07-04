package openapicheck

import (
	"fmt"
	"path/filepath"
)

var readyzHealthRequiredProps = []string{
	"sqlMigrationExpected",
	"postgresRLSPolicyExpected",
	"rlsCatalogSummary",
	"readinessWarnings",
	"sqlMigrationVersion",
	"schemaMode",
	"otelEnabled",
	"metricsEventReplayEnabled",
}

// ValidateReadyzContract ensures /readyz documents HealthResponse with RLS/SQL drift fields.
func ValidateReadyzContract(repoRoot string) error {
	contractPath := filepath.Join(repoRoot, contractRelPath)
	swaggerPath := filepath.Join(repoRoot, swaggerRelPath)

	contractNames, err := SchemaPropertyNames(contractPath, "HealthResponse")
	if err != nil {
		return err
	}
	swaggerNames, err := SwaggerDefinitionPropertyNames(swaggerPath, "internal_api.HealthResponse")
	if err != nil {
		return err
	}
	contractSet := toSet(contractNames)
	swaggerSet := toSet(swaggerNames)
	for _, prop := range readyzHealthRequiredProps {
		if _, ok := contractSet[prop]; !ok {
			return fmt.Errorf("contract HealthResponse missing property %q", prop)
		}
		if _, ok := swaggerSet[prop]; !ok {
			return fmt.Errorf("swagger HealthResponse missing property %q", prop)
		}
	}
	missingInContract, missingInSwagger := DiffPropertyNames(contractNames, swaggerNames)
	if len(missingInContract) > 0 || len(missingInSwagger) > 0 {
		return fmt.Errorf("HealthResponse drift: contract only=%v swagger only=%v",
			missingInSwagger, missingInContract)
	}

	contract, err := LoadPathMethods(contractPath)
	if err != nil {
		return err
	}
	if methods, ok := contract["/readyz"]; !ok {
		return fmt.Errorf("contract missing /readyz")
	} else if _, ok := methods["get"]; !ok {
		return fmt.Errorf("contract missing GET /readyz")
	}
	swagger, err := LoadPathMethods(swaggerPath)
	if err != nil {
		return err
	}
	if methods, ok := swagger["/readyz"]; !ok {
		return fmt.Errorf("swagger missing /readyz")
	} else if _, ok := methods["get"]; !ok {
		return fmt.Errorf("swagger missing GET /readyz")
	}
	return nil
}
