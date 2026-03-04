package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

var (
	ErrAuthCodeNotFound = errors.New("auth code not found or expired")
	ErrAuthCodeExpired  = errors.New("auth code expired")
)

type AuthCode struct {
	Token     string
	UserID    int64
	CreatedAt time.Time
}

type AuthCodeStore struct {
	mu    sync.Mutex
	codes map[string]*AuthCode
	ttl   time.Duration
	stop  chan struct{}
}

func NewAuthCodeStore(ttl time.Duration) *AuthCodeStore {
	s := &AuthCodeStore{
		codes: make(map[string]*AuthCode),
		ttl:   ttl,
		stop:  make(chan struct{}),
	}
	go s.cleanup()
	return s
}

func (s *AuthCodeStore) Generate(token string, userID int64) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := base64.URLEncoding.EncodeToString(b)

	s.mu.Lock()
	s.codes[code] = &AuthCode{
		Token:     token,
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	s.mu.Unlock()

	return code, nil
}

func (s *AuthCodeStore) Exchange(code string) (string, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ac, ok := s.codes[code]
	if !ok {
		return "", 0, ErrAuthCodeNotFound
	}

	delete(s.codes, code)

	if time.Since(ac.CreatedAt) > s.ttl {
		return "", 0, ErrAuthCodeExpired
	}

	return ac.Token, ac.UserID, nil
}

func (s *AuthCodeStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for code, ac := range s.codes {
				if now.Sub(ac.CreatedAt) > s.ttl {
					delete(s.codes, code)
				}
			}
			s.mu.Unlock()
		case <-s.stop:
			return
		}
	}
}

func (s *AuthCodeStore) Stop() {
	close(s.stop)
}
