package util

import (
	"crypto/sha1" //nolint:gosec // required for compatibility
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/rand"
)

// Returns the base 64 encoded sha1 hash of the given string
func EncodeBase64Sha1(str string) string {
	hash := sha1.Sum([]byte(str)) //nolint:gosec // required for compatibility
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func GenerateRandomSha256() (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes) //nolint:staticcheck,gosec // migration to crypto/rand is a separate task
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256(randomBytes)), nil
}
