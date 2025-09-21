package kubeopenapi

import (
	"bytes"
	"errors"
	"testing"
)

func TestStrictYAMLReader_DuplicateKey_Root(t *testing.T) {
	y := []byte("kind: A\nkind: B\n")
	r := NewStrictYAMLReader(bytes.NewReader(y))
	_, err := r.Next()
	if err == nil {
		t.Fatalf("expected duplicate key error")
	}
	var de *DuplicateKeyError
	if !errors.As(err, &de) {
		t.Fatalf("expected DuplicateKeyError, got %T %v", err, err)
	}
	if de.Key != "kind" {
		t.Fatalf("expected key=kind, got %q", de.Key)
	}
	if de.FirstLine <= 0 || de.Line <= 0 {
		t.Fatalf("expected positive line numbers, got first=%d dup=%d", de.FirstLine, de.Line)
	}
}

func TestStrictYAMLReader_DuplicateKey_Nested(t *testing.T) {
	y := []byte("metadata:\n  name: a\n  name: b\n")
	r := NewStrictYAMLReader(bytes.NewReader(y))
	_, err := r.Next()
	if err == nil {
		t.Fatalf("expected duplicate key error")
	}
	var de *DuplicateKeyError
	if !errors.As(err, &de) {
		t.Fatalf("expected DuplicateKeyError, got %T %v", err, err)
	}
	if de.Key != "name" {
		t.Fatalf("expected key=name, got %q", de.Key)
	}
}

func TestStrictYAMLReader_ReadAll_MultiDoc(t *testing.T) {
	y := []byte("kind: A\n---\nkind: B\n")
	r := NewStrictYAMLReader(bytes.NewReader(y))
	docs, err := r.ReadAll()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}
}
