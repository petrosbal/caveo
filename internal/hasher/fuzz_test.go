package hasher

import "testing"

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
