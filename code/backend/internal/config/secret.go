package config

import (
	"log/slog"
	"strings"
)

const redacted = "[redacted]"

// Secret holds a value that must never reach a log line, an error or a response.
// It prints as [redacted] under %v, %+v, %#v, slog and encoding/json, so a Config
// that holds one can be logged by accident without leaking it. The only way to
// read the value is Reveal, which the code that signs with it calls.
type Secret struct{ value string }

// NewSecret trims the value; a blank one is "not set".
func NewSecret(value string) Secret { return Secret{value: strings.TrimSpace(value)} }

// IsSet says whether a non-blank value was given.
func (s Secret) IsSet() bool { return s.value != "" }

// Reveal returns the value. Only a signing function calls it.
func (s Secret) Reveal() string { return s.value }

func (Secret) String() string                { return redacted }
func (Secret) GoString() string              { return redacted }
func (Secret) LogValue() slog.Value          { return slog.StringValue(redacted) }
func (Secret) MarshalText() ([]byte, error)  { return []byte(redacted), nil }
