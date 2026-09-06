package repository

import (
	"strings"
	"testing"
)

func TestSecretBoxRoundTrip(t *testing.T) {
	for _, keyLen := range []int{16, 24, 32} {
		t.Run(string(rune('0'+keyLen/8))+"-byte-key", func(t *testing.T) {
			box, err := NewSecretBox([]byte(strings.Repeat("k", keyLen)))
			if err != nil {
				t.Fatalf("NewSecretBox: %v", err)
			}

			const secret = "test-secret-not-a-real-credential"
			sealed, err := box.Encrypt(secret)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}
			if sealed == secret || strings.Contains(sealed, secret) {
				t.Fatal("ciphertext contains the plaintext")
			}

			opened, err := box.Decrypt(sealed)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if opened != secret {
				t.Errorf("round trip = %q, want %q", opened, secret)
			}
		})
	}
}

// GCM is nonce-based, so the same plaintext must not seal to the same
// ciphertext — otherwise identical keys across tenants would be detectable
// from the stored rows.
func TestSecretBoxCiphertextIsNotDeterministic(t *testing.T) {
	box, err := NewSecretBox([]byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatalf("NewSecretBox: %v", err)
	}

	a, _ := box.Encrypt("same-secret")
	b, _ := box.Encrypt("same-secret")

	if a == b {
		t.Error("the same plaintext sealed to identical ciphertext; the nonce is not random")
	}
}

// Empty must stay empty so "no credential stored" is distinguishable from
// "a credential that decrypts to an empty string".
func TestSecretBoxEmptyStaysEmpty(t *testing.T) {
	box, _ := NewSecretBox([]byte(strings.Repeat("k", 32)))

	sealed, err := box.Encrypt("")
	if err != nil || sealed != "" {
		t.Errorf("Encrypt(\"\") = %q, %v; want empty", sealed, err)
	}
	opened, err := box.Decrypt("")
	if err != nil || opened != "" {
		t.Errorf("Decrypt(\"\") = %q, %v; want empty", opened, err)
	}
}

func TestSecretBoxRejectsBadKeyLength(t *testing.T) {
	for _, keyLen := range []int{1, 15, 20, 31, 33} {
		if _, err := NewSecretBox([]byte(strings.Repeat("k", keyLen))); err == nil {
			t.Errorf("key length %d was accepted, want an error", keyLen)
		}
	}
}

// A tampered or wrong-key ciphertext must fail rather than return garbage —
// this is what makes a rotated key a detectable error instead of a silent
// bad password sent to the vendor.
func TestSecretBoxRejectsTamperedCiphertext(t *testing.T) {
	box, _ := NewSecretBox([]byte(strings.Repeat("k", 32)))
	sealed, _ := box.Encrypt("secret")

	// Flip a character in the middle of the base64.
	runes := []rune(sealed)
	mid := len(runes) / 2
	if runes[mid] == 'A' {
		runes[mid] = 'B'
	} else {
		runes[mid] = 'A'
	}

	if _, err := box.Decrypt(string(runes)); err == nil {
		t.Error("tampered ciphertext decrypted without error")
	}
}

func TestSecretBoxRejectsWrongKey(t *testing.T) {
	sealer, _ := NewSecretBox([]byte(strings.Repeat("a", 32)))
	opener, _ := NewSecretBox([]byte(strings.Repeat("b", 32)))

	sealed, _ := sealer.Encrypt("secret")

	if _, err := opener.Decrypt(sealed); err == nil {
		t.Error("a different key decrypted the ciphertext")
	}
}

// No key is a development convenience only, and it must announce itself so
// startup can warn.
func TestSecretBoxWithoutKeyIsOnlyEncoding(t *testing.T) {
	box, err := NewSecretBox(nil)
	if err != nil {
		t.Fatalf("NewSecretBox(nil): %v", err)
	}
	if !box.NoEncryption() {
		t.Error("NoEncryption() = false with no key; startup could not warn")
	}

	sealed, _ := box.Encrypt("secret")
	if sealed == "secret" {
		t.Error("value was stored verbatim; expected at least base64")
	}
	opened, err := box.Decrypt(sealed)
	if err != nil || opened != "secret" {
		t.Errorf("round trip = %q, %v; want the original", opened, err)
	}
}

func TestSecretBoxRejectsNonBase64(t *testing.T) {
	box, _ := NewSecretBox([]byte(strings.Repeat("k", 32)))

	if _, err := box.Decrypt("not!valid!base64!"); err == nil {
		t.Error("non-base64 input decrypted without error")
	}
}

func TestSecretBoxRejectsShortCiphertext(t *testing.T) {
	box, _ := NewSecretBox([]byte(strings.Repeat("k", 32)))

	// Valid base64 but far shorter than a GCM nonce.
	if _, err := box.Decrypt("AAAA"); err == nil {
		t.Error("a ciphertext shorter than the nonce decrypted without error")
	}
}
