package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strings"

	"golang.org/x/crypto/argon2"
)

var ErrMalformedHash = errors.New("malformed password hash")

var encoding = base64.RawStdEncoding

type Argon2idParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var DefaultArgon2idParams = Argon2idParams{
	Memory:      19 * 1024,
	Iterations:  2,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

type Argon2idHasher struct {
	params Argon2idParams
}

func NewArgon2idHasher(params Argon2idParams) *Argon2idHasher {
	return &Argon2idHasher{params: params}
}

func (h *Argon2idHasher) Hash(password string) (string, error) {
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, h.params.Iterations, h.params.Memory, h.params.Parallelism, h.params.KeyLength)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.params.Memory, h.params.Iterations, h.params.Parallelism,
		encoding.EncodeToString(salt),
		encoding.EncodeToString(key),
	), nil
}

func (h *Argon2idHasher) Verify(password, encoded string) (bool, error) {
	decoded, err := decodeArgon2id(encoded)
	if err != nil {
		return false, err
	}

	candidate := argon2.IDKey([]byte(password), decoded.salt, decoded.params.Iterations, decoded.params.Memory, decoded.params.Parallelism, decoded.params.KeyLength)

	return subtle.ConstantTimeCompare(decoded.key, candidate) == 1, nil
}

type argon2idHash struct {
	params Argon2idParams
	salt   []byte
	key    []byte
}

func decodeArgon2id(encoded string) (argon2idHash, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argon2idHash{}, fmt.Errorf("%w: unexpected format", ErrMalformedHash)
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return argon2idHash{}, fmt.Errorf("%w: unsupported version", ErrMalformedHash)
	}

	var decoded argon2idHash
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &decoded.params.Memory, &decoded.params.Iterations, &decoded.params.Parallelism); err != nil {
		return argon2idHash{}, fmt.Errorf("%w: invalid parameters", ErrMalformedHash)
	}

	var err error
	if decoded.salt, err = encoding.DecodeString(parts[4]); err != nil {
		return argon2idHash{}, fmt.Errorf("%w: invalid salt", ErrMalformedHash)
	}
	if decoded.key, err = encoding.DecodeString(parts[5]); err != nil {
		return argon2idHash{}, fmt.Errorf("%w: invalid key", ErrMalformedHash)
	}

	saltLen := len(decoded.salt)
	if saltLen < 0 || saltLen > math.MaxUint32 {
		return argon2idHash{}, fmt.Errorf("%w: salt length out of bounds", ErrMalformedHash)
	}
	decoded.params.SaltLength = uint32(saltLen)

	keyLen := len(decoded.key)
	if keyLen < 0 || keyLen > math.MaxUint32 {
		return argon2idHash{}, fmt.Errorf("%w: key length out of bounds", ErrMalformedHash)
	}
	decoded.params.KeyLength = uint32(keyLen)

	return decoded, nil
}
