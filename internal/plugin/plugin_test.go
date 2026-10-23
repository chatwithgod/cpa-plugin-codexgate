package plugin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

func setup(t *testing.T) (*Manager, []byte, []byte, string) {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	claims := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, time.Now().Unix()+3600)))
	token := header + "." + claims + ".c3ludGhldGlj"
	path := filepath.Join(t.TempDir(), "auth.json")
	data, _ := json.Marshal(map[string]any{"auth_mode": "chatgpt", "tokens": map[string]string{"access_token": token}})
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	quoted, _ := json.Marshal(path)
	config, _ := json.Marshal(lifecycleRequest{SchemaVersion: 6, ConfigYAML: []byte("enabled: true\nauth-file: " + string(quoted))})
	request, _ := json.Marshal(map[string]any{"Method": "GET", "Path": "/v1/models", "Headers": map[string][]string{"Authorization": {"Bearer " + token}}, "Body": "ignored-non-base64-content"})
	m := &Manager{}
	if _, err := m.Handle(pluginabi.MethodPluginRegister, config); err != nil {
		t.Fatal(err)
	}
	return m, config, request, token
}

func TestNonExclusiveLifecycle(t *testing.T) {
	m, config, request, _ := setup(t)
	reg, _ := json.Marshal(registration())
	var metadata struct {
		Metadata pluginapi.Metadata `json:"metadata"`
	}
	if err := json.Unmarshal(reg, &metadata); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{metadata.Metadata.Name, metadata.Metadata.Version, metadata.Metadata.Author, metadata.Metadata.GitHubRepository} {
		if strings.TrimSpace(value) == "" {
			t.Fatal("CPA requires complete registration metadata")
		}
	}
	if !strings.Contains(string(reg), `"frontend_auth_provider":true`) || !strings.Contains(string(reg), `"frontend_auth_provider_exclusive":false`) {
		t.Fatal("must preserve native API-key auth")
	}
	r, err := m.Handle(pluginabi.MethodFrontendAuthAuthenticate, request)
	if err != nil || !r.(pluginapi.FrontendAuthResponse).Authenticated {
		t.Fatal("authentication failed")
	}
	m.Close()
	r, _ = m.Handle(pluginabi.MethodFrontendAuthAuthenticate, request)
	if r.(pluginapi.FrontendAuthResponse).Authenticated {
		t.Fatal("closed plugin accepted credentials")
	}
	if _, err := m.Handle(pluginabi.MethodPluginReconfigure, config); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Handle(pluginabi.MethodPluginReconfigure, []byte(`{"schema_version":1}`)); err == nil {
		t.Fatal("old contract accepted")
	}
	r, _ = m.Handle(pluginabi.MethodFrontendAuthAuthenticate, request)
	if r.(pluginapi.FrontendAuthResponse).Authenticated {
		t.Fatal("invalid reconfiguration did not close auth")
	}
}

func TestManagementRedaction(t *testing.T) {
	m, _, _, token := setup(t)
	_, err := m.Handle(pluginabi.MethodManagementRegister, []byte(`{"BasePath":"/v0/management"}`))
	if err != nil {
		t.Fatal(err)
	}
	r, err := m.Handle(pluginabi.MethodManagementHandle, []byte(`{"Method":"GET","Path":"/v0/management/codexgate/status"}`))
	if err != nil {
		t.Fatal(err)
	}
	response := r.(pluginapi.ManagementResponse)
	if response.StatusCode != 200 || strings.Contains(string(response.Body), token) || !strings.Contains(string(response.Body), `"ready":true`) {
		t.Fatal("unsafe or unavailable status")
	}
	r, _ = m.Handle(pluginabi.MethodFrontendAuthAuthenticate, []byte("{"))
	if r.(pluginapi.FrontendAuthResponse).Authenticated {
		t.Fatal("malformed RPC authenticated")
	}
}

func TestConcurrentLifecycle(t *testing.T) {
	m, config, request, _ := setup(t)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, _ = m.Handle(pluginabi.MethodFrontendAuthAuthenticate, request)
				_ = m.status()
			}
		}()
	}
	for i := 0; i < 10; i++ {
		m.Close()
		if _, err := m.Handle(pluginabi.MethodPluginReconfigure, config); err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
	m.Close()
}
