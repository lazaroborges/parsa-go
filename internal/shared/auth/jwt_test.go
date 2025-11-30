package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestJWT_GenerateAndValidate(t *testing.T) {
	secret := "my-secret-key"
	j := NewJWT(secret)

	userID := int64(123)
	email := "test@example.com"

	// 1. Test Generate
	token, err := j.Generate(userID, email)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}
	if token == "" {
		t.Fatal("Generate() returned empty token")
	}

	// 2. Test Validate Success
	claims, err := j.Validate(token)
	if err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("Validate() got UserID %d, want %d", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Errorf("Validate() got Email %s, want %s", claims.Email, email)
	}

	// 3. Test Tampered Token (Wrong Signature)
	parts := strings.Split(token, ".")
	tamperedSignature := "invalid-signature"
	tamperedToken := parts[0] + "." + parts[1] + "." + tamperedSignature
	_, err = j.Validate(tamperedToken)
	if err == nil {
		t.Error("Validate() accepted tampered signature")
	} else if err.Error() != "invalid signature" {
		t.Errorf("Validate() returned wrong error for tampered signature: %v", err)
	}

	// 4. Test Invalid Format
	_, err = j.Validate("invalid.token")
	if err == nil {
		t.Error("Validate() accepted invalid format")
	}
}

func TestJWT_ExpiredToken(t *testing.T) {
	secret := "my-secret-key"
	j := NewJWT(secret)

	// Manually create an expired token
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	claims := JWTClaims{
		UserID: 1,
		Email:  "expired@example.com",
		Iat:    time.Now().Add(-25 * time.Hour).Unix(),
		Exp:    time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	message := headerB64 + "." + claimsB64
	signature := j.sign(message) // Use internal sign method via public API if I could, but I can't access 'sign' directly as it is private?
	// Wait, 'sign' is private (lowercase 's').
	// But I am in package 'auth', so I can access it!
	
	token := message + "." + signature

	_, err := j.Validate(token)
	if err == nil {
		t.Error("Validate() accepted expired token")
	} else if err.Error() != "token expired" {
		t.Errorf("Validate() returned wrong error for expired token: %v", err)
	}
}
