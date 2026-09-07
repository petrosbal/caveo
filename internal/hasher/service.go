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

// the PHC-style encoding that Hash produces and parseHash accepts.
// order: version, memory, iterations, parallelism, b64 salt, b64 key
const encodedFormat = "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s"

// OWASP-recommended defaults
// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
const (
	TargetMemory      = 19 * 1024 //19MiB (argon2 memory is in KiB)
	TargetIterations  = 2
	TargetParallelism = 1
	TargetSaltLength  = 16
	TargetKeyLength   = 32
)

const (
	MaxMemory      = 256 * 1024 //256MiB
	MaxIterations  = 32
	MaxParallelism = 16
	MinSaltLength  = 8 //RFC 9106
	MinKeyLength   = 4 //RFC 9106
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
	//filling the salt slice with random bytes. Read CANNOT return an error
	_, _ = rand.Read(salt)

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
		encodedFormat,
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

	p, err := parseHash(encodedHash)
	if err != nil {
		return false, err
	}

	//rehash password with parsed params
	newHash := argon2.IDKey(
		[]byte(password),
		p.salt,
		p.iterations,
		p.memory,
		p.parallelism,
		uint32(len(p.key)), //nolint:gosec // exact conversion: bounds-checked in parseHash, never truncates
	)

	return subtle.ConstantTimeCompare(p.key, newHash) == 1, nil
}

type parsedHash struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	salt        []byte
	key         []byte
}

// parses and validates an encoded hash
// on success the params are guaranteed to be safe for argon2.IDKey
func parseHash(encodedHash string) (parsedHash, error) {

	//split the hash to extract components
	parts := strings.Split(encodedHash, "$")

	if len(parts) != 6 {
		return parsedHash{}, fmt.Errorf("invalid hash format")
	}

	//parts[1] - algorithm checking
	if parts[1] != "argon2id" {
		return parsedHash{}, fmt.Errorf("unsupported algorithm: %s", parts[1])
	}
	//parts[2] - version checking
	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return parsedHash{}, fmt.Errorf("incompatible version format")
	}
	if version != argon2.Version {
		return parsedHash{}, fmt.Errorf("unsupported argon2 version: %d", version)
	}

	//parts[3] - config params
	var memory, iterations uint32
	var parallelism uint8

	n, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil || n != 3 {
		return parsedHash{}, fmt.Errorf("failed to parse parameters: %v", err)
	}

	if memory < 1 || memory > MaxMemory ||
		iterations < 1 || iterations > MaxIterations ||
		parallelism < 1 || parallelism > MaxParallelism {
		return parsedHash{}, fmt.Errorf("hash parameters out of range")
	}

	//decode salt and hash (base64->raw bytes)
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return parsedHash{}, fmt.Errorf("salt decode error: %v", err)
	}

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return parsedHash{}, fmt.Errorf("hash decode error: %v", err)
	}

	if len(salt) < MinSaltLength {
		return parsedHash{}, fmt.Errorf("salt too short: %d bytes", len(salt))
	}

	if len(key) < MinKeyLength {
		return parsedHash{}, fmt.Errorf("hash too short: %d bytes", len(key))
	}
	if len(key) > math.MaxUint32 {
		return parsedHash{}, fmt.Errorf("stored hash length exceeds maximum representable key length")
	}

	canonical := fmt.Sprintf(
		encodedFormat,
		version,
		memory,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	if canonical != encodedHash {
		return parsedHash{}, fmt.Errorf("hash is not in canonical form")
	}

	return parsedHash{
		memory:      memory,
		iterations:  iterations,
		parallelism: parallelism,
		salt:        salt,
		key:         key,
	}, nil
}
