# CodexGate v0.1.1

## 中文

CodexGate 解决接入 CPA 后 dot 入口消失的问题，配合 ECPA 的“使用 openai 提供商标识”功能使用。

**本次更新：**
- **修复内置生图 401 问题**：补上 `POST /v1/images/generations` 和 `POST /v1/images/edits` 的本机登录鉴权，解决这两个接口遗漏造成的 HTTP 401。
- **升级后无需额外配置**：原有 CPA API Key、其他路由及鉴权检查保持不变，安装并启用插件即可。

**平台与验证：**
- Windows amd64、Linux amd64/arm64、macOS amd64/arm64 五个平台均通过原生 CI 构建、商店安装器及官方 CPA 宿主测试。
- Windows 开发版已实测一次真实内置生图成功。详细记录见[验证文档](https://github.com/chatwithgod/cpa-plugin-codexgate/blob/main/docs/VERIFICATION.md)。
- 安装包为 `codexgate_0.1.1_<goos>_<goarch>.zip`，附带 `checksums.txt` 和 `verification.json`。安装步骤见[中文指南](https://github.com/chatwithgod/cpa-plugin-codexgate/blob/v0.1.1/README.zh-CN.md)。

---

## English

CodexGate fixes the missing dot entry in Codex after connecting to CPA, working with ECPA's “Use openai provider identifier” feature.

**What's new:**
- **Fixed built-in image-tool authentication (HTTP 401):** Added local login support for `POST /v1/images/generations` and `POST /v1/images/edits`, fixing the 401 caused by these missing routes.
- **No extra setup after upgrading:** Existing CPA API keys, other routes, and authentication checks remain unchanged. Install and enable the plugin as usual.

**Platforms and verification:**
- Windows amd64, Linux amd64/arm64, and macOS amd64/arm64 passed native CI builds, store-installer checks, and official CPA host tests.
- One real built-in image-generation call succeeded on the Windows development build. See the [verification record](https://github.com/chatwithgod/cpa-plugin-codexgate/blob/main/docs/VERIFICATION.md) for details.
- Packages are named `codexgate_0.1.1_<goos>_<goarch>.zip`, accompanied by `checksums.txt` and `verification.json`. Follow the [installation guide](https://github.com/chatwithgod/cpa-plugin-codexgate/blob/v0.1.1/README.md).
