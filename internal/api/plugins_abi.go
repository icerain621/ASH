package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/pluginabi"
)

const currentPluginABI = pluginabi.CurrentABI

// GetPluginABIProfile godoc
// @Summary Get plugin ABI profile
// @Tags plugins
// @Produce json
// @Success 200 {object} PluginABIProfileResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/plugins/abi [get]
func (h *Handler) getPluginABIProfile(c *gin.Context) {
	if !h.requirePermission(c, permPluginRead, currentSpace(c)) {
		return
	}
	files, err := pluginProtoFiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("PLUGIN_ABI_PROFILE_FAILED", err.Error()))
		return
	}
	cfg := config.Load()
	c.JSON(http.StatusOK, PluginABIProfileResponse{
		CurrentABI:         currentPluginABI,
		SupportedABIs:      []string{currentPluginABI},
		SupportedProtocols: []string{"grpc", "http", "mcp"},
		GRPCEnabled:        strings.TrimSpace(cfg.PluginGRPCAddr) != "",
		PluginGRPCAddr:     cfg.PluginGRPCAddr,
		ProtoPackage:       "ash.v1",
		GoPackage:          "github.com/ash-repwiki/ash/proto/ash/v1;ashv1",
		BreakingPolicy:     "buf:FILE",
		ProtoFiles:         files,
		SigningAlg:         pluginabi.SignAlgHMAC,
		SigningRequired:    pluginabi.SigningRequired(),
		SigningKeyConfigured: pluginabi.SigningKey() != "",
		SignCapabilityPrefix: pluginabi.CapabilitySignPrefix,
	})
}

func pluginProtoFiles() ([]PluginProtoFile, error) {
	root := resolvePluginProtoDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := make([]PluginProtoFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".proto" {
			continue
		}
		path := filepath.Join(root, entry.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(body)
		out = append(out, PluginProtoFile{
			Path:   filepath.ToSlash(filepath.Join("proto", "ash", "v1", entry.Name())),
			Digest: "sha256:" + hex.EncodeToString(sum[:]),
			Bytes:  int64(len(body)),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func resolvePluginProtoDir() string {
	for _, p := range []string{
		filepath.Join("proto", "ash", "v1"),
		filepath.Join("..", "proto", "ash", "v1"),
		filepath.Join("..", "..", "proto", "ash", "v1"),
	} {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return filepath.Join("proto", "ash", "v1")
}

func normalizePluginABI(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return currentPluginABI
	}
	return value
}
