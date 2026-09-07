package hasher

import (
	"encoding/base64"
	"fmt"
	"testing"

	"golang.org/x/crypto/argon2"
)

const validHash = "$argon2id$v=19$m=65536,t=2,p=1$c29tZXNhbHQ$CTFhFdXPJO1aFaMaO6Mm5c8y7cJHAph8ArZWb2GRPPc"

// this is the fast target, it never calls argon2
// it asserts the contract of the parser
// whatever parseHash accepts is safe to give to argon2.IDKey
func FuzzParseHash(f *testing.F) {

	f.Add(validHash)                                                                                       // the only seed that reaches the assertions
	f.Add("")                                                                                              // empty string
	f.Add("not_a_hash")                                                                                    // ...not a hash
	f.Add("$argon2id$v=19$m=65536,t=2,p=1$c29tZXNhbHQ")                                                    // five parts
	f.Add("$argon2id$v=19$m=65536,t=0,p=1$c29tZXNhbHQ$CTFhFdXPJO1aFaMaO6Mm5c8y7cJHAph8ArZWb2GRPPc")        // t=0 panicked argon2
	f.Add("$argon2id$v=19$m=65536,t=2,p=0$c29tZXNhbHQ$CTFhFdXPJO1aFaMaO6Mm5c8y7cJHAph8ArZWb2GRPPc")        // p=0 panicked argon2
	f.Add("$argon2id$v=19$m=0,t=2,p=1$c29tZXNhbHQ$CTFhFdXPJO1aFaMaO6Mm5c8y7cJHAph8ArZWb2GRPPc")            // argon2 silently raises m=0
	f.Add("$argon2id$v=19$m=65536,t=2,p=1$$CTFhFdXPJO1aFaMaO6Mm5c8y7cJHAph8ArZWb2GRPPc")                   // empty salt
	f.Add("$argon2id$v=19$m=65536,t=2,p=1$c29tZXNhbHQ$")                                                   // empty tag, nil deref in blake2b
	f.Add("$argon2id$v=19$m=65536,t=2,p=1GARBAGE$c29tZXNhbHQ$CTFhFdXPJO1aFaMaO6Mm5c8y7cJHAph8ArZWb2GRPPc") // was malleable

	f.Fuzz(func(t *testing.T, encoded string) {
		parsed, err := parseHash(encoded)

		if err != nil {
			return
		}

		if parsed.memory < 1 || parsed.memory > MaxMemory {
			t.Errorf("accepted out-of-range memory %d from %q", parsed.memory, encoded)
		}
		if parsed.iterations < 1 || parsed.iterations > MaxIterations {
			t.Errorf("accepted out-of-range iterations %d from %q", parsed.iterations, encoded)
		}
		if parsed.parallelism < 1 || parsed.parallelism > MaxParallelism {
			t.Errorf("accepted out-of-range parallelism %d from %q", parsed.parallelism, encoded)
		}
		if len(parsed.salt) < MinSaltLength {
			t.Errorf("accepted short salt of %d bytes from %q", len(parsed.salt), encoded)
		}
		if len(parsed.key) < MinKeyLength {
			t.Errorf("accepted short key of %d bytes from %q", len(parsed.key), encoded)
		}
	})
}

// slow target, calls argon2
// it asserts that a hash built correctly must verify
func FuzzVerify(f *testing.F) {

	s := NewService()

	// the seed is raw input that the modulo reshapes
	f.Add("password", uint32(1016), uint32(2), uint8(1), uint8(3), []byte("somesalt"))

	f.Fuzz(func(t *testing.T, password string, mRaw, tRaw uint32, pRaw, kRaw uint8, salt []byte) {
		memory := 8 + mRaw%1024
		iterations := 1 + tRaw%3
		parallelism := 1 + pRaw%4
		keyLen := uint32(4 + kRaw%60)

		for len(salt) < MinSaltLength {
			salt = append(salt, 0)
		}

		key := argon2.IDKey(
			[]byte(password),
			salt,
			iterations,
			memory,
			parallelism,
			keyLen,
		)

		encoded := fmt.Sprintf(
			encodedFormat,
			argon2.Version,
			memory,
			iterations,
			parallelism,
			base64.RawStdEncoding.EncodeToString(salt),
			base64.RawStdEncoding.EncodeToString(key),
		)

		match, err := s.Verify(password, encoded)
		if err != nil {
			t.Fatalf("Verify rejected a hash it should accept: %v (encoded %q)", err, encoded)
		}
		if !match {
			t.Errorf("want match for a correctly built hash, got none (encoded %q)", encoded)
		}
	})
}
