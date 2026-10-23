# 验证记录

日期：2026-10-23（Asia/Shanghai）。

## 已通过

开发版 `0.1.0-dev.3`，Windows amd64，Go 1.26.8，CPA 8.0.38（提交 `0db19025`）。

- `go test ./...`：通过。
- `go vet ./...`：通过。
- `go test -race ./internal/...`：通过。
- 原生 DLL 被 CPA 注册，插件有效启用，使用非独占入口鉴权。
- 不配置 `auth-file`，自动读取隔离 `CODEX_HOME/auth.json`。
- 没有凭证、其他 JWT、过期 token、已退出的登录、损坏的登录文件均返回 401。
- 原有 CPA API Key 仍返回 200，包括退出 Codex 和禁用插件之后。
- 当前本机登录 token 返回 200；替换登录文件后，新 token 生效，旧 token 立即返回 401。
- Responses WebSocket：缺少凭证时 401，有效凭证时握手 101，然后立即关闭，没有发送模型请求。
- 管理状态接口仍需要管理密钥，不能用 Codex token 访问。
- 通过 CPA 管理 API 禁用插件后，Codex token 被拒绝；重新启用后恢复。
- 测试日志不包含合成 token、测试 API Key 或测试管理密钥。
- 生产 Codex、ECPA、CPA 配置和真实登录文件的校验值未变化；隔离进程已停止。

完整隔离报告：`.runs/20261023-220825-bf6d42/summary.json`。

DLL：`build/0.1.0-dev.3/codexgate-v0.1.0-dev.3.dll`。

SHA-256：`29f756e03f2f41cdd3bc5c2de723962047f684713e6cc011918a39ab2f506df3`。

## 测试中修正的接入问题

- `dev.1`：CPA 要求源码仓库字段非空；已在后续版本补齐拟建地址，未创建或发布仓库。
- `dev.2`：管理注册传入的是 `/v0/management`，不是包含插件名的路径；已在 `dev.3` 修正。
- 隔离脚本曾原子替换整个配置文件，但未观察到本机 CPA 执行热重载。启停验证已改用 ECPA 同类的官方管理操作 `PATCH /v0/management/plugins/codexgate/enabled`。没有为此修改 CPA 核心。

旧版本 DLL 未覆盖。

## Windows 真实安装与客户端验收（开发版）

2026-10-23，用户授权安装 `0.1.0-dev.3` 到其 CPA 8.0.38，保留原 API Key 和其他插件。安装后验证原 API Key 与本机 Codex 登录凭证均可读取模型列表，无凭证仍返回 401。

随后用户开启 ECPA“使用 openai 提供商标识”并重启 Codex，明确反馈：**dot 通信成功，新聊天通信成功**。这是开发版的人工客户端验收，不是对所有平台、全部模型或所有传输路径的验收。

此前的隔离测试只用合成凭证和空上游账号目录，没有模型调用。正式版自动测试保持同样的隔离方式，不更新或重启上述生产实例。

## 正式版 0.1.0 平台验证

| 平台 | 构建/单元/vet/race | CPA 商店安装器 | 官方 CPA 8.0.38 宿主鉴权与 WS 握手 | 人工 Codex 客户端 |
| --- | --- | --- | --- | --- |
| Windows amd64 | 待 CI | 待 CI | 待 CI | 开发版通过；正式版未另测 |
| Linux amd64 | 待 CI | 待 CI | 待 CI | 未测 |
| Linux arm64 | 待 CI | 待 CI | 待 CI | 未测 |
| macOS amd64 | 待 CI | 待 CI | 待 CI | 未测 |
| macOS arm64 | 待 CI | 待 CI | 待 CI | 未测 |

正式发布工作流要求五个平台的自动验证全部通过。正式 Release 的 `verification.json` 记录具体提交、工具链、动态库校验值、安装器与宿主检查结果。系统凭据库不在支持范围内，其他 CPA 版本未据此自动视为兼容。

鉴权发生在新的 HTTP 请求和 WebSocket 握手。已经建立的 WebSocket 连接不会由本插件主动断开；不能把握手测试理解成消息逐条重新鉴权。
