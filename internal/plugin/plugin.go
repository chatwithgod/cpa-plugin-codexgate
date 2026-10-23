// Package plugin implements the CPA native plugin RPC contract.
package plugin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"codexgate/internal/auth"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"gopkg.in/yaml.v3"
)

var Version = "0.1.0-dev.3"

type Manager struct {
	mu             sync.RWMutex
	enabled        bool
	authFile       string
	managementPath string
}

type config struct {
	Enabled  bool   `yaml:"enabled"`
	AuthFile string `yaml:"auth-file"`
}

type lifecycleRequest struct {
	ConfigYAML    []byte `json:"config_yaml"`
	SchemaVersion uint32 `json:"schema_version"`
}

// Deliberately omit Body: auth must not decode or retain model request content.
type frontendRequest struct {
	Method  string
	Path    string
	Headers http.Header
}

func registration() any {
	return map[string]any{
		"schema_version": pluginabi.SchemaVersion,
		"metadata": pluginapi.Metadata{
			Name: "codexgate", Version: Version, Author: "CodexGate contributors",
			// CPA requires this field even for unpublished local development builds.
			// This is the planned address; creating/publishing the repository is separate.
			GitHubRepository: "https://github.com/chatwithgod/cpa-plugin-codexgate",
			ConfigFields: []pluginapi.ConfigField{{
				Name: "auth-file", Type: pluginapi.ConfigFieldTypeString, Default: "",
				Description: "通常留空。默认读取 CODEX_HOME/auth.json 或当前用户的 .codex/auth.json；只读，不保存 token。非默认位置可填写绝对路径。",
			}},
		},
		"capabilities": map[string]bool{
			"frontend_auth_provider": true, "frontend_auth_provider_exclusive": false,
			"management_api": true,
		},
	}
}

func (m *Manager) configure(raw []byte) error {
	var req lifecycleRequest
	if json.Unmarshal(raw, &req) != nil || req.SchemaVersion < pluginabi.SchemaVersion {
		m.Close()
		return errors.New("CodexGate requires CPA plugin schema 6 or newer")
	}
	cfg := config{Enabled: true}
	if yaml.Unmarshal(req.ConfigYAML, &cfg) != nil {
		m.Close()
		return errors.New("invalid CodexGate configuration")
	}
	path, err := auth.ResolveFile(cfg.AuthFile)
	if err != nil {
		m.Close()
		return errors.New("auth-file and CODEX_HOME must resolve to an absolute path")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.authFile, m.enabled = path, cfg.Enabled
	return nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
}

func (m *Manager) authenticate(raw []byte) pluginapi.FrontendAuthResponse {
	var req frontendRequest
	if json.Unmarshal(raw, &req) != nil {
		return pluginapi.FrontendAuthResponse{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.enabled || !auth.Authenticate(m.authFile, req.Method, req.Path, req.Headers, time.Now()) {
		return pluginapi.FrontendAuthResponse{}
	}
	return pluginapi.FrontendAuthResponse{
		Authenticated: true,
		Principal:     "codexgate-local",
		Metadata:      map[string]string{"auth_kind": "local_codex_login"},
	}
}

func (m *Manager) status() any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state := auth.State{Reason: "disabled"}
	if m.enabled {
		state = auth.Inspect(m.authFile, time.Now())
	}
	return struct {
		Version   string     `json:"version"`
		Enabled   bool       `json:"enabled"`
		Exclusive bool       `json:"exclusive"`
		AuthFile  string     `json:"auth_file"`
		Login     auth.State `json:"login"`
	}{Version: Version, Enabled: m.enabled, AuthFile: m.authFile, Login: state}
}

func jsonResponse(code int, value any) pluginapi.ManagementResponse {
	raw, _ := json.Marshal(value)
	return pluginapi.ManagementResponse{
		StatusCode: code,
		Headers: http.Header{"Content-Type": {"application/json; charset=utf-8"},
			"Cache-Control": {"no-store"}, "X-Content-Type-Options": {"nosniff"}},
		Body: raw,
	}
}

// Handle adds no listener, model executor, request rewriting, or background job.
func (m *Manager) Handle(method string, raw []byte) (any, error) {
	switch method {
	case pluginabi.MethodPluginRegister, pluginabi.MethodPluginReconfigure:
		if err := m.configure(raw); err != nil {
			return nil, err
		}
		return registration(), nil
	case pluginabi.MethodPluginQuiesce, pluginabi.MethodPluginShutdown:
		m.Close()
		return map[string]bool{"closed": true}, nil
	case pluginabi.MethodFrontendAuthIdentifier:
		return map[string]string{"identifier": "local-codex-login"}, nil
	case pluginabi.MethodFrontendAuthAuthenticate:
		return m.authenticate(raw), nil
	case pluginabi.MethodManagementRegister:
		var req pluginapi.ManagementRegistrationRequest
		if json.Unmarshal(raw, &req) != nil || strings.TrimRight(req.BasePath, "/") != "/v0/management" {
			return nil, errors.New("invalid management registration")
		}
		path := strings.TrimRight(req.BasePath, "/") + "/codexgate/status"
		m.mu.Lock()
		m.managementPath = path
		m.mu.Unlock()
		return map[string]any{"routes": []map[string]string{{"method": "GET", "path": path}}, "resources": []any{}}, nil
	case pluginabi.MethodManagementHandle:
		var req pluginapi.ManagementRequest
		if json.Unmarshal(raw, &req) != nil {
			return jsonResponse(400, map[string]string{"error": "invalid_request"}), nil
		}
		m.mu.RLock()
		path := m.managementPath
		m.mu.RUnlock()
		if req.Method != http.MethodGet || path == "" || req.Path != path {
			return jsonResponse(404, map[string]string{"error": "not_found"}), nil
		}
		return jsonResponse(200, m.status()), nil
	default:
		return nil, errors.New("unsupported CodexGate RPC method")
	}
}
