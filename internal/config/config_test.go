package config

import "testing"

func TestLoadReadsPluginGRPCAddr(t *testing.T) {
	t.Setenv("ASH_PLUGIN_GRPC_ADDR", "127.0.0.1:19090")
	cfg := Load()
	if cfg.PluginGRPCAddr != "127.0.0.1:19090" {
		t.Fatalf("PluginGRPCAddr=%q want 127.0.0.1:19090", cfg.PluginGRPCAddr)
	}
}

func TestLoadRagIndexerGRPCAddrDefaultEmpty(t *testing.T) {
	t.Setenv("ASH_RAG_INDEXER_GRPC_ADDR", "")
	cfg := Load()
	if cfg.RagIndexerGRPCAddr != "" {
		t.Fatalf("RagIndexerGRPCAddr=%q want empty", cfg.RagIndexerGRPCAddr)
	}
	t.Setenv("ASH_RAG_INDEXER_GRPC_ADDR", "127.0.0.1:19092")
	cfg = Load()
	if cfg.RagIndexerGRPCAddr != "127.0.0.1:19092" {
		t.Fatalf("RagIndexerGRPCAddr=%q want 127.0.0.1:19092", cfg.RagIndexerGRPCAddr)
	}
}
