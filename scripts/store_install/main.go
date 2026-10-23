// Runs CPA's actual release installer against local release fixtures, never a
// production config or a network endpoint. The installed library is then used
// by smoke_cpa.py in a real, isolated official CPA process.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginstore"
)

type fixtures map[string][]byte

func (f fixtures) Do(req *http.Request) (*http.Response, error) {
	body, ok := f[req.URL.String()]
	if !ok {
		return nil, fmt.Errorf("unexpected fixture request (external network disabled): %s", req.URL.Path)
	}
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body)), Request: req}, nil
}

func main() {
	archive := flag.String("archive", "", "platform release ZIP")
	checksums := flag.String("checksums", "", "checksums.txt")
	version := flag.String("version", "", "release version without v")
	root := flag.String("root", "", "isolated run parent")
	flag.Parse()
	if err := run(*archive, *checksums, *version, *root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(archive, checksums, version, root string) error {
	if archive == "" || checksums == "" || version == "" || root == "" {
		return fmt.Errorf("archive, checksums, version and root are required")
	}
	archiveData, err := os.ReadFile(archive)
	if err != nil {
		return err
	}
	checksumData, err := os.ReadFile(checksums)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	runDir, err := os.MkdirTemp(root, "store-install-")
	if err != nil {
		return err
	}
	const repository = "https://github.com/chatwithgod/cpa-plugin-codexgate"
	apiURL := "https://api.github.com/repos/chatwithgod/cpa-plugin-codexgate/releases/tags/v" + version
	archiveURL := repository + "/releases/download/v" + version + "/" + filepath.Base(archive)
	checksumsURL := repository + "/releases/download/v" + version + "/checksums.txt"
	releaseData, _ := json.Marshal(pluginstore.Release{TagName: "v" + version, Assets: []pluginstore.ReleaseAsset{
		{Name: filepath.Base(archive), BrowserDownloadURL: archiveURL},
		{Name: "checksums.txt", BrowserDownloadURL: checksumsURL},
	}})
	files := fixtures{apiURL: releaseData, archiveURL: archiveData, checksumsURL: checksumData}
	plugin := pluginstore.Plugin{ID: "codexgate", Name: "CodexGate", Description: "Local Codex login compatibility", Author: "chatwithgod", Repository: repository}
	client := pluginstore.NewClient(files, "")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bad := append([]byte(nil), checksumData...)
	if len(bad) == 0 {
		return fmt.Errorf("empty checksums")
	}
	if bad[0] == '0' {
		bad[0] = '1'
	} else {
		bad[0] = '0'
	}
	files[checksumsURL] = bad
	_, err = client.InstallVersion(ctx, plugin, "v"+version, version, pluginstore.InstallOptions{PluginsDir: filepath.Join(runDir, "invalid"), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "checksum") {
		return fmt.Errorf("installer must reject a mismatched checksum: %v", err)
	}
	files[checksumsURL] = checksumData
	result, err := client.InstallVersion(ctx, plugin, "v"+version, version, pluginstore.InstallOptions{PluginsDir: filepath.Join(runDir, "valid"), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"success": true, "installer": "CLIProxyAPI v8.0.38 release installer", "network": "local fixtures only",
		"checksum_mismatch_rejected": true, "path": result.Path, "version": result.Version, "overwritten": result.Overwritten,
	})
}
