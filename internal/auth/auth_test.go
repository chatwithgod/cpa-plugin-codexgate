package auth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixtureToken(exp int64, nonce string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{"exp": exp, "sub": nonce})
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString([]byte("synthetic-test-signature"))
}

func writeLogin(t *testing.T, path, mode, token string) {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"auth_mode": mode, "tokens": map[string]string{"access_token": token, "account_id": "synthetic-account"}})
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestAuthentication(t *testing.T) {
	now := time.Unix(1800000000, 0)
	valid := fixtureToken(now.Unix()+3600, "current")
	expired := fixtureToken(now.Unix(), "expired")
	other := fixtureToken(now.Unix()+3600, "other")
	for _, tc := range []struct {
		name, mode, local, incoming, method, path, account string
		want                                               bool
	}{
		{"models", "chatgpt", valid, valid, "GET", "/v1/models", "", true},
		{"responses", "chatgpt", valid, valid, "POST", "/v1/responses", "synthetic-account", true},
		{"websocket", "chatgpt", valid, valid, "GET", "/v1/responses", "", true},
		{"compact", "chatgpt", valid, valid, "POST", "/backend-api/codex/responses/compact", "", true},
		{"no credential", "chatgpt", valid, "", "GET", "/v1/models", "", false},
		{"other valid-looking JWT", "chatgpt", valid, other, "GET", "/v1/models", "", false},
		{"API key left to host", "chatgpt", valid, "synthetic-cpa-key", "GET", "/v1/models", "", false},
		{"expired", "chatgpt", expired, expired, "GET", "/v1/models", "", false},
		{"wrong auth mode", "apikey", valid, valid, "GET", "/v1/models", "", false},
		{"missing login mode", "", valid, valid, "GET", "/v1/models", "", false},
		{"wrong account", "chatgpt", valid, valid, "GET", "/v1/models", "another-account", false},
		{"management denied", "chatgpt", valid, valid, "GET", "/v0/management/config", "", false},
		{"unrelated route denied", "chatgpt", valid, valid, "POST", "/v1/chat/completions", "", false},
		{"wrong method", "chatgpt", valid, valid, "DELETE", "/v1/responses", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "auth.json")
			writeLogin(t, path, tc.mode, tc.local)
			h := http.Header{"Authorization": {"Bearer " + tc.incoming}}
			if tc.account != "" {
				h.Set("ChatGPT-Account-ID", tc.account)
			}
			if got := Authenticate(path, tc.method, tc.path, h, now); got != tc.want {
				t.Fatalf("authenticated=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestRotationLogoutAndMalformedFile(t *testing.T) {
	now := time.Unix(1800000000, 0)
	a, b := fixtureToken(now.Unix()+3600, "one"), fixtureToken(now.Unix()+3600, "two")
	path := filepath.Join(t.TempDir(), "auth.json")
	check := func(token string) bool {
		return Authenticate(path, "GET", "/v1/models", http.Header{"Authorization": {"Bearer " + token}}, now)
	}
	writeLogin(t, path, "chatgpt", a)
	if !check(a) {
		t.Fatal("first credential rejected")
	}
	writeLogin(t, path, "chatgpt", b)
	if check(a) || !check(b) {
		t.Fatal("rotation did not take effect immediately")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if check(b) {
		t.Fatal("credential survived logout")
	}
	for _, raw := range []string{"{", strings.Repeat("x", maxAuthFileBytes+1), `{"auth_mode":"chatgpt","tokens":{}}`} {
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if check(b) {
			t.Fatal("invalid file authenticated")
		}
	}
}

func TestClaimsAndHeaders(t *testing.T) {
	now := time.Unix(1800000000, 0)
	token := fixtureToken(now.Unix()+3600, "current")
	path := filepath.Join(t.TempDir(), "auth.json")
	writeLogin(t, path, "chatgpt", token)
	for _, h := range []http.Header{
		{"Authorization": {"Basic " + token}},
		{"Authorization": {"Bearer " + token, "Bearer " + token}},
		{"Authorization": {"Bearer " + token}, "authorization": {"Bearer " + token}},
		{"Authorization": {"Bearer " + token}, "ChatGPT-Account-ID": {"synthetic-account", "synthetic-account"}},
	} {
		if Authenticate(path, "GET", "/v1/models", h, now) {
			t.Fatal("ambiguous headers accepted")
		}
	}
	for _, claims := range []string{`{}`, `{"exp":0}`, `{"exp":"2000000000"}`, `{"exp":2000000000,"nbf":1900000000}`} {
		parts := strings.Split(token, ".")
		parts[1] = base64.RawURLEncoding.EncodeToString([]byte(claims))
		bad := strings.Join(parts, ".")
		writeLogin(t, path, "chatgpt", bad)
		if Authenticate(path, "GET", "/v1/models", http.Header{"Authorization": {"Bearer " + bad}}, now) {
			t.Fatal("invalid time claims accepted")
		}
	}
}

func TestDefaultPathAndDiagnostics(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	path, err := ResolveFile("")
	if err != nil || path != filepath.Join(home, "auth.json") {
		t.Fatal("CODEX_HOME was not used")
	}
	if _, err := ResolveFile("relative/auth.json"); err == nil {
		t.Fatal("relative auth file accepted")
	}
	now := time.Unix(1800000000, 0)
	token := fixtureToken(now.Unix()+3600, "private-subject")
	writeLogin(t, path, "chatgpt", token)
	state := Inspect(path, now)
	raw, _ := json.Marshal(state)
	if !state.Ready || strings.Contains(string(raw), token) || strings.Contains(string(raw), "synthetic-account") {
		t.Fatal("diagnostics leaked identity or were not ready")
	}
}
