package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func PKCEVerifier() string {
	// 43..128 chars. base64url without padding.
	return RandB64URL(64)
}

func PKCEChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func RandB64URL(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
