package codec

import (
	"context"
	"testing"
	"time"

	goskema "github.com/reoring/goskema"
)

func TestTimeRFC3339_Codec_Basic(t *testing.T) {
	c := TimeRFC3339()
	ctx := context.Background()

	in := "2025-01-01T00:00:00Z"
	got, err := c.Decode(ctx, in)
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	if !got.Equal(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected time: %v", got)
	}

	out, err := c.Encode(ctx, got)
	if err != nil {
		t.Fatalf("encode err: %v", err)
	}
	if out != in {
		t.Fatalf("roundtrip mismatch: %s != %s", out, in)
	}
}

func TestTimeRFC3339_EncodePreserving_Scalar(t *testing.T) {
	c := TimeRFC3339()
	ctx := context.Background()

	// decode with meta to get presence
	dx, err := c.DecodeWithMeta(ctx, "2025-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("decode with meta err: %v", err)
	}
	// change value and encode preserving
	dx.Value = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	s, err := c.EncodePreserving(ctx, dx)
	if err != nil {
		t.Fatalf("encode preserving err: %v", err)
	}
	if s == "" {
		t.Fatalf("expected non-empty output")
	}
}

func TestTimeRFC3339_EncodePreserving_PresenceWasNull_Error(t *testing.T) {
	c := TimeRFC3339()
	ctx := context.Background()

	dx := goskema.Decoded[time.Time]{
		Value:    time.Time{},
		Presence: goskema.PresenceMap{"/": goskema.PresenceWasNull | goskema.PresenceSeen},
	}
	if _, err := c.EncodePreserving(ctx, dx); err == nil {
		t.Fatalf("expected invalid_type when PresenceWasNull is set")
	}
}

func TestTimeRFC3339_EncodePreserving_NotSeen_Error(t *testing.T) {
	c := TimeRFC3339()
	ctx := context.Background()

	// Presence metadata exists but the root PresenceSeen bit is not set.
	dx := goskema.Decoded[time.Time]{
		Value:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Presence: goskema.PresenceMap{"/": 0},
	}
	if _, err := c.EncodePreserving(ctx, dx); err == nil {
		t.Fatalf("expected required error when PresenceSeen is not set")
	}
}

func TestTimeRFC3339_EncodePreserving_DecodeWithMeta_Roundtrip(t *testing.T) {
	c := TimeRFC3339()
	ctx := context.Background()

	dx, err := c.DecodeWithMeta(ctx, "2025-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("decode with meta err: %v", err)
	}
	s, err := c.EncodePreserving(ctx, dx)
	if err != nil {
		t.Fatalf("encode preserving err: %v", err)
	}
	if s != "2025-01-01T00:00:00Z" {
		t.Fatalf("unexpected preserving output: %q", s)
	}
}
