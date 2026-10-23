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

## 仍未验证

- 安装到生产 CPA 后，真实 Codex 发出的 token 是否与登录文件一致。
- 真实 Codex 新聊天、dot 回复，以及实际 HTTP/SSE 或 WebSocket 模型流。
- 与用户当前其他插件同时启用的真实任务。
- Linux、macOS、系统凭据库登录模式及其他 CPA 版本。

本轮全部使用合成凭证和空上游账号目录，没有模型额度消耗。生产安装和实际模型验证需要另外授权。

鉴权发生在新的 HTTP 请求和 WebSocket 握手。已经建立的 WebSocket 连接不会由本插件主动断开；不能把握手测试理解成消息逐条重新鉴权。
