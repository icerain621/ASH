package hooks

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ParseConfigJSON(body string) (Config, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return Config{}, nil
	}
	var cfg Config
	if err := json.Unmarshal([]byte(body), &cfg); err != nil {
		return Config{}, fmt.Errorf("hooks config: %w", err)
	}
	if err := validateConfigVersion(cfg.Version); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func ConfigFromSpaceBodyJSON(bodyJSON string) (Config, error) {
	bodyJSON = strings.TrimSpace(bodyJSON)
	if bodyJSON == "" || bodyJSON == "{}" {
		return Config{}, nil
	}
	var body struct {
		Hooks json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(bodyJSON), &body); err != nil {
		return Config{}, fmt.Errorf("space policy bodyJson: %w", err)
	}
	if len(body.Hooks) == 0 || string(body.Hooks) == "null" {
		return Config{}, nil
	}
	return ParseConfigJSON(string(body.Hooks))
}

func validateConfigVersion(version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return nil
	}
	if version != SchemaVersion {
		return fmt.Errorf("hooks config: version must be %q, got %q", SchemaVersion, version)
	}
	return nil
}
