package secretbox

import (
	"encoding/base64"
	"testing"
)

func testKey() string {
	return base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
}

func TestRoundTrip(t *testing.T) {
	box, err := New(testKey())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	enc, err := box.Encrypt("hello 团购券")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc == "" {
		t.Fatal("empty ciphertext")
	}
	plain, err := box.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if plain != "hello 团购券" {
		t.Fatalf("plain = %q", plain)
	}
}

func TestEmpty(t *testing.T) {
	box, err := New(testKey())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if enc, err := box.Encrypt(""); err != nil || enc != "" {
		t.Fatalf("Encrypt empty = %q, %v", enc, err)
	}
	if plain, err := box.Decrypt(""); err != nil || plain != "" {
		t.Fatalf("Decrypt empty = %q, %v", plain, err)
	}
}

func TestBadKey(t *testing.T) {
	if _, err := New("not-base64"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := New(base64.StdEncoding.EncodeToString([]byte("short"))); err == nil {
		t.Fatal("expected error")
	}
}
