package artifacts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// MarshalCanonicalJSON encodes v as canonical JSON:
// - object keys sorted lexicographically (recursive)
// - no insignificant whitespace
// - LF only (no CR)
// Used for JSON artifact digests (appendix F §5.3).
func MarshalCanonicalJSON(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var node any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&node); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := writeCanonical(&buf, node); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DigestCanonicalJSON returns sha256:<hex> of canonical JSON bytes.
func DigestCanonicalJSON(v any) (string, error) {
	b, err := MarshalCanonicalJSON(v)
	if err != nil {
		return "", err
	}
	return DigestBytes(b), nil
}

// DigestBytes returns sha256:<hex> for arbitrary bytes.
func DigestBytes(b []byte) string {
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}

func writeCanonical(buf *bytes.Buffer, node any) error {
	switch v := node.(type) {
	case nil:
		buf.WriteString("null")
		return nil
	case bool:
		if v {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil
	case json.Number:
		buf.WriteString(v.String())
		return nil
	case string:
		enc, err := json.Marshal(v)
		if err != nil {
			return err
		}
		buf.Write(enc)
		return nil
	case []any:
		buf.WriteByte('[')
		for i, item := range v {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonical(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
		return nil
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			encKey, err := json.Marshal(k)
			if err != nil {
				return err
			}
			buf.Write(encKey)
			buf.WriteByte(':')
			if err := writeCanonical(buf, v[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
		return nil
	default:
		return fmt.Errorf("unsupported canonical JSON type %T", node)
	}
}
