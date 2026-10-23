# Development and configuration

User guides: [English](../README.md) · [简体中文](../README.zh-CN.md). This document contains manual configuration, authentication details, compatibility requirements, and the build/release workflow. Test results are kept in [VERIFICATION.md](VERIFICATION.md).

## Compatibility baseline

- CPA **8.0.38**, native plugin ABI **1**, metadata schema **6**; Go **1.26.8**.
- Windows amd64; Linux amd64/arm64 with **GLIBC 2.17+**; macOS amd64/arm64.
- Linux CPA `no-plugin` builds and musl-only environments are not supported.
- CodexGate uses non-exclusive frontend authentication. CPA's original API keys continue to work. It does not add a daemon, model routing, request/response rewriting, reasoning continuation, or upstream account management.
- This is an independent MIT-licensed project. It does not change the account's feature entitlements. The dot behavior observed with the native `openai` provider identity is recorded separately in the verification document.

## Plugin configuration

Same-user local installations normally need only the plugin enabled in the management UI. For manual configuration, merge this into CPA's existing YAML configuration; do not replace other plugin entries:

```yaml
plugins:
  enabled: true
  configs:
    codexgate:
      enabled: true
      # auth-file: /absolute/path/to/auth.json
```

Keep at least one existing CPA API key, bind a local CPA instance to `127.0.0.1`, and keep the login directory private. Do not copy a ChatGPT login token into CPA's static API-key list.

### Check login readiness

Use the **CPA management key** to query `GET /v0/management/codexgate/status`. The plugin should report `enabled: true` and `login.ready: true` before switching Codex to the native provider identity. The endpoint does not expose tokens or account IDs and cannot be authenticated with a Codex login token instead of the management key.

## Manual Codex configuration

ECPA users can follow the illustrated toggle instructions in the README. Without ECPA, install and enable CodexGate first, confirm login readiness, then back up and edit **Codex's `config.toml`**, not CPA's YAML configuration.

Set these fields at the **top level** of Codex's TOML file:

```toml
model_provider = "openai"
openai_base_url = "http://127.0.0.1:8317/v1"
```

Use the address and port of your own CPA instance. These are not entries under `[model_providers.openai]`, and you do not need to edit `auth.json`. Wait for active tasks to finish before restarting Codex, then check dot, a new chat, and CPA usage records.

