package crypto

import (
	"strings"
	"testing"
)

const testKey = "01234567890123456789012345678901" // 32 bytes for AES-256

func TestNewEncryptor_ValidKey(t *testing.T) {
	enc, err := NewEncryptor(testKey)
	if err != nil {
		t.Fatalf("NewEncryptor() failed: %v", err)
	}
	if enc == nil {
		t.Fatal("NewEncryptor() returned nil")
	}
}

func TestNewEncryptor_InvalidKeyLength(t *testing.T) {
	_, err := NewEncryptor("too-short")
	if err == nil {
		t.Error("NewEncryptor() expected error for short key, got nil")
	}
	if err != ErrInvalidKey {
		t.Errorf("NewEncryptor() error = %v, want %v", err, ErrInvalidKey)
	}
}

func TestNewEncryptor_EmptyKey(t *testing.T) {
	_, err := NewEncryptor("")
	if err == nil {
		t.Error("NewEncryptor() expected error for empty key, got nil")
	}
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	enc, _ := NewEncryptor(testKey)

	plaintext := "sensitive financial data"
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	if ciphertext == plaintext {
		t.Error("Encrypt() returned plaintext")
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypt() = %q, want %q", decrypted, plaintext)
	}
}

func TestEncrypt_EmptyString(t *testing.T) {
	enc, _ := NewEncryptor(testKey)

	ciphertext, err := enc.Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}
	if ciphertext != "" {
		t.Errorf("Encrypt(\"\") = %q, want empty string", ciphertext)
	}
}

func TestDecrypt_EmptyString(t *testing.T) {
	enc, _ := NewEncryptor(testKey)

	plaintext, err := enc.Decrypt("")
	if err != nil {
		t.Fatalf("Decrypt() failed: %v", err)
	}
	if plaintext != "" {
		t.Errorf("Decrypt(\"\") = %q, want empty string", plaintext)
	}
}

func TestEncrypt_DifferentCiphertexts(t *testing.T) {
	enc, _ := NewEncryptor(testKey)

	c1, _ := enc.Encrypt("same text")
	c2, _ := enc.Encrypt("same text")

	if c1 == c2 {
		t.Error("Encrypt() produced identical ciphertexts for same plaintext (nonce should differ)")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	enc, _ := NewEncryptor(testKey)

	ciphertext, _ := enc.Encrypt("secret data")

	// Tamper with the ciphertext
	tampered := ciphertext[:len(ciphertext)-2] + "XX"
	_, err := enc.Decrypt(tampered)
	if err == nil {
		t.Error("Decrypt() accepted tampered ciphertext")
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	enc, _ := NewEncryptor(testKey)

	_, err := enc.Decrypt("not-valid-base64!!!")
	if err == nil {
		t.Error("Decrypt() accepted invalid base64")
	}
}

func TestDecrypt_TooShortCiphertext(t *testing.T) {
	enc, _ := NewEncryptor(testKey)

	// Base64 encode a very short byte slice (shorter than nonce)
	_, err := enc.Decrypt("YQ==") // "a" in base64
	if err == nil {
		t.Error("Decrypt() accepted ciphertext shorter than nonce")
	}
}

func TestEncryptDecrypt_UnicodeContent(t *testing.T) {
	enc, _ := NewEncryptor(testKey)

	plaintext := "Transação financeira: R$ 1.500,00 — café ☕"
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() failed with unicode: %v", err)
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() failed with unicode: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Unicode roundtrip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_LongContent(t *testing.T) {
	enc, _ := NewEncryptor(testKey)

	plaintext := strings.Repeat("long content ", 1000)
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() failed with long content: %v", err)
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() failed with long content: %v", err)
	}

	if decrypted != plaintext {
		t.Error("Long content roundtrip failed")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	enc1, _ := NewEncryptor(testKey)
	enc2, _ := NewEncryptor("98765432109876543210987654321098")

	ciphertext, _ := enc1.Encrypt("encrypted with key1")

	_, err := enc2.Decrypt(ciphertext)
	if err == nil {
		t.Error("Decrypt() succeeded with wrong key")
	}
}
