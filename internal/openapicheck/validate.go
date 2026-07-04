package openapicheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	contractRelPath = "doc/api/openapi-ash-v1.yaml"
	swaggerRelPath  = "internal/api/docs/swagger.yaml"
	enforcedPrefix  = "/api/v1/"
	legacyPrefix    = "/v1/"
)

// ContractReport summarizes handwritten OpenAPI contract health.
type ContractReport struct {
	MissingPaths    []string
	GenericEnvelope []string
	LegacyPlanned   int
}

// DefaultRepoRoot locates the repository root (ASH_REPO_ROOT or walk-up for openapi draft).
func DefaultRepoRoot() (string, error) {
	if v := strings.TrimSpace(os.Getenv("ASH_REPO_ROOT")); v != "" {
		return v, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, contractRelPath)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("repo root not found (set ASH_REPO_ROOT or run from ash checkout)")
}

// ValidateContract checks enforced /api/v1 path coverage and 2xx JSON envelope usage.
func ValidateContract(repoRoot string) (ContractReport, error) {
	contractPath := filepath.Join(repoRoot, contractRelPath)
	swaggerPath := filepath.Join(repoRoot, swaggerRelPath)

	contract, err := LoadPathMethods(contractPath)
	if err != nil {
		return ContractReport{}, err
	}
	swagger, err := LoadPathMethods(swaggerPath)
	if err != nil {
		return ContractReport{}, err
	}
	rep := AlignContract(contract, swagger, enforcedPrefix, legacyPrefix)

	generic, err := FindGenericSuccessEnvelopeOps(contractPath, enforcedPrefix, "ApiResponse")
	if err != nil {
		return ContractReport{}, err
	}
	return ContractReport{
		MissingPaths:    rep.Missing,
		GenericEnvelope: generic,
		LegacyPlanned:   len(rep.LegacyPlanned),
	}, nil
}

// ValidateContractOrError returns nil when the contract is aligned with swag output.
func ValidateContractOrError(repoRoot string) error {
	rep, err := ValidateContract(repoRoot)
	if err != nil {
		return err
	}
	if len(rep.MissingPaths) > 0 {
		return fmt.Errorf("contract missing in swagger:\n%s", strings.Join(rep.MissingPaths, "\n"))
	}
	if len(rep.GenericEnvelope) > 0 {
		return fmt.Errorf("/api/v1 2xx must not use ApiResponse:\n%s", strings.Join(rep.GenericEnvelope, "\n"))
	}
	return nil
}
