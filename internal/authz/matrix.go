package authz

import (
	"encoding/json"

	"github.com/ash-repwiki/ash/internal/store"
)

// OrgRoleRow is a custom organization role in the matrix response.
type OrgRoleRow struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

// MatrixResponse is the M2 permission matrix payload.
type MatrixResponse struct {
	SpaceID         string              `json:"spaceId"`
	Catalog         []PermissionDef     `json:"catalog"`
	BuiltinRoles    []BuiltinRole       `json:"builtinRoles"`
	OrgRoles        []OrgRoleRow        `json:"orgRoles,omitempty"`
	ScenarioTools   []ScenarioMatrixRow `json:"scenarioTools"`
	CurrentRole     string              `json:"currentRole,omitempty"`
	CurrentActor    string              `json:"currentActor,omitempty"`
}

// BuildMatrix assembles the permission matrix for a space.
func BuildMatrix(db *store.DB, spaceID, orgID, actorRole, actorID string) (*MatrixResponse, error) {
	scenarios, err := ScenarioPoliciesForSpace(db, spaceID)
	if err != nil {
		return nil, err
	}
	resp := &MatrixResponse{
		SpaceID:       spaceID,
		Catalog:       Catalog(),
		BuiltinRoles:  BuiltinRoles(),
		ScenarioTools: scenarios,
		CurrentRole:   actorRole,
		CurrentActor:  actorID,
	}
	if orgID == "" || db == nil {
		return resp, nil
	}
	var roles []store.Role
	if err := db.Where("org_id = ?", orgID).Order("created_at asc").Find(&roles).Error; err != nil {
		return nil, err
	}
	for _, row := range roles {
		resp.OrgRoles = append(resp.OrgRoles, OrgRoleRow{
			ID: row.ID, Name: row.Name, Permissions: parsePermissionList(row.Permissions),
		})
	}
	return resp, nil
}

func parsePermissionList(raw string) []string {
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err == nil {
		return values
	}
	if raw != "" {
		return []string{raw}
	}
	return nil
}
