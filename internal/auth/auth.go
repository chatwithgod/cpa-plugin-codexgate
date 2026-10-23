// Package auth accepts only the current, unexpired token in a trusted local
// Codex login file. It is not a general JWT signature-verification service.
package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxAuthFileBytes = 64 << 10
	maxTokenBytes    = 16 << 10
)

// State contains diagnostics only; never add a token or account identifier.
type State struct {
	Ready     bool   `json:"ready"`
	Reason    string `json:"reason"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

type login struct {
	Mode   string `json:"auth_mode"`
	Tokens struct {
		AccessToken string `json:"access_token"`
		AccountID   string `json:"account_id"`
	} `json:"tokens"`
}

// ResolveFile does not create, refresh, or modify any Codex files.
func ResolveFile(explicit string) (string, error) {
	path := strings.TrimSpace(explicit)
	if path == "" {
		home := strings.TrimSpace(os.Getenv("CODEX_HOME"))
		if home == "" {
			userHome, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			home = filepath.Join(userHome, ".codex")
		}
		path = filepath.Join(home, "auth.json")
	}
	if !filepath.IsAbs(path) {
		return "", &os.PathError{Op: "resolve", Path: "auth-file", Err: os.ErrInvalid}
	}
	return filepath.Clean(path), nil
}

// AllowedRoute intentionally excludes management, resource, and unrelated APIs.
// Responses HTTP/SSE, WebSocket handshakes, and the Images API use these paths.
func AllowedRoute(method, path string) bool {
	if method == http.MethodGet && path == "/v1/models" {
		return true
	}
	if method == http.MethodGet && strings.HasPrefix(path, "/v1/models/") {
		return len(strings.TrimPrefix(path, "/v1/models/")) > 0
	}
	if path == "/v1/images/generations" || path == "/v1/images/edits" {
		return method == http.MethodPost
	}
	for _, prefix := range []string{"/v1", "/backend-api/codex"} {
		switch path {
		case prefix + "/responses":
			return method == http.MethodGet || method == http.MethodPost
		case prefix + "/responses/compact", prefix + "/alpha/search":
			return method == http.MethodPost
		}
	}
	return false
}

func singleHeader(headers http.Header, name string) (string, bool) {
	var values []string
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			values = append(values, value...)
		}
	}
	if len(values) != 1 {
		return "", false
	}
	return values[0], true
}

func bearer(headers http.Header) string {
	value, ok := singleHeader(headers, "Authorization")
	if !ok || len(value) > maxTokenBytes+32 {
		return ""
	}
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || len(parts[1]) > maxTokenBytes {
		return ""
	}
	return parts[1]
}

func readLogin(path string, now time.Time) (login, State) {
	var value login
	f, err := os.Open(path)
	if err != nil {
		return value, State{Reason: "login_file_unavailable"}
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxAuthFileBytes {
		return value, State{Reason: "invalid_login_file"}
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxAuthFileBytes+1))
	if err != nil || len(raw) > maxAuthFileBytes || json.Unmarshal(raw, &value) != nil {
		return login{}, State{Reason: "invalid_login_file"}
	}
	if value.Mode != "chatgpt" {
		return login{}, State{Reason: "not_chatgpt_login"}
	}
	if value.Tokens.AccessToken == "" || len(value.Tokens.AccessToken) > maxTokenBytes {
		return login{}, State{Reason: "missing_or_invalid_access_token"}
	}
	state := tokenState(value.Tokens.AccessToken, now)
	if !state.Ready {
		return login{}, state
	}
	return value, state
}

// Claims are read only from the trusted login-file token. Exact token matching
// below is the trust check; merely decoding a client's JWT never grants access.
func tokenState(token string, now time.Time) State {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[2] == "" {
		return State{Reason: "invalid_access_token"}
	}
	header, err := base64.RawURLEncoding.Strict().DecodeString(parts[0])
	if err != nil {
		return State{Reason: "invalid_access_token"}
	}
	var h struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal(header, &h) != nil || h.Alg == "" || strings.EqualFold(h.Alg, "none") {
		return State{Reason: "invalid_access_token"}
	}
	payload, err := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	if err != nil {
		return State{Reason: "invalid_access_token"}
	}
	var claims struct {
		Exp int64 `json:"exp"`
		Nbf int64 `json:"nbf"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Exp <= 0 {
		return State{Reason: "invalid_access_token"}
	}
	if claims.Exp <= now.Unix() {
		return State{Reason: "expired_access_token", ExpiresAt: claims.Exp}
	}
	if claims.Nbf > now.Unix() {
		return State{Reason: "access_token_not_yet_valid", ExpiresAt: claims.Exp}
	}
	return State{Ready: true, Reason: "ready", ExpiresAt: claims.Exp}
}

// Authenticate re-reads the small file each time. There is no background watcher,
// stale token allowlist, persisted token copy, or network refresh request.
func Authenticate(path, method, requestPath string, headers http.Header, now time.Time) bool {
	if !AllowedRoute(method, requestPath) {
		return false
	}
	token := bearer(headers)
	if token == "" || strings.Count(token, ".") != 2 {
		return false
	}
	value, state := readLogin(path, now)
	if !state.Ready {
		return false
	}
	want, got := sha256.Sum256([]byte(value.Tokens.AccessToken)), sha256.Sum256([]byte(token))
	if subtle.ConstantTimeCompare(want[:], got[:]) != 1 {
		return false
	}
	// The account header is optional, but a provided header must match this login.
	for name := range headers {
		if strings.EqualFold(name, "ChatGPT-Account-ID") {
			account, ok := singleHeader(headers, "ChatGPT-Account-ID")
			if !ok || value.Tokens.AccountID == "" || account != value.Tokens.AccountID {
				return false
			}
			break
		}
	}
	return true
}

func Inspect(path string, now time.Time) State {
	_, state := readLogin(path, now)
	return state
}
