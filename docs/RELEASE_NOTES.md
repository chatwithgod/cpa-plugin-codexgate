# CodexGate v0.1.0

## English

CodexGate adds local Codex file-login authentication to CPA without replacing existing API keys. Keep Codex's `openai` provider identity while sending model requests through CPA; the tested Windows setup retains dot and new-chat communication after restart.

- Exact current-token matching with expiry checks; no extra daemon or upstream account management.
- Model listings, Responses HTTP/SSE and WebSocket handshakes, compact and alpha/search.
- Windows amd64, Linux amd64/arm64 (GLIBC 2.17+), macOS amd64/arm64.
- Pinned CPA 8.0.38 / ABI 1 / schema 6 baseline and Go 1.26.8.
- Five native build, store-installer and official CPA host-smoke jobs gate publication. See `verification.json` for this release's results.

Real Codex UI acceptance has been confirmed on Windows with development version `0.1.0-dev.3`; automated host checks do not establish full client/model-stream acceptance on every platform. Only file-based ChatGPT login is supported, not system credential vaults.

Install and enable the plugin before enabling ECPA's “使用 openai 提供商标识” option. Keep the original API keys. To roll back, restore the previous provider configuration and restart Codex before disabling the plugin. Store inclusion is tracked separately from this release.

## 中文

CodexGate 让 CPA 接受本机 Codex 的文件登录凭证，同时保留原有 API Key。Codex 可以继续使用 `openai` 提供商标识，模型请求仍走 CPA；已验证的 Windows 环境中，重启后 dot 和新聊天都能正常通信。

- 当前 Token 精确匹配及有效期检查，不增加独立后台服务或账号管理。
- 支持模型列表、Responses HTTP/SSE、WebSocket 握手、compact 和 alpha/search。
- 提供 Windows amd64、Linux amd64/arm64（GLIBC 2.17+）、macOS amd64/arm64 的安装包。
- 基线为 CPA 8.0.38、ABI 1/schema 6、Go 1.26.8。
- 五个平台的构建、商店安装器及真实 CPA 宿主冒烟全部通过后才发布；结果见 `verification.json`。

Windows 开发版 `0.1.0-dev.3` 已有用户确认的客户端验收；自动宿主测试不代表每个平台都完成了真实客户端/模型流测试。目前仅支持文件式 ChatGPT 登录，不读取系统凭据库。

请先安装启用插件，再开启 ECPA“使用 openai 提供商标识”，保留原 API Key。回退时先恢复提供商配置并重启 Codex，再禁用插件。商店收录状态与此 Release 分开跟踪。
