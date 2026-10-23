# CodexGate

[简体中文](README.zh-CN.md) · [Releases](https://github.com/chatwithgod/cpa-plugin-codexgate/releases) · [Verification](docs/VERIFICATION.md)

> **Release status:** v0.1.0 is available on [GitHub Releases](https://github.com/chatwithgod/cpa-plugin-codexgate/releases/tag/v0.1.0). CPA Plugin Store [PR #284](https://github.com/router-for-me/CLIProxyAPI-Plugins-Store/pull/284) is awaiting review, so use the manual installation instructions below until it is listed. All five target platforms passed native CI and post-release package verification.

CodexGate is an inbound authentication compatibility plugin for CLIProxyAPI (CPA). It lets Codex keep `model_provider="openai"` while model requests still go through your local CPA instance. This addresses the provider-identity restriction that hid the **dot** entry in our tested desktop setup; it does not grant account features that are otherwise unavailable.

Existing CPA API keys keep working. No extra daemon, model routing, request rewriting, reasoning continuation, or upstream account management is added. This is an independent MIT-licensed project, not an official OpenAI or CPA plugin.

## How it works

For each supported request, CodexGate reads the current login file and accepts only a Bearer token that **exactly matches** its `access_token` and has valid time claims. A supplied `ChatGPT-Account-ID` must also match. It does not verify JWT cryptographic signatures or contact an external service: the trusted local file is the source of identity.

The lookup order is:

1. An explicit absolute `auth-file` path in the plugin settings.
2. The CPA process's `CODEX_HOME/auth.json`.
3. The CPA process user's `~/.codex/auth.json`.

Same-user local installations with a file-based ChatGPT login normally need no extra plugin configuration. Refreshing the login takes effect on the next request. Old, expired, missing, or malformed credentials are rejected; CPA's original API-key path remains available.

Only file-based `auth_mode=chatgpt` logins are supported; system keychains/keyrings are not read. A service account or container needs access to the correct user's file. For containers, read-only-mount the directory containing `auth.json`, not just the single file, so atomic replacement remains visible; set `auth-file` to its absolute path inside the container. Keep CPA local and the login directory private.

## Install and enable

Install from the CPA plugin store once listed; before then, use the matching platform ZIP from [Releases](https://github.com/chatwithgod/cpa-plugin-codexgate/releases).

1. Keep at least one existing CPA API key and bind CPA to `127.0.0.1`. Confirm that Codex already has a file-based ChatGPT login.
2. Put the ZIP's library in CPA's plugin directory under `<goos>/<goarch>`: `codexgate.dll`, `codexgate.so`, or `codexgate.dylib`. macOS uses `darwin`; Apple Silicon uses `arm64`. Do not overwrite a loaded library. For manual replacement, stop CPA during a maintenance window first.
3. Enable the plugin in the management UI. If editing YAML, **merge** this into the existing settings rather than replacing other plugins:

   ```yaml
   plugins:
     enabled: true
     configs:
       codexgate:
         enabled: true
         # auth-file: /absolute/path/to/auth.json
   ```

4. Check `GET /v0/management/codexgate/status` with the CPA management key. It should report `enabled: true` and `login.ready: true`. This protected endpoint reports no token or account ID.
5. **ECPA users:** enable “使用 openai 提供商标识” (“Use openai provider identity”). **Plain CPA users:** back up Codex's configuration and set its top-level fields:

   ```toml
   model_provider = "openai"
   openai_base_url = "http://127.0.0.1:8317/v1"
   ```

   Use your actual CPA address. Do not place these fields in `[model_providers.openai]`, and do not change `auth.json`.
6. Wait for active Codex work to finish, restart Codex, and check dot plus a new chat. Confirm requests appear in CPA usage records.

## Scope and rollback

Supported routes are model listings, Responses HTTP/SSE and WebSocket handshakes, compact, alpha/search, and the corresponding `/backend-api/codex` aliases. Unrelated routes such as Chat Completions still require CPA's normal authentication. Management authentication is not replaced.

There is no remote token refresh or revocation lookup. Authentication applies to new HTTP requests and WebSocket handshakes; already-open WebSockets are not forcibly disconnected or authenticated again on each message.

To roll back, **first** disable ECPA's provider-identity option or restore your previous CPA provider settings, then restart Codex when idle. **Afterward**, disable or uninstall CodexGate. Keep the original API keys. Do not copy login tokens into CPA's static key list.

## Compatibility

Verified native targets: Windows amd64; Linux amd64/arm64 with GLIBC 2.17+; macOS amd64/arm64. The baseline is CPA **8.0.38**, ABI 1/schema 6, built with Go **1.26.8**. Linux CPA `no-plugin` builds and musl-only environments are not supported.

Windows development version `0.1.0-dev.3` passed isolated authentication tests and user-confirmed dot/new-chat use after restart on 2026-10-23. Formal multi-platform verification is tracked separately in [VERIFICATION.md](docs/VERIFICATION.md), Actions, and the release's `verification.json`.

See [development and release instructions](docs/DEVELOPMENT.md) and [third-party notices](THIRD_PARTY_NOTICES.md).
