package hasher

import (
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

func TestInvalidHashFormat(t *testing.T) {
	s := NewService()

	// test with garbage string
	_, err := s.Verify("password", "not_a_hash")
	if err == nil {
		t.Error("Expected error for invalid hash format, got nil")
	}
}
