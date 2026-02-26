package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	password := "my-secure-password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() failed: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword() returned empty hash")
	}
	if hash == password {
		t.Fatal("HashPassword() returned plaintext password")
	}

	// Verify it's a valid bcrypt hash
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		t.Errorf("HashPassword() produced invalid bcrypt hash: %v", err)
	}
}

func TestHashPassword_DifferentHashesForSamePassword(t *testing.T) {
	password := "same-password"
	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)

	if hash1 == hash2 {
		t.Error("HashPassword() produced identical hashes for same password (no salt)")
	}
}

func TestVerifyPassword_Success(t *testing.T) {
	password := "correct-password"
	hash, _ := HashPassword(password)

	err := VerifyPassword(hash, password)
	if err != nil {
		t.Errorf("VerifyPassword() failed with correct password: %v", err)
	}
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	hash, _ := HashPassword("correct-password")

	err := VerifyPassword(hash, "wrong-password")
	if err == nil {
		t.Error("VerifyPassword() accepted wrong password")
	}
}

func TestVerifyPassword_EmptyPassword(t *testing.T) {
	hash, _ := HashPassword("some-password")

	err := VerifyPassword(hash, "")
	if err == nil {
		t.Error("VerifyPassword() accepted empty password against non-empty hash")
	}
}

func TestHashPassword_EmptyPassword(t *testing.T) {
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword() failed with empty password: %v", err)
	}

	err = VerifyPassword(hash, "")
	if err != nil {
		t.Errorf("VerifyPassword() failed for empty password roundtrip: %v", err)
	}
}
