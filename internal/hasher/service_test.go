package hasher

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

// known-answer vectors from the Argon2 reference implementation.
// source: github.com/P-H-C/phc-winner-argon2, src/test.c (hashtest calls, Argon2_id).
// these prove Caveo agrees with the standard rather than only with itself
var referenceVectors = []struct {
	name     string
	password string
	encoded  string
}{
	{"baseline m=65536,t=2,p=1", "password", "$argon2id$v=19$m=65536,t=2,p=1$c29tZXNhbHQ$CTFhFdXPJO1aFaMaO6Mm5c8y7cJHAph8ArZWb2GRPPc"},
	{"memory at MaxMemory", "password", "$argon2id$v=19$m=262144,t=2,p=1$c29tZXNhbHQ$eP4eyR+zqlZX1y5xCFTkw9m5GYx0L5YWwvCFvtlbLow"},
	{"low memory m=256", "password", "$argon2id$v=19$m=256,t=2,p=1$c29tZXNhbHQ$nf65EOgLrQMR/uIPnA4rEsF5h7TKyQwu9U1bMCHGi/4"},
	{"parallelism p=2", "password", "$argon2id$v=19$m=256,t=2,p=2$c29tZXNhbHQ$bQk8UB/VmZZF4Oo79iDXuL5/0ttZwg2f/5U52iv1cDc"},
	{"single iteration t=1", "password", "$argon2id$v=19$m=65536,t=1,p=1$c29tZXNhbHQ$9qWtwbpyPd3vm1rB1GThgPzZ3/ydHL92zKL+15XZypg"},
	{"four iterations t=4", "password", "$argon2id$v=19$m=65536,t=4,p=1$c29tZXNhbHQ$kCXUjmjvc5XMqQedpMTsOv+zyJEf5PhtGiUghW9jFyw"},
	{"different password, same params and salt", "differentpassword", "$argon2id$v=19$m=65536,t=2,p=1$c29tZXNhbHQ$C4TWUs9rDEvq7w3+J4umqA32aWKB1+DSiRuBfYxFj94"},
	{"different salt, same params and password", "password", "$argon2id$v=19$m=65536,t=2,p=1$ZGlmZnNhbHQ$vfMrBczELrFdWP0ZsfhWsRPaHppYdP3MVEMIVlqoFBw"},
}

func TestVerifyReferenceVectors(t *testing.T) {
	s := NewService()

	for _, v := range referenceVectors {
		t.Run(v.name, func(t *testing.T) {
			match, err := s.Verify(v.password, v.encoded)
			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if !match {
				t.Error("want reference vector to verify, got no match")
			}
		})
	}
}

func TestVerifyReferenceVectorsRejectWrongPassword(t *testing.T) {
	s := NewService()

	for _, v := range referenceVectors {
		t.Run(v.name, func(t *testing.T) {
			match, err := s.Verify("not-the-password", v.encoded)
			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if match {
				t.Error("want no match for wrong password, got match")
			}
		})
	}
}

// this hash was generated with golang.org/x/crypto v0.47.0 and must keep
// verifying correctly no matter what version the module is on later.
// the hash is persistent data in consumers' databases.
// this is what enforces that compatibility
func TestPinnedHash(t *testing.T) {
	s := NewService()
	const (
		pinnedHash = "$argon2id$v=19$m=19456,t=2,p=1$yScNCPxKfF2BTmAD2q5uFA$hI3hVUeoEftGcC0X0NXGgYEHwrtnqzXdlxnZdb4qoSs"
		password   = "pinned-hash-password" //nolint:gosec // test fixture, not a credential
	)

	match, err := s.Verify(password, pinnedHash)
	if err != nil {
		t.Fatalf("Verify failed with error: %v", err)
	}
	if !match {
		t.Error("want pinned hash to still verify, but it didn't")
	}
}

