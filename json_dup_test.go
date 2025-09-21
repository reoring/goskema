package goskema

import "testing"

func TestDetectJSONDuplicateKeysBytes_NoDup(t *testing.T) {
	js := []byte(`{"a":1,"b":2}`)
	iss, err := DetectJSONDuplicateKeysBytes(js, Strictness{OnDuplicateKey: Warn}, -1)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(iss) != 0 {
		t.Fatalf("expected 0 issues, got %d: %v", len(iss), iss)
	}
}

func TestDetectJSONDuplicateKeysBytes_WithDup(t *testing.T) {
	js := []byte(`{"a":1,"a":2}`)
	iss, err := DetectJSONDuplicateKeysBytes(js, Strictness{OnDuplicateKey: Warn}, -1)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(iss) == 0 {
		t.Fatalf("expected duplicate_key issue")
	}
	if iss[0].Code != CodeDuplicateKey {
		t.Fatalf("expected duplicate_key, got %s", iss[0].Code)
	}
}
