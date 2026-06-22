// Package auth provides authentication and authorization primitives.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024 // 64 MiB
	argonThreads uint8  = 2
	argonKeyLen  uint32 = 32
	saltLen      int    = 16
)

// HashPassword hashes a plaintext password using Argon2id and returns a
// self-describing encoded string.
func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf(
		"argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword checks whether password matches the encoded Argon2id hash.
func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 {
		return false, errors.New("invalid password hash format")
	}
	if parts[0] != "argon2id" || parts[1] != "v=19" {
		return false, errors.New("unsupported password hash algorithm/version")
	}

	var mem uint32
	var timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &mem, &timeCost, &threads); err != nil {
		return false, fmt.Errorf("parse password hash params: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, fmt.Errorf("decode salt: %w", err)
	}
	originalHash, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("decode hash: %w", err)
	}

	calculated := argon2.IDKey([]byte(password), salt, timeCost, mem, threads, uint32(len(originalHash)))
	if subtle.ConstantTimeCompare(originalHash, calculated) == 1 {
		return true, nil
	}
	return false, nil
}

// ValidatePasswordStrength applies a minimal policy used for bootstrap/admin creation.
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	var hasLetter bool
	var hasDigit bool
	for _, r := range password {
		if r >= '0' && r <= '9' {
			hasDigit = true
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			hasLetter = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("password must include letters and numbers")
	}
	return nil
}

// ParseInt64ID is a tiny helper for future handlers parsing path IDs.
func ParseInt64ID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
