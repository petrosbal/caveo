package hasher

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func TestWorkflow(t *testing.T) {
	// initialize
	s := NewService()
	password := "supersafepassword2000"

	// test hashing
	hash, err := s.Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	// check format (must start with $argon2id)
	if !strings.HasPrefix(hash, "$argon2id") {
		t.Errorf("Invalid hash format: %s", hash)
	}
	t.Logf("\nGenerated Hash: %s\n", hash)

	// test verification (good case)
	match, err := s.Verify(password, hash)
	if err != nil {
		t.Fatalf("Verify failed with error: %v", err)
	}
	if !match {
		t.Error("Expected password to match, but it didn't")
	}

	// test verification (bad case)
	match, err = s.Verify("wrong_password", hash)
	if err != nil {
		t.Fatalf("Verify (negative) failed with error: %v", err)
	}
	if match {
		t.Error("Expected password NOT to match, but it did")
	}
}

// This hash was generated with golang.org/x/crypto v0.47.0 and must keep
// verifying correctly regardless of which x/crypto version the module is
// on later. The hash format is persistent data in consumers' databases;
// this is what enforces that compatibility.
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
		t.Error("expected pinned hash to still verify, but it didn't")
	}
}

func TestInvalidHashFormat(t *testing.T) {
	s := NewService()

	// test with garbage string
	_, err := s.Verify("password", "not_a_hash")
	if err == nil {
		t.Error("Expected error for invalid hash format, got nil")
	}
}

func TestVerifyRejectsOutOfRangeParams(t *testing.T) {
	s := NewService()

	valid, _ := s.Hash("pw")
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
				t.Errorf("Expected error for %s, got nil", c.name)
			}
		})
	}
}

func TestVerifyRejectsUndersizedSaltAndHash(t *testing.T) {
	s := NewService()
	valid, _ := s.Hash("pw")

	replacePart := func(idx int, value string) string {
		parts := strings.Split(valid, "$")
		parts[idx] = value
		return strings.Join(parts, "$")
	}

	b64 := func(n int) string {
		return base64.RawStdEncoding.EncodeToString(make([]byte, n))
	}

	cases := []struct {
		name string
		evil string
	}{
		{"empty salt", replacePart(4, "")},
		{"empty hash", replacePart(5, "")},
		{"salt one byte under min", replacePart(4, b64(MinSaltLength-1))},
		{"hash one byte under min", replacePart(5, b64(MinKeyLength-1))},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := s.Verify("pw", c.evil); err == nil {
				t.Errorf("want error for %s, got nil", c.name)
			}
		})
	}
}
