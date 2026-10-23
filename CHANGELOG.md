# Changelog

## 0.1.1 — 2026-10-24

- Fix built-in Codex image-tool authentication through CPA by accepting the current local login on `POST /v1/images/generations` and `POST /v1/images/edits`.
- Keep exact credential matching, expiry/account checks, original CPA API keys, and management protection unchanged. Add route-boundary and isolated Images API authentication checks without model inference.

## 0.1.0

- First store-format release of CodexGate.
- Keep CPA API-key authentication and accept the current local Codex file-login token through a non-exclusive frontend provider.
- Cover Codex model listings, Responses HTTP/SSE and WebSocket handshakes, compact and alpha/search.
- Add five native build targets, pinned toolchains, Linux GLIBC 2.17 builds, CPA release-installer checks, and isolated official-host tests.
- Isolate Intel macOS Go callbacks on joined C-owned threads to avoid the reproduced host/plugin TLS collision; verify concurrent native callbacks on every target.
- Add English/Chinese setup and rollback instructions. Record the user-confirmed Windows `0.1.0-dev.3` dot/new-chat acceptance separately from release CI.

## 0.1.0-dev.3 — 2026-10-23

- Correct CPA metadata and management-route registration.
- Pass Windows unit, race, isolated host lifecycle and WebSocket handshake tests.
- Install locally; user confirms dot and new chats work after enabling ECPA's openai provider identity and restarting Codex.

Earlier development artifacts are preserved; the development source baseline is tagged `v0.1.0-dev.3`.
