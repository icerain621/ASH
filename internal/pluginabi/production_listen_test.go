package pluginabi

import "testing"

func TestProductionPluginGRPCListen(t *testing.T) {
	cases := []struct {
		name     string
		mode     string
		addr     string
		key      string
		required string
		ok       bool
	}{
		{name: "dev skips", mode: "dev", addr: "127.0.0.1:19091", ok: true},
		{name: "prod off", mode: "oidc", addr: "", key: "", required: "", ok: true},
		{name: "prod missing both", mode: "oidc", addr: "127.0.0.1:19091", ok: false},
		{name: "prod key only", mode: "oidc", addr: "127.0.0.1:19091", key: "k", ok: false},
		{name: "prod required only", mode: "oidc", addr: "127.0.0.1:19091", required: "1", ok: false},
		{name: "prod ready", mode: "oidc", addr: "127.0.0.1:19091", key: "k", required: "1", ok: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ASH_PLUGIN_SIGNING_KEY", tc.key)
			t.Setenv("ASH_PLUGIN_SIGNING_REQUIRED", tc.required)
			err := ProductionPluginGRPCListen(tc.mode, tc.addr)
			if tc.ok && err != nil {
				t.Fatal(err)
			}
			if !tc.ok && err == nil {
				t.Fatal("expected refusal")
			}
		})
	}
}
