package pluginabi

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
	ashv1 "github.com/ash-repwiki/ash/proto/ash/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestRegistryServerRegisterHeartbeatStatus(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	client, cleanup := registryClient(t, db)
	defer cleanup()

	ctx := context.Background()
	reg, err := client.Register(ctx, &ashv1.RegisterRequest{
		Name:         "otel-exporter",
		Version:      "1.0.0",
		Protocol:     "grpc",
		Abi:          CurrentABI,
		Endpoint:     "bufconn",
		Capabilities: []string{"observability.export"},
		Context:      &ashv1.TraceContext{TraceId: "trc_test", SpaceId: "space_plugin"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reg.GetAccepted() || !reg.GetCompatible() || reg.GetPluginId() == "" {
		t.Fatalf("register=%+v want accepted compatible plugin id", reg)
	}

	var row store.PluginRegistry
	if err := db.First(&row, "id = ? AND space_id = ?", reg.GetPluginId(), "space_plugin").Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != "registered" || !row.Compatible || row.Protocol != "grpc" || row.ABI != CurrentABI {
		t.Fatalf("row=%+v want registered compatible grpc current ABI", row)
	}

	heartbeat, err := client.Heartbeat(ctx, &ashv1.HeartbeatRequest{
		PluginId: reg.GetPluginId(),
		Context:  &ashv1.TraceContext{SpaceId: "space_plugin"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !heartbeat.GetOk() || heartbeat.GetStatus().GetCode() != "OK" {
		t.Fatalf("heartbeat=%+v want OK", heartbeat)
	}

	status, err := client.GetStatus(ctx, &ashv1.GetStatusRequest{
		PluginId: reg.GetPluginId(),
		Context:  &ashv1.TraceContext{SpaceId: "space_plugin"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if status.GetState() != "registered" || !status.GetCompatible() {
		t.Fatalf("status=%+v want registered compatible", status)
	}
}

func TestRegistryServerRejectsIncompatiblePlugin(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	client, cleanup := registryClient(t, db)
	defer cleanup()

	reg, err := client.Register(context.Background(), &ashv1.RegisterRequest{
		Name:     "future-plugin",
		Version:  "2.0.0",
		Protocol: "grpc",
		Abi:      "ash.plugin.v2",
		Context:  &ashv1.TraceContext{SpaceId: "space_plugin"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if reg.GetAccepted() || reg.GetCompatible() || reg.GetStatus().GetCode() != "INCOMPATIBLE" {
		t.Fatalf("register=%+v want incompatible rejection", reg)
	}

	var row store.PluginRegistry
	if err := db.First(&row, "id = ? AND space_id = ?", reg.GetPluginId(), "space_plugin").Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != "incompatible" || row.Compatible || row.LastError == "" {
		t.Fatalf("row=%+v want persisted incompatible plugin", row)
	}
}

func TestStartRegistryServerListensOnTCP(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rt, err := StartRegistryServer("127.0.0.1:0", db)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(
		ctx,
		rt.Addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp, err := ashv1.NewPluginRegistryServiceClient(conn).Register(ctx, &ashv1.RegisterRequest{
		Name:     "tcp-plugin",
		Version:  "1.0.0",
		Protocol: "grpc",
		Abi:      CurrentABI,
		Endpoint: rt.Addr,
		Context:  &ashv1.TraceContext{SpaceId: "space_tcp_plugin"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetAccepted() || !resp.GetCompatible() || resp.GetPluginId() == "" {
		t.Fatalf("register=%+v want accepted compatible plugin", resp)
	}
}

func registryClient(t *testing.T, db *store.DB) (ashv1.PluginRegistryServiceClient, func()) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	ashv1.RegisterPluginRegistryServiceServer(srv, NewRegistryServer(db))
	go func() {
		_ = srv.Serve(lis)
	}()
	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		_ = conn.Close()
		srv.Stop()
		_ = lis.Close()
	}
	return ashv1.NewPluginRegistryServiceClient(conn), cleanup
}
