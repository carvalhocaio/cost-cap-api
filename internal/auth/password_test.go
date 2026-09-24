package auth_test

import (
	"errors"
	"testing"

	"github.com/carvalhocaio/cost-cap-api/internal/auth"
)

var fastParams = auth.Argon2idParams{
	Memory:      64,
	Iterations:  1,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

func TestArgon2idHasherRoundTrip(t *testing.T) {
	hasher := auth.NewArgon2idHasher(fastParams)

	encoded, err := hasher.Hash("stressed-out")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"matching password", "stressed-out", true},
		{"different password", "ride-slowly", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hasher.Verify(tt.password, encoded)
			if err != nil {
				t.Fatalf("verify: %v", err)
			}
			if got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArgon2idHasherUsesRandomSalt(t *testing.T) {
	hasher := auth.NewArgon2idHasher(fastParams)

	first, _ := hasher.Hash("stressed-out")
	second, _ := hasher.Hash("stressed-out")

	if first == second {
		t.Error("identical hashes for the same password, salt is not random")
	}
}

func TestArgon2idHasherVerifiesHashesFromOtherParams(t *testing.T) {
	legacy := auth.NewArgon2idHasher(fastParams)
	encoded, _ := legacy.Hash("stressed-out")

	current := auth.NewArgon2idHasher(auth.Argon2idParams{
		Memory: 128, Iterations: 2, Parallelism: 1, SaltLength: 16, KeyLength: 32,
	})

	got, err := current.Verify("stressed-out", encoded)
	if err != nil || !got {
		t.Fatalf("Verify() = %v, %v, want true, nil", got, err)
	}
}

func TestArgon2idHasherRejectsMalformedHashes(t *testing.T) {
	hasher := auth.NewArgon2idHasher(fastParams)

	tests := []struct {
		name    string
		encoded string
	}{
		{"empty", ""},
		{"wrong algorithm", "$bcrypt$v=19$m=64,t=1,p=1$c2FsdA$a2V5"},
		{"wrong version", "$argon2id$v=16$m=64,t=1,p=1$c2FsdA$a2V5"},
		{"broken parameters", "$argon2id$v=19$m=x,t=1,p=1$c2FsdA$a2V5"},
		{"invalid salt", "$argon2id$v=19$m=64,t=1,p=1$!!!$a2V5"},
		{"invalid key", "$argon2id$v=19$m=64,t=1,p=1$c2FsdA$!!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := hasher.Verify("stressed-out", tt.encoded); !errors.Is(err, auth.ErrMalformedHash) {
				t.Errorf("error = %v, want %v", err, auth.ErrMalformedHash)
			}
		})
	}
}
