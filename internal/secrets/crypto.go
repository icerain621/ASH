package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

const envelopePrefix = "ash.v1."

func Seal(plaintext, keyMaterial string) (ciphertext, digest string, err error) {
	block, err := aes.NewCipher(secretKey(keyMaterial))
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return envelopePrefix + base64.RawStdEncoding.EncodeToString(sealed), Digest(plaintext), nil
}

func Open(ciphertext, keyMaterial string) (string, error) {
	if len(ciphertext) <= len(envelopePrefix) || ciphertext[:len(envelopePrefix)] != envelopePrefix {
		return "", fmt.Errorf("unsupported secret envelope")
	}
	raw, err := base64.RawStdEncoding.DecodeString(ciphertext[len(envelopePrefix):])
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(secretKey(keyMaterial))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid secret envelope")
	}
	nonce, body := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	opened, err := gcm.Open(nil, nonce, body, nil)
	if err != nil {
		return "", err
	}
	return string(opened), nil
}

func Digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func secretKey(keyMaterial string) []byte {
	sum := sha256.Sum256([]byte(keyMaterial))
	return sum[:]
}
