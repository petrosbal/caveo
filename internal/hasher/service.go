// contains config and service structure,
// as well as funcs like service init, hash and verify

package hasher

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math"
	"strings"

	"golang.org/x/crypto/argon2"
)

// OWASP-recommended defaults
// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
const (
	TargetMemory      = 19 * 1024 //19MB
	TargetIterations  = 2
	TargetParallelism = 1
	TargetSaltLength  = 16
	TargetKeyLength   = 32
)

const (
	MaxMemory      = 256 * 1024 //256MB
	MaxIterations  = 32
	MaxParallelism = 16
)

// holds the argon2 params
// these settings determine computational cost
type config struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

// handles the password hash ops
type Service struct {
	config config
}

// initializes the service with the const OWASP defaults
func NewService() *Service {
	return &Service{
		config: config{
			memory:      TargetMemory,
			iterations:  TargetIterations,
			parallelism: TargetParallelism,
			saltLength:  TargetSaltLength,
			keyLength:   TargetKeyLength,
		},
	}
}

// uses argon2id to generate a hash string, given a password
func (s *Service) Hash(password string) (string, error) {

	//salt generation
	salt := make([]byte, s.config.saltLength)
	//filling the salt slice with random bytes
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	//password hash
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		s.config.iterations,
		s.config.memory,
		s.config.parallelism,
		s.config.keyLength,
	)

	//encode to base64
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	//return formatted string
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		s.config.memory,
		s.config.iterations,
		s.config.parallelism,
		b64Salt,
		b64Hash,
	)

	return encoded, nil
}

// check if a password matches a certain hash
// it parses the params directly from it
func (s *Service) Verify(password, encodedHash string) (bool, error) {

	//split the hash to extract components
	parts := strings.Split(encodedHash, "$")

	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format")
	}

	//parts[1] - algorithm checking
	if parts[1] != "argon2id" {
		return false, fmt.Errorf("unsupported algorithm: %s", parts[1])
	}
	//parts[2] - version checking
	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, fmt.Errorf("incompatible version format")
	}
	if version != argon2.Version {
		return false, fmt.Errorf("unsupported argon2 version: %d", version)
	}

	//parts[3] - config params
	var memory, iterations uint32
	var parallelism uint8

	n, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil || n != 3 {
		return false, fmt.Errorf("failed to parse parameters: %v", err)
	}

	if memory < 1 || memory > MaxMemory ||
		iterations < 1 || iterations > MaxIterations ||
		parallelism < 1 || parallelism > MaxParallelism {
		return false, fmt.Errorf("hash parameters out of range")
	}

	//decode salt and hash (base64->raw bytes)
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("salt decode error: %v", err)
	}
	storedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("hash decode error: %v", err)
	}
	if len(storedHash) > math.MaxUint32 {
		return false, fmt.Errorf("stored hash length exceeds maximum representable key length")
	}

	//rehash password with parsed params
	newHash := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		parallelism,
		uint32(len(storedHash)), //nolint:gosec // exact conversion: bounds-checked above, never truncates
	)

	return subtle.ConstantTimeCompare(storedHash, newHash) == 1, nil
}
