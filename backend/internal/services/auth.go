package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type AuthService struct {
	enabled  bool
	username string
	password string
	secret   string
	ttl      time.Duration
}

type AuthStatus struct {
	Enabled       bool   `json:"enabled"`
	Username      string `json:"username,omitempty"`
	Authenticated bool   `json:"authenticated"`
}

func NewAuthService(enabled bool, username string, password string, secret string, ttl time.Duration) *AuthService {
	if username == "" {
		username = "admin"
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &AuthService{
		enabled:  enabled,
		username: username,
		password: password,
		secret:   secret,
		ttl:      ttl,
	}
}

func NewAuthServiceFromEnv() *AuthService {
	enabled := strings.EqualFold(envString("AUTH_ENABLED", "false"), "true")
	ttlHours := envInt("AUTH_TOKEN_TTL_HOURS", 24)
	return NewAuthService(
		enabled,
		envString("ADMIN_USERNAME", "admin"),
		os.Getenv("ADMIN_PASSWORD"),
		envString("AUTH_SESSION_SECRET", os.Getenv("ADMIN_PASSWORD")),
		time.Duration(ttlHours)*time.Hour,
	)
}

func (s *AuthService) Enabled() bool {
	return s != nil && s.enabled
}

func (s *AuthService) Status(token string) AuthStatus {
	status := AuthStatus{Enabled: s.Enabled()}
	if s == nil {
		return status
	}
	status.Username = s.username
	status.Authenticated = s.Verify(token)
	return status
}

func (s *AuthService) Login(username string, password string) (string, int64, error) {
	if !s.Enabled() {
		return "", 0, errors.New("auth is disabled")
	}
	if s.password == "" || s.secret == "" {
		return "", 0, errors.New("ADMIN_PASSWORD and AUTH_SESSION_SECRET are required")
	}
	if subtle.ConstantTimeCompare([]byte(username), []byte(s.username)) != 1 ||
		subtle.ConstantTimeCompare([]byte(password), []byte(s.password)) != 1 {
		return "", 0, errors.New("invalid username or password")
	}
	expires := time.Now().Add(s.ttl).Unix()
	payload := fmt.Sprintf("%s:%d", s.username, expires)
	tokenPayload := base64.RawURLEncoding.EncodeToString([]byte(payload))
	signature := s.sign(tokenPayload)
	return tokenPayload + "." + signature, expires, nil
}

func (s *AuthService) Verify(token string) bool {
	if !s.Enabled() {
		return true
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	expected := s.sign(parts[0])
	if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expected)) != 1 {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	payloadParts := strings.Split(string(decoded), ":")
	if len(payloadParts) != 2 || payloadParts[0] != s.username {
		return false
	}
	expires, err := strconv.ParseInt(payloadParts[1], 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Unix() < expires
}

func (s *AuthService) sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