To roll back, restore the previous Codex provider settings (or turn off ECPA's provider-identity toggle), restart Codex when idle, and only then disable CodexGate. Stop CPA before removing or replacing its loaded plugin library.

## Login file and deployment

The login-file path is selected in this order:

1. An explicit absolute `auth-file` path in the plugin settings.
2. The **CPA process's** `CODEX_HOME/auth.json`.
3. The **CPA process user's** `~/.codex/auth.json`.

The resolved path must be absolute. The selected file must be readable and contain a file-based `auth_mode=chatgpt` login. System keychains/keyrings are not read. If the selected file is missing or invalid, login authentication fails closed.

A service account or container does not automatically have access to the desktop user's login. If using a container, read-only-mount the **directory** containing `auth.json` and set `auth-file` to its absolute path inside the container. Do not mount only the single file: Codex may update it by atomic replacement. For a remote CPA instance, the desktop file is not automatically synchronized; providing the correct readable file is a deployment responsibility, not a plugin feature.

## Authentication and supported routes

For each supported request, CodexGate reads the current file and accepts only a Bearer token that **exactly matches** its `access_token` and has valid time claims. If `ChatGPT-Account-ID` is supplied, it must match the local login too.

Refreshing the file takes effect on the next request. Old, expired, missing, or malformed credentials are rejected; CPA's original API-key authentication remains available. CodexGate does not verify JWT cryptographic signatures, perform network refreshes, or query remote revocation state. Its trust comes from exact matching against the trusted local file.

Supported routes include model listings, Responses HTTP/SSE and WebSocket handshakes, compact, alpha/search, and the applicable `/backend-api/codex` aliases. See `internal/auth/auth.go` for the method/path allowlist. Unrelated routes such as Chat Completions still require CPA's normal authentication, and management endpoints still require the management key.

Authentication applies to new HTTP requests and WebSocket handshakes. Already-open WebSockets are not forcibly disconnected on rotation/logout, and messages on an established connection are not re-authenticated individually.

## Verification and test records

The full platform matrix, commands, hashes, CI links, startup fixes, published-package checks, and manual client acceptance are in [VERIFICATION.md](VERIFICATION.md). The release's `verification.json` records its tested source commit, toolchains, installed libraries, and host-check results.

Windows development version `0.1.0-dev.3` has user-confirmed dot and new-chat acceptance after restart. The formal release has five-platform automated installer/host checks; those are not a separate manual Codex client or live-model test on every platform. These records belong here and in the verification document rather than the user-facing installation guide.

## Build and release

### Pinned inputs

`scripts/toolchains.json` pins Go 1.26.8, CPA 8.0.38 archives and SHA-256 values, LLVM-MinGW 20260922, Xcode 16.4, and manylinux2014 image digests. CI action revisions are pinned in `.github/workflows/release.yml`.

Python 3.11+ and Git are required for scripts. No pip packages are needed. The Go SDK remains pinned in `go.mod`; plugin-store imports are verification tooling, not a second plugin service.

### Native checks

```sh
python scripts/ci.py
```

Windows needs the pinned LLVM-MinGW compiler (or pass `--cc` to its `x86_64-w64-mingw32-gcc.exe`); macOS CI selects Xcode 16.4 and `CC=clang`. `--go` and `--core` accept explicit local paths. CPA is only launched with an isolated temporary config, synthetic login, empty upstream-account directory and random loopback port.

Linux release jobs run natively on amd64 and arm64 inside the pinned manylinux2014 images, including the real official CPA host smoke. `readelf` rejects libraries requiring GLIBC newer than 2.17. A modern Linux compiler may produce a newer GLIBC requirement; use the workflow container for a portable release.

Individual steps:

```sh
python scripts/build.py
python scripts/package_release.py
python scripts/smoke_cpa.py --core /path/to/cli-proxy-api --library /path/to/codexgate.so --expected-version 0.1.0
```

`build.py` checks the Go version and native architecture, runs tests/vet/race, then writes a version/platform-specific library. Existing outputs are never overwritten. `build.ps1` and `package.py` are retained development helpers; store releases use the Python pipeline.

### What the release gate proves

1. Native tests, vet and race checks pass.
2. ZIP layout and checksums follow the store specification.
3. The pinned CPA SDK's actual release installer downloads local in-memory fixtures, rejects a mismatched checksum, parses the ZIP, and installs the library to a new temporary directory. This check cannot reach external endpoints.
4. The official CPA 8.0.38 binary loads that installed library and exercises authentication, rotation/logout/expiry, enable/disable and WebSocket handshake with synthetic credentials.
5. Only redacted summaries and packages are uploaded as artifacts. Raw test configs, login files, and host logs stay out of releases.

This proves loading and authentication, not a full Codex UI or upstream-model session on every OS. Manual acceptance is recorded separately.

### Intel macOS callback boundary

The original direct Go exports reproduced `runtime.cgocallback` / `unknown caller pc` on the official Darwin amd64 CPA binary. The host and c-shared Go runtime were sharing a thread-local `g` slot. `cmd/codexgate/bridge.c` keeps the ABI entry/free operations in C and dispatches Intel macOS Go callbacks onto fresh C-owned threads, joined before returning. No host Go callback or Go-managed pointer is passed between runtimes, and no persistent worker/process is introduced. Other architectures keep direct calls.

This is a platform-specific boundary tested against the pinned host, not a general claim that arbitrary Go c-shared libraries can safely interact. Background: https://github.com/golang/go/issues/65050 . The real-host smoke includes concurrent authenticated requests as well as lifecycle calls.

### Publish

1. Update `VERSION`, changelog, README status, and the verification record.
2. Commit a clean tree; let all five main-branch jobs pass.
3. Push a matching `v<version>` tag. The tag workflow rebuilds and verifies all targets.
4. `assemble_release.py` requires matching version/source commit, clean sources, successful host/installer evidence, valid archives/checksums, and both Linux GLIBC reports.
5. Upload all assets to a draft release before publishing: `codexgate_<version>_<goos>_<goarch>.zip`, `checksums.txt`, and `verification.json`.
6. Validate published downloads, then submit `store/registry-entry.json` to the official store in a PR changing only `registry.json`. A pending PR is not a listing.

Never replace a published artifact silently; use a new version. Do not restart or modify a user's running CPA/Codex during CI or publishing.
