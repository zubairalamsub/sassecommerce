package repository

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// SecretBox encrypts provider credentials at rest with AES-GCM. It mirrors the
// scheme user-service already uses for 2FA secrets — nonce prefixed to the
// ciphertext, the whole thing base64'd — so operators have one key format to
// manage rather than two.
//
// With no key it degrades to base64 encoding, which is obfuscation and not
// encryption. That is acceptable for local development only, and callers are
// expected to log loudly at startup; NoEncryption reports the state so they
// can.
type SecretBox struct {
	gcm   cipher.AEAD
	noEnc bool
}

// NewSecretBox builds a box from a 16, 24 or 32 byte key. An empty key yields
// a passthrough box.
func NewSecretBox(key []byte) (*SecretBox, error) {
	if len(key) == 0 {
		return &SecretBox{noEnc: true}, nil
	}
	if l := len(key); l != 16 && l != 24 && l != 32 {
		return nil, errors.New("encryption key must be 16, 24, or 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &SecretBox{gcm: gcm}, nil
}

// NoEncryption reports whether this box is only base64 encoding.
func (b *SecretBox) NoEncryption() bool { return b.noEnc }

// Encrypt seals a plaintext secret. An empty secret stays empty so "no
// credential stored" is distinguishable from "a credential that decrypts to
// an empty string".
func (b *SecretBox) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if b.noEnc {
		return base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
	}
	nonce := make([]byte, b.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b.gcm.Seal(nonce, nonce, []byte(plaintext), nil)), nil
}

// Decrypt opens a stored secret.
func (b *SecretBox) Decrypt(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return "", err
	}
	if b.noEnc {
		return string(raw), nil
	}
	ns := b.gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	plain, err := b.gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
