package hasher

import (
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

func TestInvalidHashFormat(t *testing.T) {
	s := NewService()

	// test with garbage string
	_, err := s.Verify("password", "not_a_hash")
	if err == nil {
		t.Error("Expected error for invalid hash format, got nil")
	}
}

func TestVerifyRejectsOversizedParams(t *testing.T) {
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
