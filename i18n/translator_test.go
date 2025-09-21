package i18n

import "testing"

func TestTranslator_DefaultAndJapanese(t *testing.T) {
	// default is en
	if msg := T("invalid_type", nil); msg == "invalid_type" || msg == "" {
		t.Fatalf("expected a human message, got %q", msg)
	}

	SetLanguage("ja")
	if msg := T("invalid_type", nil); msg == "invalid type" {
		t.Fatalf("expected japanese message, got %q", msg)
	}

	// reset to en
	SetLanguage("en")
}
