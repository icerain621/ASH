package registry

import "time"

const (
	StatusDraft    = "draft"
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

var AgentKinds = map[string]struct{}{
	"harness_profile": {},
	"adapter":         {},
	"skill":           {},
	"tool":            {},
	"plugin":          {},
}

var MemoryKinds = map[string]struct{}{
	"record": {},
	"skill":  {},
	"pack":   {},
	"edge":   {},
}

var AssetStatuses = map[string]struct{}{
	StatusDraft:    {},
	StatusActive:   {},
	StatusDisabled: {},
}

// AgentAssetView is the API/DTO for agent registry rows.
type AgentAssetView struct {
	ID        string `json:"id"`
	SpaceID   string `json:"spaceId"`
	Kind      string `json:"kind"`
	RefID     string `json:"refId,omitempty"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"createdAt"`
}

// MemoryAssetView is the API/DTO for memory registry rows.
type MemoryAssetView struct {
	ID        string `json:"id"`
	SpaceID   string `json:"spaceId"`
	Kind      string `json:"kind"`
	RefID     string `json:"refId,omitempty"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"createdAt"`
}

type CreateAgentAssetRequest struct {
	SpaceID string `json:"spaceId,omitempty"`
	Kind    string `json:"kind" binding:"required"`
	RefID   string `json:"refId,omitempty"`
	Name    string `json:"name" binding:"required"`
	Status  string `json:"status,omitempty"`
}

type CreateMemoryAssetRequest struct {
	SpaceID string `json:"spaceId,omitempty"`
	Kind    string `json:"kind" binding:"required"`
	RefID   string `json:"refId,omitempty"`
	Name    string `json:"name" binding:"required"`
	Status  string `json:"status,omitempty"`
}

type PatchStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type AgentAssetListResponse struct {
	Items []AgentAssetView `json:"items"`
}

type MemoryAssetListResponse struct {
	Items []MemoryAssetView `json:"items"`
}

func ms(t time.Time) int64 { return t.UTC().UnixMilli() }
