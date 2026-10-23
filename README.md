# CodexGate

让 Codex 保留原生 `openai` 接入，同时继续通过 CPA 使用模型和现有插件。

CodexGate 只补入口鉴权：CPA 原有 API Key 继续有效，Codex 发来的凭证只有与本机当前登录文件匹配、且未过期时才被接受。它不改模型请求，不处理回复，不接管 Reflow，也不启动独立服务。

## 当前状态

- 当前开发版：`0.1.0-dev.3`，Windows x64、CPA 8.0.38。
- 已通过单元测试、并发检查、`go vet` 和独立 CPA 的入口鉴权 / WebSocket 握手测试，详见 `docs/VERIFICATION.md`。
- CPA SDK 固定为 v8.0.38，原生 ABI 1 / JSON schema 6。
- 尚未安装到生产环境，也尚未完成安装插件后的真实 Codex / dot 验收。
- 本项目不自动修改 Codex 配置；正式安装与切换接入是单独一步。
- CPA 要求注册信息填写源码地址，开发版暂填拟建地址 `https://github.com/chatwithgod/cpa-plugin-codexgate`；仓库尚未创建或发布。

## 默认怎么工作

启用插件后，无须填写 token。它只读当前用户的登录文件：

1. 设置了 `CODEX_HOME`：读取其下的 `auth.json`。
2. 否则读取用户目录下的 `.codex/auth.json`。
3. 非默认位置可在插件设置的 `auth-file` 中填写绝对路径。

每次鉴权重新读取文件，不复制或持久化 token。刷新登录后，新 token 直接生效；旧 token 不再接受。退出登录、文件不可读、文件损坏或 token 到期时拒绝放行，原有 API Key 不受影响。

默认支持模型列表、Responses HTTP/SSE、Responses WebSocket 握手、压缩和 Alpha Search 入口。不会用 Codex 凭证放行 CPA 管理接口。

上述鉴权作用于新的 HTTP 请求和 WebSocket 握手，不会主动踢掉已建立的 WebSocket 连接。禁用插件也不会替你改回 Codex 配置；回退时需要同时恢复原来的 CPA 接入方式。

## 使用边界

- CPA 应继续只监听 `127.0.0.1`，保留至少一个原有 API Key。不要为了本插件清空密钥列表。
- 第一版支持文件形式的 Codex 登录，不读取系统凭据库。
- 可信来源是本机登录文件，不是“长得像 JWT”的请求。插件不进行云端撤销查询；它检查精确匹配和 token 时间字段。
- CPA 的插件鉴权接口不提供请求方 IP；本机监听限制由 CPA 本身执行，不靠客户端可伪造的转发头判断。
- 插件既不修复也不绕过账号自身的 dot 开通条件。
- 状态接口受 CPA 管理鉴权保护：`GET /v0/management/codexgate/status`，不返回 token 或账号信息。

## 构建与隔离测试

需要 Go 1.26 和支持 cgo 的 Windows x64 C 编译器。

```powershell
go mod download
pwsh -File ./scripts/build.ps1 -Go '你的 go.exe 路径' -CC '你的 gcc.exe 路径' -Version '0.1.0-dev.3'
python ./scripts/smoke_cpa.py --core '你的 cli-proxy-api.exe 路径' --dll './build/0.1.0-dev.3/codexgate-v0.1.0-dev.3.dll'
```

构建脚本拒绝覆盖已有 DLL。隔离测试使用随机本机端口、合成登录文件和空的上游账号目录，不调用真实模型，不加载生产插件，不修改生产配置。测试完自动停止隔离进程，报告保存在 `.runs`。

## 来源

原生插件 ABI 入口参考 CLIProxyAPI 的 MIT 许可示例；具体来源见 `THIRD_PARTY_NOTICES.md`。这不是 OpenAI 或 CPA 官方插件。
