package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// The cost of a new hash. Verification reads its parameters from the stored
// string, never from these, so changing them later locks nobody out.
const (
	memoryKiB = 19456
	passes    = 2
	lanes     = 1
	saltBytes = 16
	keyBytes  = 32
)

// hashSlots caps the argon2id work running at once. Sign-in is unauthenticated
// and expensive (19 MiB each); this is the only protection it has against a
// flood from many addresses, each of which would otherwise cost that much at
// the same moment in a container with a memory limit.
var hashSlots = make(chan struct{}, 4)

func acquire(ctx context.Context) (release func(), err error) {
	select {
	case hashSlots <- struct{}{}:
		return func() { <-hashSlots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

var b64 = base64.RawStdEncoding

// HashPassword returns an argon2id hash of plain, encoded as
// $argon2id$v=19$m=19456,t=2,p=1$<salt>$<key>. Call it before opening a
// transaction, never inside one: it takes tens of milliseconds and must not
// hold a row lock. It logs nothing. It waits for a hash slot as long as it
// takes; a request handler uses HashPasswordContext, which gives up when its
// request does.
func HashPassword(plain string) (string, error) {
	return HashPasswordContext(context.Background(), plain)
}

// HashPasswordContext is HashPassword that stops waiting for a hash slot when
// ctx ends (the request was cancelled or its deadline passed) and returns the
// context's error, so a flood of sign-ups cannot leave handlers queued behind
// work nobody is waiting for.
func HashPasswordContext(ctx context.Context, plain string) (string, error) {
	salt := make([]byte, saltBytes)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("auth: no randomness for a salt: %w", err)
	}
	release, err := acquire(ctx)
	if err != nil {
		return "", err
	}
	defer release()
	key := argon2.IDKey([]byte(plain), salt, passes, memoryKiB, lanes, keyBytes)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoryKiB, passes, lanes, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// phc is a parsed argon2id hash.
type phc struct {
	memory, passes uint32
	lanes          uint8
	salt, key      []byte
}

// parsePHC reads an encoded hash and refuses one outside what this service
// would ever make or accept: version 19, m at most 262144 KiB, t at most 10,
// p at most 16, a key of 16 to 64 bytes.
func parsePHC(s string) (phc, bool) {
	parts := strings.Split(s, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return phc{}, false
	}
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return phc{}, false
	}
	m, okM := number(params[0], "m=", 32)
	t, okT := number(params[1], "t=", 32)
	p, okP := number(params[2], "p=", 8)
	salt, errS := b64.DecodeString(parts[4])
	key, errK := b64.DecodeString(parts[5])
	if !okM || !okT || !okP || errS != nil || errK != nil {
		return phc{}, false
	}
	if p < 1 || p > 16 || t < 1 || t > 10 || m < 8*p || m > 262144 ||
		len(key) < 16 || len(key) > 64 || len(salt) < 8 || len(salt) > 64 {
		return phc{}, false
	}
	return phc{memory: uint32(m), passes: uint32(t), lanes: uint8(p), salt: salt, key: key}, true
}

func number(s, prefix string, bits int) (uint64, bool) {
	v, ok := strings.CutPrefix(s, prefix)
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseUint(v, 10, bits)
	return n, err == nil
}

// check runs one verification inside one of the hash slots. valid is false
// when encoded is not a hash this service accepts (nothing was computed).
func check(ctx context.Context, encoded, plain string) (match, valid bool, err error) {
	release, err := acquire(ctx)
	if err != nil {
		return false, false, err
	}
	defer release()
	p, ok := parsePHC(encoded)
	if !ok {
		return false, false, nil
	}
	got := argon2.IDKey([]byte(plain), p.salt, p.passes, p.memory, p.lanes, uint32(len(p.key)))
	return subtle.ConstantTimeCompare(got, p.key) == 1, true, nil
}

// verifyPassword is the service's verifier. A stored string that does not
// parse verifies false after running the dummy verification, so a damaged row
// costs the same time as any other attempt.
func (s *Service) verifyPassword(ctx context.Context, encoded, plain string) (bool, error) {
	match, valid, err := check(ctx, encoded, plain)
	if err != nil {
		return false, err
	}
	if !valid {
		_, _, err = check(ctx, s.dummy, plain)
		return false, err
	}
	return match, nil
}

// makeDummy makes the hash an attempt with no usable account is verified
// against: from 32 random bytes, with the current parameters.
func makeDummy() string {
	seed := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, seed); err != nil {
		panic("auth: no randomness for the dummy hash: " + err.Error())
	}
	h, err := HashPassword(hex.EncodeToString(seed))
	if err != nil {
		panic("auth: cannot make the dummy hash: " + err.Error())
	}
	return h
}
