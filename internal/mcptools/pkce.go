package mcptools

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
)

// computeS256Challenge returns the PKCE S256 challenge for a verifier.
func computeS256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// verifyPKCE reports whether verifier matches the stored challenge (constant-time).
func verifyPKCE(verifier, challenge string) bool {
	got := computeS256Challenge(verifier)
	return subtle.ConstantTimeCompare([]byte(got), []byte(challenge)) == 1
}
