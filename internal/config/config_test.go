package config

import "testing"

func TestLoadReadsPluginGRPCAddr(t *testing.T) {
	t.Setenv("ASH_PLUGIN_GRPC_ADDR", "127.0.0.1:19090")
	cfg := Load()
	if cfg.PluginGRPCAddr != "127.0.0.1:19090" {
		t.Fatalf("PluginGRPCAddr=%q want 127.0.0.1:19090", cfg.PluginGRPCAddr)
	}
}
