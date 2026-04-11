package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"empty", []byte("")},
		{"short", []byte("secret")},
		{"long", []byte("this is a longer secret with special chars: !@#$%^&*()")},
		{"json", []byte(`{"access_token": "APP_USR-123", "webhook_secret": "abc123"}`)},
	}

	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := Encrypt(tt.plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			if encrypted == "" {
				t.Fatal("Encrypt returned empty string")
			}

			decrypted, err := Decrypt(encrypted, key)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("round-trip failed: got %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptDifferentNonces(t *testing.T) {
	plaintext := []byte("secret")
	key := []byte("0123456789abcdef0123456789abcdef")

	// Encrypt the same plaintext twice
	encrypted1, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("first encrypt failed: %v", err)
	}

	encrypted2, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("second encrypt failed: %v", err)
	}

	// With random nonces, outputs should be different
	if encrypted1 == encrypted2 {
		t.Error("two encryptions of same plaintext should differ (nonces should be random)")
	}

	// But both should decrypt to the same plaintext
	decrypted1, err := Decrypt(encrypted1, key)
	if err != nil {
		t.Fatalf("decrypt1 failed: %v", err)
	}

	decrypted2, err := Decrypt(encrypted2, key)
	if err != nil {
		t.Fatalf("decrypt2 failed: %v", err)
	}

	if !bytes.Equal(decrypted1, plaintext) || !bytes.Equal(decrypted2, plaintext) {
		t.Error("both decryptions should yield original plaintext")
	}
}

func TestEncryptInvalidKeyLength(t *testing.T) {
	tests := []struct {
		name       string
		keyLength  int
		shouldFail bool
	}{
		{"too short", 16, true},
		{"correct", 32, false},
		{"too long", 64, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLength)
			_, err := Encrypt([]byte("plaintext"), key)

			if tt.shouldFail && err == nil {
				t.Error("expected error for invalid key length, got nil")
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("expected success, got error: %v", err)
			}
		})
	}
}

func TestDecryptInvalidKeyLength(t *testing.T) {
	// First encrypt with correct key
	key := []byte("0123456789abcdef0123456789abcdef")
	encrypted, err := Encrypt([]byte("secret"), key)
	if err != nil {
		t.Fatalf("setup encrypt failed: %v", err)
	}

	// Try to decrypt with wrong key length
	wrongKey := make([]byte, 16)
	_, err = Decrypt(encrypted, wrongKey)
	if err == nil {
		t.Error("expected error with invalid key length, got nil")
	}
}

func TestDecryptCorruptedCiphertext(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"invalid base64", "!!!invalid!!!"},
		{"empty string", ""},
		{"too short", "AA=="}, // too short after base64 decode
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.ciphertext, key)
			if err == nil {
				t.Error("expected error for corrupted ciphertext, got nil")
			}
		})
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := []byte("0123456789abcdef0123456789abcdef")
	key2 := []byte("fedcba9876543210fedcba9876543210")

	plaintext := []byte("secret message")

	// Encrypt with key1
	encrypted, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	// Try to decrypt with key2 (wrong key)
	_, err = Decrypt(encrypted, key2)
	if err == nil {
		t.Error("expected error when decrypting with wrong key, got nil")
	}
}
