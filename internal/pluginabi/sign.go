package pluginabi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

const (
	// SignAlgHMAC is the MVP plugin packaging signature algorithm.
	SignAlgHMAC = "hmac-sha256"
	// CapabilitySignPrefix embeds a signature in gRPC Register capabilities when
	// the proto client has not yet populated the dedicated signature field.
	CapabilitySignPrefix = "ash.sign.hmac="
)

// SignMaterial is the canonical string hashed for plugin registration.
// Format: name\nversion\nprotocol\nabi\nendpoint
func SignMaterial(name, version, protocol, abi, endpoint string) string {
	return strings.Join([]string{
		strings.TrimSpace(name),
		strings.TrimSpace(version),
		normalize(protocol, "grpc"),
		normalize(abi, CurrentABI),
		strings.TrimSpace(endpoint),
	}, "\n")
}

// SignHMAC returns lowercase hex HMAC-SHA256 of SignMaterial with key.
func SignHMAC(key, name, version, protocol, abi, endpoint string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(SignMaterial(name, version, protocol, abi, endpoint)))
	return hex.EncodeToString(mac.Sum(nil))
}

// SigningKey returns ASH_PLUGIN_SIGNING_KEY (empty = unsigned allowed in soft mode).
func SigningKey() string {
	return strings.TrimSpace(os.Getenv("ASH_PLUGIN_SIGNING_KEY"))
}

// SigningRequired reports whether unsigned plugins must be rejected.
// True when ASH_PLUGIN_SIGNING_REQUIRED=1 or a signing key is configured.
func SigningRequired() bool {
	if os.Getenv("ASH_PLUGIN_SIGNING_REQUIRED") == "1" {
		return true
	}
	return SigningKey() != ""
}

// VerifyRegistrationSignature validates signature against ASH_PLUGIN_SIGNING_KEY.
// When no key is set and signing is not required, returns ok.
// When required but key missing, returns error (misconfiguration).
func VerifyRegistrationSignature(signature, name, version, protocol, abi, endpoint string) error {
	key := SigningKey()
	required := SigningRequired()
	sig := strings.TrimSpace(signature)
	sig = strings.TrimPrefix(strings.ToLower(sig), "sha256:")
	sig = strings.TrimPrefix(sig, SignAlgHMAC+":")

	if key == "" {
		if required {
			return fmt.Errorf("ASH_PLUGIN_SIGNING_REQUIRED=1 but ASH_PLUGIN_SIGNING_KEY is empty")
		}
		return nil
	}
	if sig == "" {
		return fmt.Errorf("plugin signature required (set signature or capability %s<hex>)", CapabilitySignPrefix)
	}
	want := SignHMAC(key, name, version, protocol, abi, endpoint)
	if !hmac.Equal([]byte(want), []byte(strings.ToLower(sig))) {
		return fmt.Errorf("plugin signature mismatch")
	}
	return nil
}

// SignatureFromCapabilities extracts ash.sign.hmac=<hex> from capability list.
func SignatureFromCapabilities(caps []string) string {
	for _, c := range caps {
		c = strings.TrimSpace(c)
		if strings.HasPrefix(strings.ToLower(c), CapabilitySignPrefix) {
			return strings.TrimSpace(c[len(CapabilitySignPrefix):])
		}
	}
	return ""
}