func TestHashVerifyRoundTrip(t *testing.T) {
	s := NewService()
	password := "supersafepassword2000"

	hash, err := s.Hash(password)
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id") {
		t.Errorf("invalid hash format: %s", hash)
	}

	match, err := s.Verify(password, hash)
	if err != nil {
		t.Fatalf("Verify failed with error: %v", err)
	}
	if !match {
		t.Error("want password to match, but it didn't")
	}

	match, err = s.Verify("wrong_password", hash)
	if err != nil {
		t.Fatalf("Verify (negative) failed with error: %v", err)
	}
	if match {
		t.Error("want password NOT to match, but it did")
	}
}

func TestVerifyRejectsMalformedEncoding(t *testing.T) {
	s := NewService()

	valid, err := s.Hash("pw")
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}

	cases := []struct {
		name string
		evil string
	}{
		{"garbage string", "not_a_hash"},
		{"too few fields", "$argon2id$v=19$m=19456,t=2,p=1$c29tZXNhbHQ"},
		{"unsupported algorithm", replacePart(t, valid, 1, "argon2i")},
		{"unparseable version", replacePart(t, valid, 2, "v=abc")},
		{"unsupported version", replacePart(t, valid, 2, "v=16")},
		{"unparseable params", replacePart(t, valid, 3, "m=x,t=2,p=1")},
		{"missing a param", replacePart(t, valid, 3, "m=19456,t=2")},
		{"salt is not base64", replacePart(t, valid, 4, "!!!!!!!!!!!!")},
		{"tag is not base64", replacePart(t, valid, 5, "!!!!!!!!!!!!")},
		{"trailing garbage on p", replacePart(t, valid, 3, fmt.Sprintf("m=%d,t=%d,p=%dGARBAGE", TargetMemory, TargetIterations, TargetParallelism))},
		{"leading zero on memory", replacePart(t, valid, 3, fmt.Sprintf("m=0%d,t=%d,p=%d", TargetMemory, TargetIterations, TargetParallelism))},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			match, err := s.Verify("pw", c.evil)
			if err == nil {
				t.Errorf("want error for %s, got nil", c.name)
			}
			if match {
				t.Errorf("want no match on malformed input %q, got match", c.evil)
			}
		})
	}
}

func TestVerifyRejectsOutOfRangeParams(t *testing.T) {
	s := NewService()

	valid, err := s.Hash("pw")
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}

	cases := []struct {
		name    string
		find    string
		replace string
	}{
		{"memory over max", fmt.Sprintf("m=%d", TargetMemory), fmt.Sprintf("m=%d", MaxMemory+1)},
		{"iterations over max", fmt.Sprintf("t=%d", TargetIterations), fmt.Sprintf("t=%d", MaxIterations+1)},
		{"parallelism over max", fmt.Sprintf("p=%d", TargetParallelism), fmt.Sprintf("p=%d", MaxParallelism+1)},
		{"zero memory", fmt.Sprintf("m=%d", TargetMemory), "m=0"},
		{"zero iterations", fmt.Sprintf("t=%d", TargetIterations), "t=0"},
		{"zero parallelism", fmt.Sprintf("p=%d", TargetParallelism), "p=0"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			evil := strings.Replace(valid, c.find, c.replace, 1)
			_, err := s.Verify("pw", evil)
			if err == nil {
				t.Errorf("want error for %s, got nil", c.name)
			}
		})
	}
}

func TestVerifyRejectsUndersizedSaltAndHash(t *testing.T) {
	s := NewService()
	valid, err := s.Hash("pw")
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}

	b64 := func(n int) string {
		return base64.RawStdEncoding.EncodeToString(make([]byte, n))
	}

	cases := []struct {
		name string
		evil string
	}{
		{"empty salt", replacePart(t, valid, 4, "")},
		{"empty hash", replacePart(t, valid, 5, "")},
		{"salt one byte under min", replacePart(t, valid, 4, b64(MinSaltLength-1))},
		{"hash one byte under min", replacePart(t, valid, 5, b64(MinKeyLength-1))},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := s.Verify("pw", c.evil); err == nil {
				t.Errorf("want error for %s, got nil", c.name)
			}
		})
	}
}

func replacePart(t *testing.T, encoded string, idx int, value string) string {
	t.Helper()

	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		t.Fatalf("input is not a 6-part hash: %q", encoded)
	}
	parts[idx] = value
	return strings.Join(parts, "$")
}
