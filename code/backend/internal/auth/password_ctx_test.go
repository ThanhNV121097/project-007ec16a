package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// A request whose context ends while every hash slot is busy stops waiting and
// gets the context's error; it neither hashes nor leaves a slot taken.
func TestHashPasswordContextGivesUpWhenItsRequestEnds(t *testing.T) {
	for range cap(hashSlots) {
		hashSlots <- struct{}{}
	}
	released := false
	release := func() {
		if !released {
			released = true
			for range cap(hashSlots) {
				<-hashSlots
			}
		}
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	hash, err := HashPasswordContext(ctx, "lanpass123")
	if !errors.Is(err, context.DeadlineExceeded) || hash != "" {
		t.Fatalf("hash %q, err %v", hash, err)
	}

	release()
	hash, err = HashPasswordContext(context.Background(), "lanpass123")
	if err != nil || !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("with free slots: %q, %v", hash, err)
	}
	if len(hashSlots) != 0 {
		t.Errorf("%d hash slots left taken", len(hashSlots))
	}
}
