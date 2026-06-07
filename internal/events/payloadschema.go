package events

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ash-repwiki/ash/internal/schemayaml"
)

const payloadSchemaID = "ash.events.payloads.tr0/v0.1"

//go:embed schemas/event-payloads-tr0.v0.1.schema.json
var tr0PayloadSchemaJSON []byte

// PayloadValidationEnabled reports whether Append validates TR0 payload schemas.
func PayloadValidationEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ASH_VALIDATE_EVENT_PAYLOADS"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// ValidatePayload checks eventType/payload against the embedded TR0 schema $defs.
// Unknown event types without a $def are accepted.
func ValidatePayload(eventType string, payloadJSON []byte) error {
	var doc any
	if err := json.Unmarshal(payloadJSON, &doc); err != nil {
		return fmt.Errorf("payload json: %w", err)
	}
	issues := schemayaml.ValidateJSONDef(tr0PayloadSchemaJSON, payloadSchemaID, eventType, doc)
	if len(issues) == 0 {
		return nil
	}
	issue := issues[0]
	if issue.Path != "" {
		return fmt.Errorf("event %s payload %s: %s", eventType, issue.Path, issue.Message)
	}
	return fmt.Errorf("event %s payload: %s", eventType, issue.Message)
}

// KnownTR0EventTypes returns event types with embedded payload schemas.
func KnownTR0EventTypes() ([]string, error) {
	return schemayaml.KnownDefs(tr0PayloadSchemaJSON)
}
