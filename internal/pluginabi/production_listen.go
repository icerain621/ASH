package pluginabi

import (
	"fmt"
	"os"
	"strings"
)

// ProductionPluginGRPCListen refuses a non-dev plugin gRPC bind unless
// signing is forced and a key is configured (DX73). Dev and empty addr pass.
func ProductionPluginGRPCListen(authMode, addr string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" || strings.TrimSpace(authMode) == "dev" {
		return nil
	}
	if SigningKey() == "" || os.Getenv("ASH_PLUGIN_SIGNING_REQUIRED") != "1" {
		return fmt.Errorf("refusing plugin grpc %s: non-dev requires ASH_PLUGIN_SIGNING_KEY and ASH_PLUGIN_SIGNING_REQUIRED=1", addr)
	}
	return nil
}
