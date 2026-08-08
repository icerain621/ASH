package pluginabi

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/store"
	ashv1 "github.com/ash-repwiki/ash/proto/ash/v1"
)

const CurrentABI = "ash.plugin.v1"

type RegistryServer struct {
	ashv1.UnimplementedPluginRegistryServiceServer
	db *store.DB
}

func NewRegistryServer(db *store.DB) *RegistryServer {
	return &RegistryServer{db: db}
}

func (s *RegistryServer) Register(ctx context.Context, req *ashv1.RegisterRequest) (*ashv1.RegisterResponse, error) {
	if req == nil {
		return &ashv1.RegisterResponse{Accepted: false, Compatible: false, Status: status("INVALID_REQUEST", "request is required")}, nil
	}
	spaceID := traceSpace(req.GetContext())
	protocol := normalize(req.GetProtocol(), "grpc")
	abi := normalize(req.GetAbi(), CurrentABI)
	compatible, reason := Compatible(protocol, abi, req.GetName(), req.GetVersion())
	sig := strings.TrimSpace(req.GetSignature())
	if sig == "" {
		sig = SignatureFromCapabilities(req.GetCapabilities())
	}
	if err := VerifyRegistrationSignature(sig, req.GetName(), req.GetVersion(), protocol, abi, req.GetEndpoint()); err != nil {
		compatible = false
		if reason == "" {
			reason = err.Error()
		} else {
			reason = reason + "; " + err.Error()
		}
	}
	pluginID := strings.TrimSpace(req.GetId())
	if pluginID == "" {
		pluginID = "plg_" + uuid.NewString()
	}
	capabilities, _ := json.Marshal(req.GetCapabilities())
	now := time.Now().UTC()
	row := store.PluginRegistry{
		ID:           pluginID,
		SpaceID:      spaceID,
		Name:         strings.TrimSpace(req.GetName()),
		Version:      strings.TrimSpace(req.GetVersion()),
		Protocol:     protocol,
		ABI:          abi,
		Endpoint:     strings.TrimSpace(req.GetEndpoint()),
		Capabilities: string(capabilities),
		Compatible:   compatible,
		Status:       "registered",
		LastError:    reason,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if !compatible {
		row.Status = "incompatible"
	}
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	if !compatible {
		code := "INCOMPATIBLE"
		if strings.Contains(reason, "signature") || strings.Contains(reason, "SIGNING") {
			code = "PLUGIN_SIGNATURE_INVALID"
		}
		return &ashv1.RegisterResponse{
			Accepted: false, Compatible: false, PluginId: pluginID,
			Status: status(code, reason),
		}, nil
	}
	return &ashv1.RegisterResponse{
		Accepted: true, Compatible: true, PluginId: pluginID,
		Status: status("OK", "plugin registered"),
	}, nil
}

func (s *RegistryServer) Heartbeat(ctx context.Context, req *ashv1.HeartbeatRequest) (*ashv1.HeartbeatResponse, error) {
	row, err := s.plugin(ctx, req.GetPluginId(), traceSpace(req.GetContext()))
	if err != nil {
		return &ashv1.HeartbeatResponse{Ok: false, Status: status("PLUGIN_NOT_FOUND", "plugin not found")}, nil
	}
	row.UpdatedAt = time.Now().UTC()
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	return &ashv1.HeartbeatResponse{Ok: row.Compatible, Status: status("OK", row.Status)}, nil
}

func (s *RegistryServer) GetStatus(ctx context.Context, req *ashv1.GetStatusRequest) (*ashv1.GetStatusResponse, error) {
	row, err := s.plugin(ctx, req.GetPluginId(), traceSpace(req.GetContext()))
	if err != nil {
		return &ashv1.GetStatusResponse{
			PluginId: req.GetPluginId(),
			State:    "missing",
			Status:   status("PLUGIN_NOT_FOUND", "plugin not found"),
		}, nil
	}
	return &ashv1.GetStatusResponse{
		PluginId:   row.ID,
		State:      row.Status,
		Compatible: row.Compatible,
		Status:     status("OK", row.LastError),
	}, nil
}

func (s *RegistryServer) plugin(ctx context.Context, pluginID, spaceID string) (store.PluginRegistry, error) {
	var row store.PluginRegistry
	err := s.db.WithContext(ctx).First(&row, "id = ? AND space_id = ?", strings.TrimSpace(pluginID), spaceID).Error
	return row, err
}

func Compatible(protocol, abi, name, version string) (bool, string) {
	switch normalize(protocol, "grpc") {
	case "grpc", "http", "mcp":
	default:
		return false, "unsupported protocol"
	}
	if normalize(abi, CurrentABI) != CurrentABI {
		return false, "unsupported abi"
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(version) == "" {
		return false, "name and version are required"
	}
	return true, ""
}

func traceSpace(ctx *ashv1.TraceContext) string {
	if ctx == nil || strings.TrimSpace(ctx.GetSpaceId()) == "" {
		return "local"
	}
	return strings.TrimSpace(ctx.GetSpaceId())
}

func normalize(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	return value
}

func status(code, message string) *ashv1.Status {
	return &ashv1.Status{Code: code, Message: message}
}
