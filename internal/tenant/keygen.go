package tenant

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	// Key prefixes for different environments
	testKeyPrefix = "payd_test_"
	liveKeyPrefix = "payd_live_"

	// Argon2id parameters (same as internal/middleware/auth.go)
	argonTime    uint32 = 1
	argonMemory  uint32 = 64 * 1024 // 64MB
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32
)

// GenerateAPIKey creates a new API key with cryptographically secure randomness.
// Returns: plaintext (full key to show user once), prefix (for lookup), hash (to store), error
func GenerateAPIKey(env string) (plaintext, prefix, hash string, err error) {
	// Select prefix based on environment
	var keyPrefix string
	if env == "production" {
		keyPrefix = liveKeyPrefix
	} else {
		keyPrefix = testKeyPrefix
	}

	// Generate 32 random bytes for the key suffix
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", "", fmt.Errorf("generating random bytes: %w", err)
	}

	// Encode random bytes as base64 for readability
	randomSuffix := base64.URLEncoding.EncodeToString(randomBytes)

	// Full plaintext key (what we show to the user once)
	plaintext = keyPrefix + randomSuffix

	// Key prefix for database lookup (first 12 chars)
	prefix = plaintext[:12]

	// Hash using Argon2id (same algorithm as auth middleware)
	hashBytes := argon2.IDKey([]byte(plaintext), []byte(prefix), argonTime, argonMemory, argonThreads, argonKeyLen)
	hash = base64.StdEncoding.EncodeToString(hashBytes)

	return plaintext, prefix, hash, nil
}

// ValidateKeyFormat checks if a key string matches the expected format
func ValidateKeyFormat(key string) bool {
	if len(key) < 20 {
		return false
	}
	// Must start with payd_test_ or payd_live_
	return (len(key) > len(testKeyPrefix) && key[:len(testKeyPrefix)] == testKeyPrefix) ||
		(len(key) > len(liveKeyPrefix) && key[:len(liveKeyPrefix)] == liveKeyPrefix)
}
